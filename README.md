# Aether - MTProto Streaming Bot

A high-performance Telegram bot that streams media from Cobalt, TikTok, and YouTube directly to Telegram without intermediate disk storage.

## 🚀 Features

- **Streaming Architecture**: Uses `io.ReadCloser` piping to download and upload simultaneously.
- **MTProto Uploader**: Uses Telegram's native MTProto for max speed and large file support (up to 2GB+).
- **Concurrency**: Parallel chunk uploading with worker pools.
- **Low Memory Footprint**: Uses pooled buffers and efficient memory management.
- **Multiple Providers**: Supports Cobalt, TikTok, and YouTube (via yt-dlp).

## 🛠 Project Structure

```
aether-bot/
├── cmd/bot/          # Entry point
├── config/           # Configuration
├── internal/
│   ├── app/          # App wiring & shutdown
│   ├── bot/          # Bot core & router
│   ├── handler/      # Command & Download handlers
│   ├── provider/     # Download providers (Cobalt, YT, etc.)
│   ├── streaming/    # Core streaming engine
│   └── telegram/     # MTProto wrappers
├── pkg/              # Shared utilities (buffer, http, worker)
└── data/             # Session storage
```

## ⚙️ Configuration

Copy `.env.example` to `.env` and fill in the values:

```bash
# Telegram App Credentials (my.telegram.org)
TELEGRAM_APP_ID=123456
TELEGRAM_APP_HASH=your_api_hash
BOT_TOKEN=123:ABC

# Owner
OWNER_ID=123456789

# APIs
COBALT_API=http://cobalt:9000
YTDLP_API=http://yt-dlp:8080

# Streaming Tweak
MAX_CONCURRENT_STREAMS=8
CHUNK_SIZE=1048576 # 1MB
```

## 🏃 Running

### Local
```bash
go run ./cmd/bot
```

### Docker
```bash
docker-compose up -d --build
```

## 🔧 Architecture

The bot uses a **Download → Pipeline → Upload** architecture:

1. **Provider** resolves URL to a direct stream (`http.Response.Body`).
2. **Stream Manager** initiates a `Pipeline`.
3. **Pipeline** reads from stream into 1MB chunks (using buffer pool).
4. **Upload Workers** (concurrent) pick chunks and upload via MTProto `saveBigFilePart`.
5. **State Manager** tracks progress and retries.
6. Once complete, `sendMedia` commits the file to Telegram.

## 📝 License

MIT
