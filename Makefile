.PHONY: dev dev-backend dev-frontend gen build check clean

STATUS_ADMIN_TOKEN ?= dev-token
STATUS_DB_PATH    ?= $(CURDIR)/data/providers.db

export STATUS_ADMIN_TOKEN
export STATUS_DB_PATH

dev-backend:
	cd backend && go run ./cmd/server

dev-frontend:
	cd frontend && npm run dev

# Requires the backend to be running (for OpenAPI at :8080/openapi.json)
gen:
	cd frontend && npm run gen

build:
	cd backend && go build -o ../bin/status-aggregator ./cmd/server
	cd frontend && npm run build

check:
	cd backend && go vet ./... && go build ./...
	cd frontend && npm run check

clean:
	rm -rf bin data frontend/.svelte-kit frontend/build
