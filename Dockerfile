FROM debian:bookworm-slim AS downloader
RUN apt-get update && apt-get install -y curl ca-certificates zlib1g && rm -rf /var/lib/apt/lists/*
RUN curl -L https://github.com/yt-dlp/yt-dlp/releases/latest/download/yt-dlp_linux \
    -o /yt-dlp && chmod +x /yt-dlp

RUN mkdir -p /data /cookies /downloads /tmp_aether && \
    chmod 777 /data /cookies /downloads /tmp_aether

FROM gcr.io/distroless/cc-debian12
COPY --from=mwader/static-ffmpeg:6.0 /ffmpeg /usr/local/bin/
COPY --from=mwader/static-ffmpeg:6.0 /ffprobe /usr/local/bin/
COPY --from=downloader /lib/x86_64-linux-gnu/libz.so.1 /lib/x86_64-linux-gnu/
COPY --from=downloader --chown=nonroot:nonroot /yt-dlp /usr/local/bin/yt-dlp
COPY --chown=nonroot:nonroot aether-bot /app/aether-bot
COPY --from=downloader --chown=nonroot:nonroot /data /app/data
COPY --from=downloader --chown=nonroot:nonroot /cookies /app/cookies
COPY --from=downloader --chown=nonroot:nonroot /downloads /app/downloads
COPY --from=downloader --chown=nonroot:nonroot /tmp_aether /tmp/aether

ENV PATH="/usr/local/bin:${PATH}"
ENV HOME="/app"

USER nonroot
WORKDIR /app
CMD ["./aether-bot"]
