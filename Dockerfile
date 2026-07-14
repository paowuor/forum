# --- Build stage ---
FROM golang:1.22-bookworm AS builder

WORKDIR /app

# go-sqlite3 uses CGO, so we need a C toolchain in the build stage.
RUN apt-get update && apt-get install -y --no-install-recommends gcc libc6-dev \
    && rm -rf /var/lib/apt/lists/*
ENV CGO_ENABLED=1

# Cache dependency downloads separately from source changes.
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o forum-server ./cmd/server

# --- Runtime stage ---
FROM debian:bookworm-slim

WORKDIR /app

# ca-certificates isn't strictly needed for this app (no outbound HTTPS calls
# yet), but it's cheap insurance if that changes later.
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates \
    && rm -rf /var/lib/apt/lists/*

COPY --from=builder /app/forum-server ./forum-server
COPY --from=builder /app/web ./web

# The SQLite file lives here; mount a volume over this path (see docker-compose.yml)
# so data survives container restarts/rebuilds.
RUN mkdir -p /app/data
VOLUME /app/data

EXPOSE 8080

CMD ["./forum-server"]
