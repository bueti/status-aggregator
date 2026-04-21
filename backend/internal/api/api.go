package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"

	"github.com/bbu/status-aggregator/backend/internal/aggregator"
	"github.com/bbu/status-aggregator/backend/internal/providers"
	"github.com/bbu/status-aggregator/backend/internal/store"
)

type Server struct {
	Agg        *aggregator.Aggregator
	Store      *store.Store
	AdminToken string
}

func (s *Server) Register(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "healthz",
		Method:      http.MethodGet,
		Path:        "/api/healthz",
		Summary:     "Liveness probe",
		Tags:        []string{"meta"},
	}, s.healthz)

	huma.Register(api, huma.Operation{
		OperationID: "get-overview",
		Method:      http.MethodGet,
		Path:        "/api/overview",
		Summary:     "Aggregated overview of all providers",
		Tags:        []string{"status"},
	}, s.overview)

	huma.Register(api, huma.Operation{
		OperationID: "get-provider",
		Method:      http.MethodGet,
		Path:        "/api/providers/{id}",
		Summary:     "Detailed status for a single provider",
		Tags:        []string{"status"},
	}, s.getProvider)

	huma.Register(api, huma.Operation{
		OperationID: "list-feed-kinds",
		Method:      http.MethodGet,
		Path:        "/api/feed-kinds",
		Summary:     "List available feed kinds and their parameter schemas",
		Tags:        []string{"meta"},
	}, s.feedKinds)

	huma.Register(api, huma.Operation{
		OperationID: "list-providers",
		Method:      http.MethodGet,
		Path:        "/api/providers",
		Summary:     "List configured providers (config + current status)",
		Tags:        []string{"providers"},
	}, s.listProviders)

	huma.Register(api, huma.Operation{
		OperationID: "create-provider",
		Method:      http.MethodPost,
		Path:        "/api/providers",
		Summary:     "Add a new provider (admin)",
		Tags:        []string{"providers"},
	}, s.createProvider)

	huma.Register(api, huma.Operation{
		OperationID: "update-provider",
		Method:      http.MethodPut,
		Path:        "/api/providers/{id}",
		Summary:     "Update a provider (admin)",
		Tags:        []string{"providers"},
	}, s.updateProvider)

	huma.Register(api, huma.Operation{
		OperationID: "delete-provider",
		Method:      http.MethodDelete,
		Path:        "/api/providers/{id}",
		Summary:     "Delete a provider (admin)",
		Tags:        []string{"providers"},
	}, s.deleteProvider)

	huma.Register(api, huma.Operation{
		OperationID: "validate-provider",
		Method:      http.MethodPost,
		Path:        "/api/providers/validate",
		Summary:     "Dry-run validate a provider config (admin)",
		Tags:        []string{"providers"},
	}, s.validateProvider)
}

// --- shared shapes ---

type ProviderSummary struct {
	ID              string              `json:"id"`
	Name            string              `json:"name"`
	Kind            providers.Kind      `json:"kind"`
	URL             string              `json:"url,omitempty"`
	Indicator       providers.Indicator `json:"indicator"`
	Description     string              `json:"description"`
	FetchedAt       time.Time           `json:"fetched_at"`
	Stale           bool                `json:"stale"`
	ActiveIncidents int                 `json:"active_incidents"`
	Err             string              `json:"err,omitempty"`
}

type ProviderDetail struct {
	ProviderSummary
	Components []providers.Component `json:"components"`
	Incidents  []providers.Incident  `json:"incidents"`
	Params     json.RawMessage       `json:"params"`
	SortOrder  int                   `json:"sort_order"`
}

