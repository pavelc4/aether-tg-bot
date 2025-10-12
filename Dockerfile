# ---- Builder Stage ----
# This stage compiles the Go application into a binary.
FROM golang:1.24.6-alpine AS builder

# Install build dependencies.
RUN apk add --no-cache upx ca-certificates git

# Set working directory
WORKDIR /app

# Copy and download modules to leverage Docker cache
COPY go.mod go.sum ./
RUN go mod download

# Copy the remaining source code
COPY . .

# Build the Go application into a static binary and compress it with UPX
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o aether-bot . && \
    upx --best --lzma ./aether-bot

# ---- Final Stage (Optimized for Alpine) ----
# This stage creates the final minimal image using Alpine.
FROM alpine:3.21

# Create a non-root user and group first for security.
# Using -S (system) and -G (group) creates a user without a home dir or password.
RUN addgroup -S appgroup && adduser -S appuser -G appgroup

# Install ONLY the absolute necessary runtime dependencies.
# --no-cache ensures no extra cache data is stored in the layer, saving space.
RUN apk add --no-cache ca-certificates python3 py3-pip ffmpeg && \
    pip install --no-cache-dir -U yt-dlp --break-system-packages

# Set working directory
WORKDIR /app

# Copy the pre-compressed binary from the builder stage.
COPY --from=builder /app/aether-bot .

# Change ownership of the app directory to the new user.
# This is more secure than chown-ing a system directory like /usr/local/bin.
RUN chown -R appuser:appgroup /app

# Switch to the non-root user.
USER appuser

# Command to run the application.
CMD ["./aether-bot"]