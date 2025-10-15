package bot

import (
	"context"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// Compile regex once at package level for optimal performance
var (
	urlRegex = regexp.MustCompile(`(https?://[^\n]+)`)
)

const (
	maxMediaGroupSize = 10
	processingTimeout = 5 * time.Minute
)

type commandHandlerFunc func(*tgbotapi.BotAPI, *tgbotapi.Message)

var commandHandlers = map[string]commandHandlerFunc{
	"start":    HandleHelpCommand,
	"help":     HandleHelpCommand,
	"stats":    HandleStatusCommand,
	"support":  HandleSupportCommand,
	"tikaudio": handleTikTokAudioCommand,
	// FIX: Register download commands
	"mp":    handleDownloadCommand, // Audio download
	"video": handleDownloadCommand, // Video download
	"dl":    handleDownloadCommand, // Generic download
}

// Source detection map
var sourceMap = map[string]string{
	"bilibili.com":    "Bilibili",
	"bluesky.app":     "Bluesky",
	"dailymotion.com": "Dailymotion",
	"facebook.com":    "Facebook",
	"instagram.com":   "Instagram",
	"loom.com":        "Loom",
	"ok.ru":           "OK",
	"pinterest.com":   "Pinterest",
	"newgrounds.com":  "Newgrounds",
	"reddit.com":      "Reddit",
	"rutube.ru":       "Rutube",
	"snapchat.com":    "Snapchat",
	"soundcloud.com":  "Soundcloud",
	"streamable.com":  "Streamable",
	"tiktok.com":      "TikTok",
	"tumblr.com":      "Tumblr",
	"twitch.tv":       "Twitch",
	"x.com":           "X",
	"vimeo.com":       "Vimeo",
	"vk.com":          "VK",
	"xiaohongshu.com": "Xiaohongshu",
	"youtube.com":     "YouTube",
}

func handleCommand(bot *tgbotapi.BotAPI, msg *tgbotapi.Message) {
	handler, found := commandHandlers[msg.Command()]
	if !found {
		bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "❌ Unknown command. Type /help to see the list of commands."))
		return
	}
	handler(bot, msg)
}

func handleMessage(bot *tgbotapi.BotAPI, msg *tgbotapi.Message) {
	url := urlRegex.FindString(msg.Text)
	if url == "" {
		return
	}

	processingMsg, err := bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "⏳ Processing link, please wait..."))
	if err != nil {
		log.Printf("Failed to send processing message: %v", err)
		return
	}
	defer deleteMessage(bot, msg.Chat.ID, processingMsg.MessageID)

	ctx, cancel := context.WithTimeout(context.Background(), processingTimeout)
	defer cancel()

	if err := processURLMessage(ctx, bot, msg, url, processingMsg.MessageID); err != nil {
		sendError(bot, msg.Chat.ID, fmt.Sprintf("❌ Failed to process link: %s", err.Error()))
	}
}

func processURLMessage(ctx context.Context, bot *tgbotapi.BotAPI, msg *tgbotapi.Message, url string, processingMsgID int) error {
	finalURL, err := ResolveFinalURL(url)
	if err != nil {
		return fmt.Errorf("failed to process link: %w", err)
	}

	source := detectSource(finalURL)
	updateProcessingMessage(bot, msg.Chat.ID, processingMsgID, fmt.Sprintf("⏳ Source detected: %s. Downloading content...", source))

	start := time.Now()
	mediaType, filePaths, totalSize, err := downloadMediaByContentType(ctx, url, finalURL)
	if err != nil {
		return err
	}
	defer CleanupTempFiles(filePaths)

	return sendMediaFiles(bot, msg, filePaths, totalSize, mediaType, source, finalURL, start)
}

// Helper: Detect source from URL
func detectSource(url string) string {
	for domain, name := range sourceMap {
		if strings.Contains(url, domain) {
			return name
		}
	}
	return "Unknown"
}

