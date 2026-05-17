# ── Stage 1: Build ──────────────────────────────────────────────────────────
FROM golang:1.25-alpine AS builder

# Install build tools
RUN apk add --no-cache git ca-certificates

WORKDIR /app

# Cache dependencies first (layer caching optimization)
COPY go.mod go.sum ./
RUN go mod download

# Copy source and build a statically linked binary
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s -extldflags '-static'" \
    -o /secure-auth ./cmd/server

# ── Stage 2: Minimal Runtime ────────────────────────────────────────────────
# Use distroless — no shell, no package manager, minimal attack surface.
FROM gcr.io/distroless/static-debian12

COPY --from=builder /secure-auth /secure-auth
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Non-root user (distroless default uid 65532 = "nonroot")
USER nonroot:nonroot

EXPOSE 8080

ENTRYPOINT ["/secure-auth"]
