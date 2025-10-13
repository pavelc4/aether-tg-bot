# ---- Builder Stage ----
FROM golang:alpine AS builder

# Install build dependencies
RUN apk add --no-cache upx ca-certificates git

WORKDIR /app

# Copy and download modules to leverage Docker cache
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build static binary with optimizations
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-w -s" \
    -trimpath \
    -o aether-bot . && \
    upx --best --lzma ./aether-bot

# ---- Final Stage ----
FROM alpine:3.21

# Create non-root user
RUN addgroup -S appgroup && adduser -S appuser -G appgroup

# Install runtime dependencies and yt-dlp in single layer
RUN apk add --no-cache \
    ca-certificates \
    python3 \
    py3-pip \
    ffmpeg && \
    pip install --no-cache-dir --break-system-packages yt-dlp && \
    rm -rf /root/.cache

WORKDIR /app

# Copy binary from builder
COPY --from=builder --chown=appuser:appgroup /app/aether-bot .

# Switch to non-root user
USER appuser

# Health check (optional - sesuaikan dengan app kamu)
# HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
#   CMD ./aether-bot --health || exit 1

CMD ["./aether-bot"]