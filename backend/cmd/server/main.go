package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/bueti/status-aggregator/backend/internal/aggregator"
	"github.com/bueti/status-aggregator/backend/internal/api"
	"github.com/bueti/status-aggregator/backend/internal/config"
	"github.com/bueti/status-aggregator/backend/internal/store"
	"github.com/bueti/status-aggregator/backend/internal/webui"
)

// Cap on request body size for all API endpoints. JSON provider configs are
// typically <1 KB; 64 KiB is generous headroom without letting an attacker
// stream megabytes into the decoder.
const defaultMaxBodyBytes int64 = 64 << 10

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	addr := envOr("STATUS_ADDR", ":8080")
	dbPath := envOr("STATUS_DB_PATH", "data/providers.db")
	adminToken := os.Getenv("STATUS_ADMIN_TOKEN")
	corsOrigins := parseOrigins(os.Getenv("STATUS_CORS_ORIGIN"))
	maxBodyBytes := defaultMaxBodyBytes

	if err := os.MkdirAll(dirOf(dbPath), 0o755); err != nil {
		logger.Error("mkdir data dir", "err", err)
		os.Exit(1)
	}

	s, err := store.Open(dbPath)
	if err != nil {
		logger.Error("open store", "err", err)
		os.Exit(1)
	}
	defer func() { _ = s.Close() }()

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	if err := s.SeedIfEmpty(ctx, config.DefaultProviders()); err != nil {
		logger.Error("seed defaults", "err", err)
		os.Exit(1)
	}

	agg := aggregator.New(s, logger)
	go func() {
		if err := agg.Run(ctx); err != nil {
			logger.Error("aggregator stopped", "err", err)
		}
	}()

	router := chi.NewRouter()
	router.Use(middleware.RequestID)
	router.Use(middleware.RealIP)
	router.Use(middleware.Recoverer)
	router.Use(corsMiddleware(corsOrigins))
	router.Use(bodyLimitMiddleware(maxBodyBytes))

	if len(corsOrigins) > 0 {
		logger.Info("CORS enabled", "origins", corsOriginList(corsOrigins))
	}

	humaCfg := huma.DefaultConfig("Status Aggregator", "0.1.0")
	humaCfg.Info.Description = "Aggregates third-party Statuspage.io feeds (GitHub, depot.dev, Grafana Cloud, etc.)."
	humaAPI := humachi.New(router, humaCfg)

	server := &api.Server{
		Agg:        agg,
		Store:      s,
		AdminToken: adminToken,
	}
	server.Register(humaAPI)

	if ui := webui.Handler(); ui != nil {
		router.NotFound(ui.ServeHTTP)
		logger.Info("serving embedded web UI at /")
	}

	srv := &http.Server{
		Addr:              addr,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdownCtx)
	}()

	if adminToken == "" {
		logger.Warn("STATUS_ADMIN_TOKEN is unset; write endpoints will return 503")
	}
	logger.Info("listening", "addr", addr, "docs", "http://"+humanAddr(addr)+"/docs")

	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.Error("listen", "err", err)
		os.Exit(1)
	}
}

// corsMiddleware echoes Access-Control-Allow-Origin only for origins in the
// configured allowlist. When allowed is empty, CORS is disabled entirely —
// appropriate for the production setup where the UI is served from the same
// origin as the API. Set STATUS_CORS_ORIGIN to a comma-separated list of
// origins (e.g. http://localhost:5173) to enable cross-origin access in dev.
func corsMiddleware(allowed map[string]bool) func(http.Handler) http.Handler {
	if len(allowed) == 0 {
		return func(next http.Handler) http.Handler { return next }
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if origin := r.Header.Get("Origin"); origin != "" && allowed[origin] {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
				w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
				w.Header().Add("Vary", "Origin")
			}
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func bodyLimitMiddleware(max int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Body != nil {
				r.Body = http.MaxBytesReader(w, r.Body, max)
			}
			next.ServeHTTP(w, r)
		})
	}
}

func parseOrigins(s string) map[string]bool {
	out := map[string]bool{}
	for _, o := range strings.Split(s, ",") {
		if o = strings.TrimSpace(o); o != "" {
			out[o] = true
		}
	}
	return out
}

func corsOriginList(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for o := range m {
		out = append(out, o)
	}
	return out
}

func envOr(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func dirOf(p string) string {
	i := strings.LastIndex(p, "/")
	if i < 0 {
		return "."
	}
	return p[:i]
}

func humanAddr(a string) string {
	if strings.HasPrefix(a, ":") {
		return "localhost" + a
	}
	return a
}
