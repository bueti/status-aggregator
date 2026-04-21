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

// Slack's status page (https://slack-status.com/) exposes a versioned JSON
// endpoint that returns the current status plus an active_incidents list.
//   GET /api/v2.0.0/current
//   GET /api/v2.0.0/history

const slackStatusURL = "https://slack-status.com/api/v2.0.0/current"

func init() {
	Register(&slackStatusFactory{})
}

type slackStatusFactory struct{}

func (f *slackStatusFactory) Kind() Kind           { return KindSlackStatus }
func (f *slackStatusFactory) Label() string        { return "Slack" }
func (f *slackStatusFactory) Fields() []ParamField { return []ParamField{} }

func (f *slackStatusFactory) Build(cfg Config) (Provider, error) {
	return &slackStatusProvider{cfg: cfg, client: sharedHTTP}, nil
}

func (f *slackStatusFactory) Validate(ctx context.Context, cfg Config) error {
	_, err := fetchSlackStatus(ctx, sharedHTTP)
	return err
}

type slackStatusProvider struct {
	cfg    Config
	client *http.Client
}

func (p *slackStatusProvider) Config() Config { return p.cfg }

func (p *slackStatusProvider) Fetch(ctx context.Context) (Status, error) {
	data, err := fetchSlackStatus(ctx, p.client)
	if err != nil {
		return Status{}, err
	}
	return slackStatusToStatus(data), nil
}

type slackIncident struct {
	ID          int       `json:"id"`
	Title       string    `json:"title"`
	Type        string    `json:"type"`
	Status      string    `json:"status"`
	URL         string    `json:"url"`
	DateUpdated time.Time `json:"date_updated"`
	Services    []string  `json:"services"`
}

type slackCurrent struct {
	Status          string          `json:"status"`
	DateUpdated     time.Time       `json:"date_updated"`
	ActiveIncidents []slackIncident `json:"active_incidents"`
}

func fetchSlackStatus(ctx context.Context, client *http.Client) (*slackCurrent, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, slackStatusURL, nil)
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
		return nil, fmt.Errorf("slack-status.com: HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var s slackCurrent
	if err := json.NewDecoder(resp.Body).Decode(&s); err != nil {
		return nil, fmt.Errorf("slack-status.com: decode: %w", err)
	}
	return &s, nil
}

func slackStatusToStatus(s *slackCurrent) Status {
	out := Status{
		Components: []Component{},
		Incidents:  make([]Incident, 0, len(s.ActiveIncidents)),
		FetchedAt:  time.Now().UTC(),
	}
	indicators := make([]Indicator, 0, len(s.ActiveIncidents))
	for _, inc := range s.ActiveIncidents {
		impact := slackIncidentImpact(inc.Type)
		indicators = append(indicators, impact)
		name := inc.Title
		if len(inc.Services) > 0 {
			name = fmt.Sprintf("[%s] %s", strings.Join(inc.Services, ", "), inc.Title)
		}
		out.Incidents = append(out.Incidents, Incident{
			ID:        fmt.Sprintf("%d", inc.ID),
			Name:      name,
			Status:    inc.Status,
			Impact:    impact,
			URL:       inc.URL,
			UpdatedAt: inc.DateUpdated,
		})
	}
	out.Indicator = WorstIndicator(indicators)
	switch out.Indicator {
	case IndicatorOperational:
		out.Description = "All systems normal"
	case IndicatorCritical:
		out.Description = "Ongoing outage"
	default:
		out.Description = "Active incidents"
	}
	return out
}

// slack-status.com "type" values observed in the wild: "outage", "incident",
// "notice". Ordered here from worst to mildest.
func slackIncidentImpact(t string) Indicator {
	switch strings.ToLower(t) {
	case "outage":
		return IndicatorCritical
	case "incident":
		return IndicatorMajor
	case "notice":
		return IndicatorMinor
	}
	return IndicatorUnknown
}
