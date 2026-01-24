<h1 align="center">
    Aether Telegram Bot
</h1>

<p align="center">
    <img src="https://img.shields.io/badge/Go-00ADD8?style=for-the-badge&colorA=363A4F&logo=go&logoColor=D9E0EE">
    <img src="https://img.shields.io/badge/Telegram-26A5E4?style=for-the-badge&colorA=363A4F&logo=telegram&logoColor=D9E0EE">
    <img src="https://img.shields.io/badge/Docker-2496ED?style=for-the-badge&colorA=363A4F&logo=docker&logoColor=D9E0EE">
    <img src="https://img.shields.io/badge/MTProto-229ED9?style=for-the-badge&colorA=363A4F&logo=telegram&logoColor=D9E0EE">
</p>



---

## About

**Aether** is a high-performance Telegram bot that streams media from multiple sources directly to Telegram without intermediate disk storage.

It leverages **MTProto's native protocol** for maximum upload speed and supports files up to **2GB+**, while maintaining a **minimal memory footprint** through intelligent buffer pooling and concurrent chunk processing.

---

## Features

- **Zero-Disk Streaming Architecture**  
  Direct pipe from download stream to upload using `io.ReadCloser`.

- **MTProto Native Upload**  
  Utilizes Telegram's MTProto protocol for blazing-fast transfers and large file support.

- **Concurrent Processing**  
  Parallel chunk uploading with configurable worker pools for optimal throughput.

- **Memory Efficient**  
  Pooled buffer system and efficient memory management keep RAM usage minimal.

- **Multi-Provider Support**  
  Seamlessly downloads from Cobalt, TikTok, and YouTube (via yt-dlp).

- **Robust Pipeline**  
  State tracking, automatic retries, and graceful error handling.

---

## Project Structure

```
aether-bot/
├── cmd/bot/          # Application entry point
├── config/           # Configuration management
├── internal/
│   ├── app/          # Application wiring & graceful shutdown
│   ├── bot/          # Bot core & command router
│   ├── handler/      # Command & download handlers
│   ├── provider/     # Download providers (Cobalt, YouTube, etc.)
│   ├── streaming/    # Core streaming engine & pipeline
│   └── telegram/     # MTProto protocol wrappers
├── pkg/              # Shared utilities (buffers, HTTP, workers)
└── data/             # Session & state storage
```

---

## Configuration

Copy `.env.example` to `.env` and configure your credentials:

```bash
# Telegram Application Credentials (from my.telegram.org)
TELEGRAM_APP_ID=123456
TELEGRAM_APP_HASH=your_api_hash_here
BOT_TOKEN=123456789:ABCdefGHIjklMNOpqrsTUVwxyz

# Bot Owner
OWNER_ID=123456789

# External APIs
COBALT_API=http://cobalt:9000
YTDLP_API=http://yt-dlp:8080

# Performance Tuning
MAX_CONCURRENT_STREAMS=8      # Maximum parallel downloads
CHUNK_SIZE=1048576            # Upload chunk size (1MB default)
```

---

## Getting Started

### Local Development

```bash
# Clone the repository
git clone https://github.com/yourusername/aether-bot.git
cd aether-bot

# Configure environment
cp .env.example .env
# Edit .env with your credentials

# Run the bot
go run ./cmd/bot
```

### Docker Deployment

```bash
# Build and start all services
docker-compose up -d --build

# View logs
docker-compose logs -f aether-bot

# Stop services
docker-compose down
```

---

## Architecture

Aether implements a **Download → Pipeline → Upload** streaming architecture:

1. **Provider Resolution**  
   URL is resolved to a direct HTTP stream (`http.Response.Body`)

2. **Pipeline Initialization**  
   Stream Manager creates a new Pipeline for the download

3. **Chunk Processing**  
   Pipeline reads stream into 1MB chunks using pooled buffers

4. **Concurrent Upload**  
   Worker pool uploads chunks in parallel via MTProto `saveBigFilePart`

5. **State Management**  
   Progress tracking, retry logic, and error recovery

6. **File Commit**  
   Once complete, `sendMedia` finalizes the upload to Telegram

---

## Supported Platforms

- **Cobalt** – Universal media downloader
- **TikTok** – Direct video extraction
- **YouTube** – High-quality video/audio via yt-dlp

---

## Requirements

- **Go** – Version 1.21 or higher
- **Telegram Bot Token** – Create via [@BotFather](https://t.me/botfather)
- **API Credentials** – Register at [my.telegram.org](https://my.telegram.org)
- **Docker** (optional) – For containerized deployment

---

## Performance

- **Upload Speed**: Up to 50MB/s (network dependent)
- **Memory Usage**: ~50-100MB during active streaming
- **Concurrent Streams**: Configurable (default: 8)
- **File Size Limit**: 2GB (Telegram's maximum free )

---

## License

Aether is open-sourced software licensed under the **MIT License**.  
See the [LICENSE](LICENSE) file for more information.

---

