FROM golang:1.25-alpine AS builder
RUN apk update && apk add --no-cache \
	ca-certificates \
	git

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download && go mod verify

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
	-ldflags="-w -s -X main.version=$(git describe --tags --always --dirty 2>/dev/null || echo 'dev')" \
	-trimpath \
	-o aether-bot ./cmd/bot


FROM debian:bookworm-slim

RUN apt-get update && apt-get install -y --no-install-recommends \
	ca-certificates \
	ffmpeg \
	tzdata \
	curl \
	procps && \
	apt-get clean && \
	rm -rf /var/lib/apt/lists/* /tmp/* /var/tmp/* /root/.cache

RUN curl -L https://github.com/yt-dlp/yt-dlp/releases/latest/download/yt-dlp_linux \
	-o /usr/local/bin/yt-dlp && chmod +x /usr/local/bin/yt-dlp

RUN groupadd -r appgroup && \
	useradd -r -g appgroup -u 1000 -d /app -s /sbin/nologin -c "App user" appuser

WORKDIR /app

COPY --from=builder --chown=appuser:appgroup /app/aether-bot /app/aether-bot

RUN mkdir -p /app/data && \
	chown -R appuser:appgroup /app/data && \
	chmod 755 /app/data

RUN mkdir -p /tmp/aether && \
	chown -R appuser:appgroup /tmp/aether

VOLUME ["/app/data"]

USER appuser

HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
	CMD ps aux | grep "[a]ether-bot" > /dev/null || exit 1

CMD ["./aether-bot"]
