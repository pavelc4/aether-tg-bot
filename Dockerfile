FROM golang:latest AS builder
RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    git \
    upx && \
    rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download && go mod verify

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s -X main.version=$(git describe --tags --always --dirty 2>/dev/null || echo 'dev')" \
    -trimpath \
    -o aether-bot ./cmd/bot && \
    upx --best --lzma ./aether-bot

FROM debian:bookworm-slim

RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    python3 \
    python3-pip \
    ffmpeg \
    aria2 \
    tzdata \
    curl \
    procps && \
    pip install --no-cache-dir --break-system-packages yt-dlp && \
    apt-get clean && \
    rm -rf /var/lib/apt/lists/* /tmp/* /var/tmp/* /root/.cache

RUN groupadd -r appgroup && \
    useradd -r -g appgroup -u 1000 -d /app -s /sbin/nologin -c "App user" appuser

WORKDIR /app

COPY --from=builder --chown=appuser:appgroup /app/aether-bot .

RUN mkdir -p /tmp/aether && \
    chown -R appuser:appgroup /tmp/aether

USER appuser

HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD ps aux | grep "[a]ether-bot" > /dev/null || exit 1

CMD ["./aether-bot"]
