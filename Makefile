.PHONY: dev dev-backend dev-frontend gen build check clean docker-build docker-run helm-lint helm-template

STATUS_ADMIN_TOKEN ?= dev-token
STATUS_DB_PATH     ?= $(CURDIR)/data/providers.db
STATUS_CORS_ORIGIN ?= http://localhost:5173

export STATUS_ADMIN_TOKEN
export STATUS_DB_PATH
export STATUS_CORS_ORIGIN

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

IMAGE ?= ghcr.io/bueti/status-aggregator:dev

docker-build:
	docker build -t $(IMAGE) .

docker-run:
	mkdir -p data
	docker run --rm -p 8080:8080 \
		-e STATUS_ADMIN_TOKEN=$(STATUS_ADMIN_TOKEN) \
		-v $(CURDIR)/data:/data \
		$(IMAGE)

helm-lint:
	helm lint charts/status-aggregator

helm-template:
	helm template status charts/status-aggregator
