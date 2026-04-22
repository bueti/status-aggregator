package providers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Auth0's status page (https://status.auth0.com/) is a Next.js app that SSRs
// its data and embeds it in a <script id="__NEXT_DATA__"> tag. The underlying
// /_next/data/<buildId>/index.json endpoint returns the same shape, but the
// buildId rotates on every deploy, so scraping the HTML is more stable.

const auth0StatusURL = "https://status.auth0.com/"

func init() {
	Register(&auth0Factory{})
}

type auth0Factory struct{}

func (f *auth0Factory) Kind() Kind           { return KindAuth0 }
func (f *auth0Factory) Label() string        { return "Auth0" }
func (f *auth0Factory) Fields() []ParamField { return []ParamField{} }

func (f *auth0Factory) Build(cfg Config) (Provider, error) {
	return &auth0Provider{cfg: cfg, client: sharedHTTP}, nil
}

type auth0Provider struct {
	cfg    Config
	client *http.Client
}

func (p *auth0Provider) Config() Config { return p.cfg }

func (p *auth0Provider) Fetch(ctx context.Context) (Status, error) {
	data, err := fetchAuth0(ctx, p.client)
	if err != nil {
		return Status{}, err
	}
	return auth0ToStatus(data), nil
}

type auth0Incident struct {
	Status    string     `json:"status"`
	Name      string     `json:"name"`
	ID        string     `json:"id"`
	UpdatedAt time.Time  `json:"updated_at"`
	Resolved  *time.Time `json:"resolved_at"`
	Impact    string     `json:"impact"`
	IsPrivate bool       `json:"isPrivate"`
}

type auth0Region struct {
	Region      string `json:"region"`
	Environment string `json:"environment"`
	Response    struct {
		Incidents []auth0Incident `json:"incidents"`
	} `json:"response"`
}

type auth0Payload struct {
	Props struct {
		PageProps struct {
			ActiveIncidents []auth0Region `json:"activeIncidents"`
		} `json:"pageProps"`
	} `json:"props"`
}

// Plain byte-scan instead of a regex: the body can be several MB and a
// backtracking `.*?` match scales badly on adversarial input.
var (
	auth0NextDataOpen  = []byte(`<script id="__NEXT_DATA__" type="application/json">`)
	auth0NextDataClose = []byte(`</script>`)
)

func extractAuth0NextData(body []byte) ([]byte, error) {
	_, after, ok := bytes.Cut(body, auth0NextDataOpen)
	if !ok {
		return nil, fmt.Errorf("auth0: __NEXT_DATA__ not found on page (layout may have changed)")
	}
	payload, _, ok := bytes.Cut(after, auth0NextDataClose)
	if !ok {
		return nil, fmt.Errorf("auth0: __NEXT_DATA__ close tag not found")
	}
	return payload, nil
}

func fetchAuth0(ctx context.Context, client *http.Client) (*auth0Payload, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, auth0StatusURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "status-aggregator/0.1")
	req.Header.Set("Accept", "text/html")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("auth0 status: HTTP %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return nil, err
	}

	raw, err := extractAuth0NextData(body)
	if err != nil {
		return nil, err
	}
	var p auth0Payload
	if err := json.Unmarshal(raw, &p); err != nil {
		return nil, fmt.Errorf("auth0: decode __NEXT_DATA__: %w", err)
	}
	if len(p.Props.PageProps.ActiveIncidents) == 0 {
		return nil, fmt.Errorf("auth0: activeIncidents is empty (unexpected schema)")
	}
	return &p, nil
}

func auth0ToStatus(p *auth0Payload) Status {
	out := Status{
		Components: make([]Component, 0, len(p.Props.PageProps.ActiveIncidents)),
		Incidents:  make([]Incident, 0),
		FetchedAt:  time.Now().UTC(),
	}
	indicators := make([]Indicator, 0, len(p.Props.PageProps.ActiveIncidents))

	for _, r := range p.Props.PageProps.ActiveIncidents {
		regionIndicator := IndicatorOperational
		for _, inc := range r.Response.Incidents {
			if inc.IsPrivate || inc.Resolved != nil {
				continue
			}
			impact := auth0Impact(inc.Impact, inc.Status)
			if impact.Rank() > regionIndicator.Rank() {
				regionIndicator = impact
			}
			// Skip the "All Systems Operational" placeholder rows.
			if strings.TrimSpace(inc.ID) == "" || strings.EqualFold(inc.Status, "operational") {
				continue
			}
			out.Incidents = append(out.Incidents, Incident{
				ID:        inc.ID,
				Name:      fmt.Sprintf("[%s] %s", r.Region, inc.Name),
				Status:    inc.Status,
				Impact:    impact,
				URL:       auth0StatusURL,
				UpdatedAt: inc.UpdatedAt,
			})
		}

		label := r.Region
		if r.Environment != "" && !strings.EqualFold(r.Environment, "production") {
			label = fmt.Sprintf("%s (%s)", r.Region, r.Environment)
		}
		out.Components = append(out.Components, Component{
			Name:   label,
			Status: regionIndicator,
		})
		indicators = append(indicators, regionIndicator)
	}

	out.Indicator = WorstIndicator(indicators)
	switch out.Indicator {
	case IndicatorOperational:
		out.Description = "All Systems Operational"
	case IndicatorMaintenance:
		out.Description = "Scheduled Maintenance"
	default:
		out.Description = "Some environments report issues"
	}
	return out
}

func auth0Impact(impact, status string) Indicator {
	switch strings.ToLower(impact) {
	case "minor":
		return IndicatorMinor
	case "major":
		return IndicatorMajor
	case "critical":
		return IndicatorCritical
	case "maintenance":
		return IndicatorMaintenance
	case "none", "":
		switch strings.ToLower(status) {
		case "operational":
			return IndicatorOperational
		case "maintenance", "scheduled":
			return IndicatorMaintenance
		case "investigating", "identified", "monitoring":
			return IndicatorMinor
		}
		return IndicatorOperational
	}
	return IndicatorUnknown
}
