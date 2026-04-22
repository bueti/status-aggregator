################################################################################
# Stage 1 — frontend build
################################################################################
FROM node:22-alpine AS frontend
WORKDIR /app/frontend

COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci

COPY frontend/ ./
RUN npm run build

################################################################################
# Stage 2 — backend build (embeds the built frontend)
################################################################################
FROM golang:1.26-alpine AS backend
WORKDIR /src

# Cache modules
COPY backend/go.mod backend/go.sum ./backend/
RUN cd backend && go mod download

COPY backend/ ./backend/

# Overlay the built SPA into the embed directory
RUN rm -rf ./backend/internal/webui/dist && mkdir -p ./backend/internal/webui/dist
COPY --from=frontend /app/frontend/build/ ./backend/internal/webui/dist/

ARG VERSION=dev
ARG COMMIT=unknown

RUN CGO_ENABLED=0 GOOS=linux go -C backend build \
    -trimpath \
    -ldflags "-s -w -X main.version=${VERSION} -X main.commit=${COMMIT}" \
    -o /out/status-aggregator ./cmd/server

################################################################################
# Stage 3 — runtime (distroless, non-root)
################################################################################
FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /app

COPY --from=backend --chown=nonroot:nonroot /out/status-aggregator /app/status-aggregator

ENV STATUS_ADDR=:8080 \
    STATUS_DB_PATH=/data/providers.db

EXPOSE 8080
VOLUME ["/data"]
USER nonroot:nonroot

ENTRYPOINT ["/app/status-aggregator"]
