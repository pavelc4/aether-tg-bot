# ---- Builder Stage ----
# This stage compiles the Go application into a binary.
FROM golang:1.24.6-alpine AS builder

# Install build dependencies.
# upx for binary compression, ca-certificates for HTTPS connections.
RUN apk add --no-cache upx ca-certificates git

# Set working directory
WORKDIR /app

# Copy dependency files and download modules to leverage Docker cache
COPY go.mod go.sum ./
RUN go mod download

# Copy the remaining source code
COPY . .

# Build the Go application into a static binary and compress it with UPX
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o aether-bot . && \
    upx --best --lzma ./aether-bot

# ---- Final Stage ----
# This stage creates the final minimal image containing only the compiled binary.
FROM alpine:3.21

# Install runtime dependencies
RUN apk add --no-cache ca-certificates

# Create a non-root user and group
RUN addgroup -S appgroup && adduser -S appuser -G appgroup

# Set environment variable for SSL certificates
ENV SSL_CERT_FILE=/etc/ssl/certs/ca-certificates.crt

# Set working directory
WORKDIR /app

# Copy the binary from the builder stage
COPY --from=builder /app/aether-bot /usr/local/bin/aether-bot

# Change ownership of the binary to the new user
RUN chown appuser:appgroup /usr/local/bin/aether-bot

# Switch to non-root user
USER appuser

# Command to run the application
CMD ["aether-bot"]