func (s *Server) toSummary(e *aggregator.Entry) ProviderSummary {
	snap := e.Snapshot()
	stale := !snap.FetchedAt.IsZero() && time.Since(snap.FetchedAt) > s.Agg.StaleThreshold()
	if snap.FetchedAt.IsZero() && snap.Err != "" {
		stale = true
	}
	return ProviderSummary{
		ID:              e.Config.ID,
		Name:            e.Config.Name,
		Kind:            e.Config.Kind,
		URL:             extractURL(e.Config),
		Indicator:       snap.Indicator,
		Description:     snap.Description,
		FetchedAt:       snap.FetchedAt,
		Stale:           stale,
		ActiveIncidents: len(snap.Incidents),
		Err:             snap.Err,
	}
}

func extractURL(c providers.Config) string {
	if c.Kind != providers.KindStatuspageIO {
		return ""
	}
	var p struct {
		BaseURL string `json:"base_url"`
	}
	_ = json.Unmarshal(c.Params, &p)
	return p.BaseURL
}

// --- healthz ---

type HealthzOutput struct {
	Body struct {
		OK bool `json:"ok"`
	}
}

func (s *Server) healthz(_ context.Context, _ *struct{}) (*HealthzOutput, error) {
	out := &HealthzOutput{}
	out.Body.OK = true
	return out, nil
}

// --- overview ---

type OverviewOutput struct {
	Body struct {
		GeneratedAt    time.Time           `json:"generated_at"`
		WorstIndicator providers.Indicator `json:"worst_indicator"`
		Providers      []ProviderSummary   `json:"providers"`
	}
}

func (s *Server) overview(_ context.Context, _ *struct{}) (*OverviewOutput, error) {
	entries := s.Agg.Entries()
	out := &OverviewOutput{}
	out.Body.GeneratedAt = time.Now().UTC()
	out.Body.Providers = make([]ProviderSummary, 0, len(entries))
	indicators := make([]providers.Indicator, 0, len(entries))
	for _, e := range entries {
		sm := s.toSummary(e)
		out.Body.Providers = append(out.Body.Providers, sm)
		indicators = append(indicators, sm.Indicator)
	}
	out.Body.WorstIndicator = providers.WorstIndicator(indicators)
	return out, nil
}

// --- get provider ---

type GetProviderInput struct {
	ID string `path:"id"`
}

type GetProviderOutput struct {
	Body ProviderDetail
}

func (s *Server) getProvider(_ context.Context, in *GetProviderInput) (*GetProviderOutput, error) {
	e, ok := s.Agg.Get(in.ID)
	if !ok {
		return nil, huma.Error404NotFound("provider not found")
	}
	snap := e.Snapshot()
	sum := s.toSummary(e)
	out := &GetProviderOutput{
		Body: ProviderDetail{
			ProviderSummary: sum,
			Components:      snap.Components,
			Incidents:       snap.Incidents,
			Params:          e.Config.Params,
			SortOrder:       e.Config.SortOrder,
		},
	}
	return out, nil
}

// --- feed kinds ---

type FeedKindInfo struct {
	Kind   providers.Kind         `json:"kind"`
	Label  string                 `json:"label"`
	Fields []providers.ParamField `json:"fields"`
}

type FeedKindsOutput struct {
	Body struct {
		Kinds []FeedKindInfo `json:"kinds"`
	}
}

func (s *Server) feedKinds(_ context.Context, _ *struct{}) (*FeedKindsOutput, error) {
	all := providers.All()
	out := &FeedKindsOutput{}
	out.Body.Kinds = make([]FeedKindInfo, 0, len(all))
	for _, f := range all {
		out.Body.Kinds = append(out.Body.Kinds, FeedKindInfo{
			Kind:   f.Kind(),
			Label:  f.Label(),
			Fields: f.Fields(),
		})
	}
	return out, nil
}

// --- list providers ---

type ListProvidersOutput struct {
	Body struct {
		Providers []ProviderDetail `json:"providers"`
	}
}

