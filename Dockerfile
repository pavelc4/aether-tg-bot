FROM debian:bookworm-slim

RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    ffmpeg \
    curl \
    && rm -rf /var/lib/apt/lists/*

RUN curl -L https://github.com/yt-dlp/yt-dlp/releases/latest/download/yt-dlp_linux \
    -o /usr/local/bin/yt-dlp && \
    chmod +x /usr/local/bin/yt-dlp

RUN groupadd -r appgroup && \
    useradd -r -g appgroup -u 1000 -d /app -s /sbin/nologin -c "App user" appuser

WORKDIR /app
COPY --chown=appuser:appgroup aether-bot /app/aether-bot

RUN mkdir -p /app/data /app/cookies /app/downloads /tmp/aether && \
    chown -R appuser:appgroup /app/data /app/cookies /app/downloads /tmp/aether && \
    chmod 755 /app/data /app/cookies /app/downloads

VOLUME ["/app/data"]
USER appuser

CMD ["./aether-bot"]
