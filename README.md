# Aether Telegram Bot

A simple yet powerful Telegram bot to download media from various social media platforms. Just send a link and the bot will handle the rest.

## ✨ Features

-   Download videos and audio from a wide range of platforms.
-   Easy to use: just send a link to the bot.
-   Support for sending media as single files or grouped as an album.
-   Easy to deploy with Docker and Docker Compose.
-   Lightweight and fast, built with Go.

## ✨ Supported Platforms

-   Bilibili
-   Bluesky
-   Dailymotion
-   Facebook
-   Instagram
-   Loom
-   OK.ru
-   Pinterest
-   Newgrounds
-   Reddit
-   Rutube
-   Snapchat
-   Soundcloud
-   Streamable
-   TikTok
-   Tumblr
-   Twitch
-   Twitter
-   Vimeo
-   VK
-   Xiaohongshu
-   YouTube

## 🤖 Commands

-   `/start` or `/help`: Shows the welcome message and list of commands.
-   `/stats`: Shows the bot's status.
-   `/support`: Shows support information.


## 🚀 Getting Started

### Prerequisites

-   Docker and Docker Compose
-   A Telegram Bot Token. Get one from [@BotFather](https://t.me/BotFather).
-   A Telegram API ID and Hash. Get them from [my.telegram.org](https://my.telegram.org).
-   A Telegram OWNER ID . Get one from [@userinfobot](https://t.me/userinfobot).

### 🐳 Docker Deployment (Recommended)

1.  **Clone the repository:**
    ```bash
    git clone https://github.com/pavelc4/aether-tg-bot.git
    cd aether-tg-bot
    ```

2.  **Create a `.env` file:**
    Copy the `.env.example` to `.env` and fill in your credentials.
    ```bash
    cp .env.example .env
    ```
    Your `.env` file should look like this:
    ```
    BOT_TOKEN=YOUR_BOT_TOKEN
    TELEGRAM_API_ID=YOUR_TELEGRAM_API_ID
    TELEGRAM_API_HASH=YOUR_TELEGRAM_API_HASH
    OWNER_ID=YOUR_ID_TELEGRAM
    ```

3.  **Run with Docker Compose:**
    ```bash
    docker compose up --build -d
    ```

## 🙏 Credits

This bot uses the powerful [Cobalt API](https://github.com/imputnet/cobalt) for downloading media. Many thanks to the Cobalt team for their great work!

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