func (s *Server) listProviders(_ context.Context, _ *struct{}) (*ListProvidersOutput, error) {
	entries := s.Agg.Entries()
	out := &ListProvidersOutput{}
	out.Body.Providers = make([]ProviderDetail, 0, len(entries))
	for _, e := range entries {
		snap := e.Snapshot()
		sum := s.toSummary(e)
		out.Body.Providers = append(out.Body.Providers, ProviderDetail{
			ProviderSummary: sum,
			Components:      snap.Components,
			Incidents:       snap.Incidents,
			Params:          e.Config.Params,
			SortOrder:       e.Config.SortOrder,
		})
	}
	return out, nil
}

// --- create provider ---

type ProviderBody struct {
	ID        string          `json:"id,omitempty" doc:"Stable slug. Auto-derived from name if omitted on create."`
	Name      string          `json:"name"`
	Kind      providers.Kind  `json:"kind"`
	Params    json.RawMessage `json:"params"`
	SortOrder int             `json:"sort_order,omitempty"`
}

type CreateProviderInput struct {
	Authorization string `header:"Authorization" required:"true"`
	Body          ProviderBody
}

type CreateProviderOutput struct {
	Body ProviderDetail
}

func (s *Server) createProvider(ctx context.Context, in *CreateProviderInput) (*CreateProviderOutput, error) {
	if err := s.checkAdmin(in.Authorization); err != nil {
		return nil, err
	}
	cfg, err := s.validateAndPrepare(ctx, in.Body, true)
	if err != nil {
		return nil, err
	}
	if err := s.Store.Create(ctx, cfg); err != nil {
		return nil, huma.Error500InternalServerError("create failed", err)
	}
	s.Agg.Reload()
	return &CreateProviderOutput{Body: ProviderDetail{
		ProviderSummary: ProviderSummary{
			ID:          cfg.ID,
			Name:        cfg.Name,
			Kind:        cfg.Kind,
			URL:         extractURL(cfg),
			Indicator:   providers.IndicatorUnknown,
			Description: "pending first fetch",
		},
		Params:    cfg.Params,
		SortOrder: cfg.SortOrder,
	}}, nil
}

// --- update provider ---

type UpdateProviderInput struct {
	Authorization string `header:"Authorization" required:"true"`
	ID            string `path:"id"`
	Body          ProviderBody
}

type UpdateProviderOutput struct {
	Body ProviderDetail
}

func (s *Server) updateProvider(ctx context.Context, in *UpdateProviderInput) (*UpdateProviderOutput, error) {
	if err := s.checkAdmin(in.Authorization); err != nil {
		return nil, err
	}
	body := in.Body
	body.ID = in.ID
	cfg, err := s.validateAndPrepare(ctx, body, false)
	if err != nil {
		return nil, err
	}
	if err := s.Store.Update(ctx, cfg); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return nil, huma.Error404NotFound("provider not found")
		}
		return nil, huma.Error500InternalServerError("update failed", err)
	}
	s.Agg.Reload()
	return &UpdateProviderOutput{Body: ProviderDetail{
		ProviderSummary: ProviderSummary{
			ID:   cfg.ID,
			Name: cfg.Name,
			Kind: cfg.Kind,
			URL:  extractURL(cfg),
		},
		Params:    cfg.Params,
		SortOrder: cfg.SortOrder,
	}}, nil
}

// --- delete provider ---

type DeleteProviderInput struct {
	Authorization string `header:"Authorization" required:"true"`
	ID            string `path:"id"`
}

type DeleteProviderOutput struct {
	Body struct {
		OK bool `json:"ok"`
	}
}

func (s *Server) deleteProvider(ctx context.Context, in *DeleteProviderInput) (*DeleteProviderOutput, error) {
	if err := s.checkAdmin(in.Authorization); err != nil {
		return nil, err
	}
	if err := s.Store.Delete(ctx, in.ID); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return nil, huma.Error404NotFound("provider not found")
		}
		return nil, huma.Error500InternalServerError("delete failed", err)
	}
	s.Agg.Reload()
	out := &DeleteProviderOutput{}
	out.Body.OK = true
	return out, nil
}

