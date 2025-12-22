FROM golang:latest AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s -X main.version=$(git describe --tags --always --dirty 2>/dev/null || echo 'dev')" \
    -trimpath \
    -o aether-bot ./cmd/bot

FROM debian:bookworm-slim AS downloader
RUN apt-get update && apt-get install -y curl ca-certificates && rm -rf /var/lib/apt/lists/*
RUN curl -L https://github.com/yt-dlp/yt-dlp/releases/latest/download/yt-dlp_linux \
    -o /yt-dlp && chmod +x /yt-dlp

RUN mkdir -p /data /cookies /downloads /tmp_aether && \
    chmod 777 /data /cookies /downloads /tmp_aether

FROM gcr.io/distroless/cc-debian12
COPY --from=mwader/static-ffmpeg:6.0 /ffmpeg /usr/local/bin/
COPY --from=mwader/static-ffmpeg:6.0 /ffprobe /usr/local/bin/
COPY --from=downloader --chown=nonroot:nonroot /yt-dlp /usr/local/bin/yt-dlp
COPY --from=builder --chown=nonroot:nonroot /app/aether-bot /app/aether-bot
COPY --from=downloader --chown=nonroot:nonroot /data /app/data
COPY --from=downloader --chown=nonroot:nonroot /cookies /app/cookies
COPY --from=downloader --chown=nonroot:nonroot /downloads /app/downloads
COPY --from=downloader --chown=nonroot:nonroot /tmp_aether /tmp/aether

ENV PATH="/usr/local/bin:${PATH}"
ENV HOME="/app"

USER nonroot
WORKDIR /app
CMD ["./aether-bot"]