// Helper: Download media based on content type
func downloadMediaByContentType(ctx context.Context, originalURL, finalURL string) (string, []string, int64, error) {
	req, err := http.NewRequestWithContext(ctx, "HEAD", finalURL, nil)
	if err != nil {
		return "", nil, 0, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := downloadClient.Do(req)
	if err != nil {
		return "", nil, 0, fmt.Errorf("failed to check content type: %w", err)
	}
	defer resp.Body.Close()

	contentType := resp.Header.Get("Content-Type")
	log.Printf("Content-Type for %s: %s", finalURL, contentType)

	switch {
	case strings.HasPrefix(contentType, "image/"):
		filePath, size, err := DownloadImage(finalURL)
		if err != nil {
			return "", nil, 0, fmt.Errorf("failed to download image: %w", err)
		}
		return "Image", []string{filePath}, size, nil

	default: // Video or unknown
		paths, size, _, err := DownloadVideo(originalURL)
		if err != nil {
			return "", nil, 0, fmt.Errorf("failed to download content: %w", err)
		}
		return "Video", paths, size, nil
	}
}

// Helper: Send media files (single or group)
func sendMediaFiles(bot *tgbotapi.BotAPI, msg *tgbotapi.Message, filePaths []string, totalSize int64, mediaType, source, url string, start time.Time) error {
	if len(filePaths) == 0 {
		return fmt.Errorf("no file to send")
	}

	if len(filePaths) > 1 {
		return sendMediaGroup(bot, msg, filePaths, totalSize, source, url, start)
	}

	return processAndSendMediaWithMeta(bot, msg, filePaths[0], totalSize, source, start, mediaType, url)
}

// Helper: Send media group
// FIX: Remove unused mediaType parameter
func sendMediaGroup(bot *tgbotapi.BotAPI, msg *tgbotapi.Message, filePaths []string, totalSize int64, source, url string, start time.Time) error {
	duration := time.Since(start).Truncate(time.Second)
	caption := BuildMediaCaption(source, url, "Media", totalSize, duration, GetUserName(msg))

	mediaGroup := make([]interface{}, 0, maxMediaGroupSize)
	for i, path := range filePaths {
		if i >= maxMediaGroupSize {
			break
		}

		photo := tgbotapi.NewInputMediaPhoto(tgbotapi.FilePath(path))
		if i == 0 {
			photo.Caption = caption
			photo.ParseMode = "MarkdownV2"
		}
		mediaGroup = append(mediaGroup, photo)
	}

	if len(mediaGroup) == 0 {
		return fmt.Errorf("no media to send")
	}

	group := tgbotapi.NewMediaGroup(msg.Chat.ID, mediaGroup)
	if _, err := bot.Request(group); err != nil {
		log.Printf("Error sending media group: %v", err)
		return fmt.Errorf("failed to send media group: %w", err)
	}

	return nil
}

func handleDownloadCommand(bot *tgbotapi.BotAPI, msg *tgbotapi.Message) {
	args := strings.TrimSpace(msg.CommandArguments())
	if args == "" {
		bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "❌ Please include a URL after the command.\nExample: `/mp [URL]`"))
		return
	}

	processingMsg, err := bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "⏳ Processing, please wait..."))
	if err != nil {
		log.Printf("Failed to send processing message: %v", err)
		return
	}
	defer deleteMessage(bot, msg.Chat.ID, processingMsg.MessageID)

	start := time.Now()
	fileType := "Video"
	var filePaths []string
	var totalSize int64

	if msg.Command() == "mp" {
		fileType = "Audio"
		filePaths, totalSize, _, err = DownloadAudio(args)
	} else {
		filePaths, totalSize, _, err = DownloadVideo(args)
	}

	if err != nil {
		sendError(bot, msg.Chat.ID, fmt.Sprintf("❌ Failed to download %s: %s", fileType, err.Error()))
		return
	}
	defer CleanupTempFiles(filePaths)

	source := detectSource(args)

	// Send files - totalSize is used if only 1 file
	if len(filePaths) == 1 {
		if err := processAndSendMediaWithMeta(bot, msg, filePaths[0], totalSize, source, start, fileType, args); err != nil {
			log.Printf("Error sending media: %v", err)
		}
	} else {
		// If multiple files, use size per-file
		for _, path := range filePaths {
			fileInfo, err := os.Stat(path)
			if err != nil {
				log.Printf("Error getting file info: %v", err)
				continue
			}

			if err := processAndSendMediaWithMeta(bot, msg, path, fileInfo.Size(), source, start, fileType, args); err != nil {
				log.Printf("Error sending media: %v", err)
			}
		}
	}
}

type mediaSenderFunc func(bot *tgbotapi.BotAPI, msg *tgbotapi.Message, filePath, caption string) error

var mediaSenders = map[string]mediaSenderFunc{
	".jpg":  sendAsPhoto,
	".jpeg": sendAsPhoto,
	".png":  sendAsPhoto,
	".mp4":  sendAsVideo,
	".webm": sendAsVideo,
	".mov":  sendAsVideo,
}

