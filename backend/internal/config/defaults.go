package config

import (
	"encoding/json"

	"github.com/bbu/status-aggregator/backend/internal/providers"
)

func DefaultProviders() []providers.Config {
	mk := func(id, name, baseURL string) providers.Config {
		p, _ := json.Marshal(map[string]string{"base_url": baseURL})
		return providers.Config{
			ID:     id,
			Name:   name,
			Kind:   providers.KindStatuspageIO,
			Params: p,
		}
	}
	return []providers.Config{
		mk("github", "GitHub", "https://www.githubstatus.com"),
		mk("depot", "depot.dev", "https://status.depot.dev"),
		mk("grafana-cloud", "Grafana Cloud", "https://status.grafana.com"),
		mk("cloudflare", "Cloudflare", "https://www.cloudflarestatus.com"),
		mk("vercel", "Vercel", "https://www.vercel-status.com"),
	}
}
