package providers

import (
	"context"
	"encoding/json"
	"time"
)

type Indicator string

const (
	IndicatorOperational Indicator = "operational"
	IndicatorMinor       Indicator = "minor"
	IndicatorMajor       Indicator = "major"
	IndicatorCritical    Indicator = "critical"
	IndicatorMaintenance Indicator = "maintenance"
	IndicatorUnknown     Indicator = "unknown"
)

func (i Indicator) Rank() int {
	switch i {
	case IndicatorOperational:
		return 0
	case IndicatorMaintenance:
		return 1
	case IndicatorMinor:
		return 2
	case IndicatorMajor:
		return 3
	case IndicatorCritical:
		return 4
	}
	return -1
}

func WorstIndicator(xs []Indicator) Indicator {
	worst := IndicatorOperational
	for _, x := range xs {
		if x.Rank() > worst.Rank() {
			worst = x
		}
	}
	return worst
}

type Kind string

const (
	KindStatuspageIO Kind = "statuspage_io"
	KindAuth0        Kind = "auth0"
	KindSlackStatus  Kind = "slack_status"
)

type Config struct {
	ID        string          `json:"id"`
	Name      string          `json:"name"`
	Kind      Kind            `json:"kind"`
	Params    json.RawMessage `json:"params"`
	SortOrder int             `json:"sort_order"`
}

type Component struct {
	Name   string    `json:"name"`
	Status Indicator `json:"status"`
}

type Incident struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Status    string    `json:"status"`
	Impact    Indicator `json:"impact"`
	URL       string    `json:"url"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Status struct {
	Indicator   Indicator   `json:"indicator"`
	Description string      `json:"description"`
	Components  []Component `json:"components"`
	Incidents   []Incident  `json:"incidents"`
	FetchedAt   time.Time   `json:"fetched_at"`
	Err         string      `json:"err,omitempty"`
}

type Provider interface {
	Config() Config
	Fetch(ctx context.Context) (Status, error)
}

type ParamField struct {
	Name        string `json:"name"`
	Label       string `json:"label"`
	Type        string `json:"type"`
	Placeholder string `json:"placeholder,omitempty"`
	Required    bool   `json:"required"`
	Help        string `json:"help,omitempty"`
}

type Factory interface {
	Kind() Kind
	Label() string
	Fields() []ParamField
	Build(cfg Config) (Provider, error)
	Validate(ctx context.Context, cfg Config) error
}
