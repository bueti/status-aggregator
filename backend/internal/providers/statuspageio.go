package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

func init() {
	Register(&statuspageIOFactory{})
}

type statuspageIOFactory struct{}

func (f *statuspageIOFactory) Kind() Kind    { return KindStatuspageIO }
func (f *statuspageIOFactory) Label() string { return "Statuspage.io" }

func (f *statuspageIOFactory) Fields() []ParamField {
	return []ParamField{
		{
			Name:        "base_url",
			Label:       "Base URL",
			Type:        "url",
			Placeholder: "https://www.githubstatus.com",
			Required:    true,
			Help:        "Root of the status page. The adapter appends /api/v2/summary.json.",
		},
	}
}

type statuspageIOParams struct {
	BaseURL string `json:"base_url"`
}

func (p statuspageIOParams) summaryURL() (string, error) {
	u, err := parseHTTPURL("base_url", p.BaseURL)
	if err != nil {
		return "", err
	}
	u.Path = strings.TrimRight(u.Path, "/") + "/api/v2/summary.json"
	return u.String(), nil
}

func (f *statuspageIOFactory) parse(cfg Config) (statuspageIOParams, error) {
	var p statuspageIOParams
	if len(cfg.Params) == 0 {
		return p, fmt.Errorf("params are required")
	}
	if err := json.Unmarshal(cfg.Params, &p); err != nil {
		return p, fmt.Errorf("invalid params: %w", err)
	}
	u, err := parseHTTPURL("base_url", p.BaseURL)
	if err != nil {
		return p, err
	}
	p.BaseURL = u.String()
	return p, nil
}

func (f *statuspageIOFactory) Build(cfg Config) (Provider, error) {
	p, err := f.parse(cfg)
	if err != nil {
		return nil, err
	}
	raw, err := json.Marshal(p)
	if err != nil {
		return nil, fmt.Errorf("marshal params: %w", err)
	}
	cfg.Params = raw
	return &statuspageIOProvider{cfg: cfg, params: p, client: sharedHTTP}, nil
}

type statuspageIOProvider struct {
	cfg    Config
	params statuspageIOParams
	client *http.Client
}

func (p *statuspageIOProvider) Config() Config { return p.cfg }

func (p *statuspageIOProvider) Fetch(ctx context.Context) (Status, error) {
	summary, err := fetchSummary(ctx, p.client, p.params)
	if err != nil {
		return Status{}, err
	}
	return summaryToStatus(summary), nil
}

// --- wire format ---

type statuspageSummary struct {
	Page struct {
		Name string `json:"name"`
	} `json:"page"`
	Status struct {
		Indicator   string `json:"indicator"`
		Description string `json:"description"`
	} `json:"status"`
	Components []struct {
		Name   string `json:"name"`
		Status string `json:"status"`
	} `json:"components"`
	Incidents []struct {
		ID        string     `json:"id"`
		Name      string     `json:"name"`
		Status    string     `json:"status"`
		Impact    string     `json:"impact"`
		Shortlink string     `json:"shortlink"`
		UpdatedAt time.Time  `json:"updated_at"`
		Resolved  *time.Time `json:"resolved_at"`
	} `json:"incidents"`
}

var sharedHTTP = &http.Client{
	Timeout: 10 * time.Second,
}

func fetchSummary(ctx context.Context, client *http.Client, p statuspageIOParams) (*statuspageSummary, error) {
	u, err := p.summaryURL()
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "status-aggregator/0.1")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, fmt.Errorf("statuspage.io %s: status %d: %s", u, resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var s statuspageSummary
	if err := json.NewDecoder(resp.Body).Decode(&s); err != nil {
		return nil, fmt.Errorf("decode summary: %w", err)
	}
	return &s, nil
}

func summaryToStatus(s *statuspageSummary) Status {
	out := Status{
		Indicator:   mapIndicator(s.Status.Indicator),
		Description: s.Status.Description,
		Components:  make([]Component, 0, len(s.Components)),
		Incidents:   make([]Incident, 0),
		FetchedAt:   time.Now().UTC(),
	}
	for _, c := range s.Components {
		out.Components = append(out.Components, Component{
			Name:   c.Name,
			Status: mapComponentStatus(c.Status),
		})
	}
	for _, inc := range s.Incidents {
		if inc.Resolved != nil {
			continue
		}
		out.Incidents = append(out.Incidents, Incident{
			ID:        inc.ID,
			Name:      inc.Name,
			Status:    inc.Status,
			Impact:    mapIndicator(inc.Impact),
			URL:       inc.Shortlink,
			UpdatedAt: inc.UpdatedAt,
		})
	}
	return out
}

func mapIndicator(s string) Indicator {
	switch strings.ToLower(s) {
	case "", "none":
		return IndicatorOperational
	case "minor":
		return IndicatorMinor
	case "major":
		return IndicatorMajor
	case "critical":
		return IndicatorCritical
	case "maintenance":
		return IndicatorMaintenance
	}
	return IndicatorUnknown
}

func mapComponentStatus(s string) Indicator {
	switch strings.ToLower(s) {
	case "operational":
		return IndicatorOperational
	case "degraded_performance":
		return IndicatorMinor
	case "partial_outage":
		return IndicatorMajor
	case "major_outage":
		return IndicatorCritical
	case "under_maintenance":
		return IndicatorMaintenance
	}
	return IndicatorUnknown
}