func sendAsPhoto(bot *tgbotapi.BotAPI, msg *tgbotapi.Message, filePath, caption string) error {
	imgFile, err := os.Open(filePath)
	if err != nil {
		log.Printf("Error opening image file: %v. Sending as document.", err)
		return sendAsDocument(bot, msg, filePath, caption)
	}
	defer imgFile.Close()

	img, _, err := image.Decode(imgFile)
	if err != nil {
		log.Printf("Error decoding image: %v. Sending as document.", err)
		return sendAsDocument(bot, msg, filePath, caption)
	}

	reencodedFilePath := filepath.Join(filepath.Dir(filePath), "reencoded_"+filepath.Base(filePath))
	if err := reencodeImage(img, reencodedFilePath, filepath.Ext(filePath)); err != nil {
		log.Printf("Error re-encoding image: %v. Sending original as document.", err)
		return sendAsDocument(bot, msg, filePath, caption)
	}
	defer os.Remove(reencodedFilePath)

	photo := tgbotapi.NewPhoto(msg.Chat.ID, tgbotapi.FilePath(reencodedFilePath))
	photo.Caption = caption
	photo.ParseMode = "MarkdownV2"

	if _, err := bot.Send(photo); err != nil {
		log.Printf("Error sending photo: %v. Falling back to document.", err)
		return sendAsDocument(bot, msg, reencodedFilePath, caption)
	}

	return nil
}

// Helper: Re-encode image
func reencodeImage(img image.Image, outputPath, ext string) error {
	reencodedFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create re-encoded file: %w", err)
	}
	defer reencodedFile.Close()

	switch ext {
	case ".jpg", ".jpeg":
		return jpeg.Encode(reencodedFile, img, &jpeg.Options{Quality: 90})
	case ".png":
		return png.Encode(reencodedFile, img)
	default:
		return fmt.Errorf("unsupported image format: %s", ext)
	}
}

func sendAsVideo(bot *tgbotapi.BotAPI, msg *tgbotapi.Message, filePath, caption string) error {
	video := tgbotapi.NewVideo(msg.Chat.ID, tgbotapi.FilePath(filePath))
	video.Caption = caption
	video.ParseMode = "MarkdownV2"

	if _, err := bot.Send(video); err != nil {
		log.Printf("Error sending video: %v. Falling back to document.", err)
		return sendAsDocument(bot, msg, filePath, caption)
	}

	return nil
}

func sendAsDocument(bot *tgbotapi.BotAPI, msg *tgbotapi.Message, filePath, caption string) error {
	doc := tgbotapi.NewDocument(msg.Chat.ID, tgbotapi.FilePath(filePath))
	doc.Caption = caption
	doc.ParseMode = "MarkdownV2"

	if _, err := bot.Send(doc); err != nil {
		return fmt.Errorf("failed to send file as document: %w", err)
	}

	return nil
}

func processAndSendMediaWithMeta(bot *tgbotapi.BotAPI, msg *tgbotapi.Message, filePath string, fileSize int64, source string, start time.Time, fileType string, url string) error {
	ext := filepath.Ext(filePath)
	duration := time.Since(start).Truncate(time.Second)
	caption := BuildMediaCaption(source, url, fileType, fileSize, duration, GetUserName(msg))

	sender, ok := mediaSenders[ext]
	if !ok {
		return sendAsDocument(bot, msg, filePath, caption)
	}

	return sender(bot, msg, filePath, caption)
}

func handleTikTokAudioCommand(bot *tgbotapi.BotAPI, msg *tgbotapi.Message) {
	url := strings.TrimSpace(msg.CommandArguments())
	if url == "" || !strings.Contains(url, "tiktok.com") {
		bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "❌ Please include a TikTok URL after the command.\nExample: `/tikaudio {URL}`"))
		return
	}

	processingMsg, err := bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "⏳ Processing TikTok audio, please wait..."))
	if err != nil {
		log.Printf("Failed to send processing message: %v", err)
		return
	}
	defer deleteMessage(bot, msg.Chat.ID, processingMsg.MessageID)

	start := time.Now()
	filePath, title, author, err := DownloadTikTokAudio(url)
	if err != nil {
		sendError(bot, msg.Chat.ID, fmt.Sprintf("❌ Failed to download audio: %s", err.Error()))
		return
	}
	defer CleanupTempFiles([]string{filePath})

	fileInfo, err := os.Stat(filePath)
	if err != nil {
		log.Printf("Failed to get file info: %v", err)
		sendError(bot, msg.Chat.ID, "❌ Sorry, failed to get file information.")
		return
	}

	fileSize := fileInfo.Size()
	duration := time.Since(start).Truncate(time.Second)

	audioMsg := tgbotapi.NewAudio(msg.Chat.ID, tgbotapi.FilePath(filePath))
	audioMsg.Caption = BuildMediaCaption("TikTok", url, "Audio", fileSize, duration, GetUserName(msg))
	audioMsg.ParseMode = "MarkdownV2"
	audioMsg.Title = title
	audioMsg.Performer = author

	if _, err := bot.Send(audioMsg); err != nil {
		log.Printf("Failed to send audio to Telegram: %v", err)
		sendError(bot, msg.Chat.ID, "❌ Sorry, failed to send audio file.")
	}
}

func sendError(bot *tgbotapi.BotAPI, chatID int64, message string) {
	if _, err := bot.Send(tgbotapi.NewMessage(chatID, message)); err != nil {
		log.Printf("Failed to send error message: %v", err)
	}
}
