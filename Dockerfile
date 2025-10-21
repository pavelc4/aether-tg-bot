# ---- Builder Stage ----
FROM golang:alpine AS builder

# Install build dependencies
RUN apk add --no-cache upx ca-certificates git

WORKDIR /app

# Copy go.mod and go.sum to leverage Docker cache
COPY go.mod go.sum ./
RUN go mod download && go mod verify

# Copy source code
COPY . .

# Build static binary with optimizations
# FIXED: Build from ./cmd/bot instead of root
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s -X main.version=$(git describe --tags --always --dirty 2>/dev/null || echo 'dev')" \
    -trimpath \
    -o aether-bot ./cmd/bot && \
    upx --best --lzma ./aether-bot

# ---- Final Stage ----
FROM alpine:3.21

# Metadata
LABEL maintainer="your-email@example.com"
LABEL description="Aether Telegram Downloader Bot"
LABEL org.opencontainers.image.source="https://github.com/pavelc4/aether-tg-bot"

# Create non-root user
RUN addgroup -S appgroup && \
    adduser -S appuser -G appgroup

# Install runtime dependencies (aria2, yt-dlp, ffmpeg) in single layer
RUN apk add --no-cache \
    ca-certificates \
    python3 \
    py3-pip \
    ffmpeg \
    aria2 \
    tzdata && \
    pip install --no-cache-dir --break-system-packages yt-dlp && \
    rm -rf /root/.cache /tmp/* /var/cache/apk/*

# Set timezone (optional, adjust as needed)
ENV TZ=Asia/Jakarta

WORKDIR /app

# Copy binary from builder
COPY --from=builder --chown=appuser:appgroup /app/aether-bot .

# Create temp directory for downloads
RUN mkdir -p /tmp/aether && \
    chown -R appuser:appgroup /tmp/aether

# Switch to non-root user
USER appuser

# Health check (optional)
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD pgrep -x aether-bot > /dev/null || exit 1

# Expose port if needed (for metrics/health endpoint)
# EXPOSE 8080

CMD ["./aether-bot"]