// --- validate provider ---

type ValidateProviderInput struct {
	Authorization string `header:"Authorization" required:"true"`
	Body          ProviderBody
}

type ValidateProviderOutput struct {
	Body struct {
		OK          bool                `json:"ok"`
		Indicator   providers.Indicator `json:"indicator"`
		Description string              `json:"description"`
	}
}

func (s *Server) validateProvider(ctx context.Context, in *ValidateProviderInput) (*ValidateProviderOutput, error) {
	if err := s.checkAdmin(in.Authorization); err != nil {
		return nil, err
	}
	cfg, err := normalizeConfig(in.Body, true)
	if err != nil {
		return nil, huma.Error400BadRequest(err.Error())
	}
	factory, err := providers.Lookup(cfg.Kind)
	if err != nil {
		return nil, huma.Error400BadRequest(err.Error())
	}
	if err := factory.Validate(ctx, cfg); err != nil {
		return nil, huma.Error400BadRequest("validation failed: " + err.Error())
	}
	p, err := factory.Build(cfg)
	if err != nil {
		return nil, huma.Error400BadRequest("build failed: " + err.Error())
	}
	st, err := p.Fetch(ctx)
	if err != nil {
		return nil, huma.Error400BadRequest("fetch failed: " + err.Error())
	}
	out := &ValidateProviderOutput{}
	out.Body.OK = true
	out.Body.Indicator = st.Indicator
	out.Body.Description = st.Description
	return out, nil
}

// --- helpers ---

func (s *Server) checkAdmin(header string) error {
	if s.AdminToken == "" {
		return huma.Error503ServiceUnavailable("admin endpoints disabled: STATUS_ADMIN_TOKEN is not set")
	}
	const prefix = "Bearer "
	if !strings.HasPrefix(header, prefix) {
		return huma.Error401Unauthorized("missing bearer token")
	}
	if subtleEqual(strings.TrimPrefix(header, prefix), s.AdminToken) {
		return nil
	}
	return huma.Error401Unauthorized("invalid admin token")
}

func subtleEqual(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	var v byte
	for i := 0; i < len(a); i++ {
		v |= a[i] ^ b[i]
	}
	return v == 0
}

var slugRE = regexp.MustCompile(`[^a-z0-9-]+`)

func slugify(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = slugRE.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	return s
}

func normalizeConfig(b ProviderBody, allowDeriveID bool) (providers.Config, error) {
	cfg := providers.Config{
		ID:        strings.TrimSpace(b.ID),
		Name:      strings.TrimSpace(b.Name),
		Kind:      b.Kind,
		Params:    b.Params,
		SortOrder: b.SortOrder,
	}
	if cfg.Name == "" {
		return cfg, fmt.Errorf("name is required")
	}
	if cfg.Kind == "" {
		return cfg, fmt.Errorf("kind is required")
	}
	if cfg.ID == "" {
		if !allowDeriveID {
			return cfg, fmt.Errorf("id is required")
		}
		cfg.ID = slugify(cfg.Name)
	}
	cfg.ID = slugify(cfg.ID)
	if cfg.ID == "" {
		return cfg, fmt.Errorf("id could not be derived; please provide one")
	}
	if len(cfg.Params) == 0 {
		cfg.Params = json.RawMessage(`{}`)
	}
	return cfg, nil
}

func (s *Server) validateAndPrepare(ctx context.Context, b ProviderBody, allowDeriveID bool) (providers.Config, error) {
	cfg, err := normalizeConfig(b, allowDeriveID)
	if err != nil {
		return cfg, huma.Error400BadRequest(err.Error())
	}
	factory, err := providers.Lookup(cfg.Kind)
	if err != nil {
		return cfg, huma.Error400BadRequest(err.Error())
	}
	if err := factory.Validate(ctx, cfg); err != nil {
		return cfg, huma.Error400BadRequest("validation failed: " + err.Error())
	}
	return cfg, nil
}
