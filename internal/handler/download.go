package handler

import (
	"context"
	"fmt"
	"html"
	"regexp"
	"strings"
	"time"
	"unicode/utf16"

	"github.com/gotd/td/telegram/message"
	"github.com/gotd/td/tg"

	"github.com/pavelc4/aether-tg-bot/internal/provider"
	"github.com/pavelc4/aether-tg-bot/internal/stats"
	"github.com/pavelc4/aether-tg-bot/internal/streaming"
	"github.com/pavelc4/aether-tg-bot/internal/telegram"
	"github.com/pavelc4/aether-tg-bot/pkg/logger"
)

const (
	MaxAlbumSize = 10
)

type DownloadHandler struct {
	streamMgr *streaming.Manager
	client    *telegram.Client
}

func NewDownloadHandler(sm *streaming.Manager, cli *telegram.Client) *DownloadHandler {
	return &DownloadHandler{
		streamMgr: sm,
		client:    cli,
	}
}

func (h *DownloadHandler) Handle(ctx context.Context, e tg.Entities, msg *tg.Message, url string) error {
	api := h.client.API()
	
	inputPeer, err := resolvePeer(msg.PeerID, e)
	if err != nil {
		return fmt.Errorf("failed to resolve peer: %w", err)
	}

	sender := message.NewSender(api)
	b := sender.To(inputPeer).Reply(msg.ID)

	sentUpdates, err := b.Text(ctx, "🔎 Detecting...")
	if err != nil {
		return fmt.Errorf("send message failed: %w", err)
	}
	
	sentMsgID := getMsgID(sentUpdates)

	editMsg := func(text string) {
		_, err := sender.To(inputPeer).Edit(sentMsgID).Text(ctx, text)
		if err != nil {
			logger.Error("Failed to edit message", "msg_id", sentMsgID, "error", err)
		}
	}

	infos, providerName, err := provider.Resolve(ctx, url)
	if err != nil {
		editMsg(fmt.Sprintf("❌ Failed from %s: %v", providerName, err))
		return err
	}

	editMsg(fmt.Sprintf("🚀 starting download from %s (found %d items)", providerName, len(infos)))

	uploader := telegram.NewUploader(api)
	startTime := time.Now()

	var album []tg.InputMediaClass
	var uploadedInfos []provider.VideoInfo

	for i, info := range infos {
		if len(infos) > 1 {
			editMsg(fmt.Sprintf("📂 processing item %d/%d: %s", i+1, len(infos), info.FileName))
		}

		input := streaming.StreamInput{
			URL:      info.URL,
			Filename: info.FileName,
			Size:     info.FileSize,
			Headers:  info.Headers,
			MIME:     info.MimeType,
			Duration: info.Duration,
			Width:    info.Width,
			Height:   info.Height,
		}

		fileID := time.Now().UnixNano() + int64(i)
		
		uploadFn := func(ctx context.Context, chunk streaming.Chunk, _ int64) error {
			return uploader.UploadChunk(ctx, chunk, fileID)
		}
		
		var actualParts int
		actualParts, err = h.streamMgr.Stream(ctx, input, uploadFn, func(read, total int64) {})
		
		if err != nil {
			logger.Error("Failed to stream item", "index", i, "error", err)
			continue
		}

		if actualParts > 0 {
			media := h.createInputMedia(input, fileID, actualParts)
			if media != nil {
				album = append(album, media)
				uploadedInfos = append(uploadedInfos, info)
			}
		}
	}

	if len(album) == 0 {
		editMsg("❌ No items were successfully downloaded.")
		return nil
	}

	logger.Info("Starting batch send", "total_items", len(album))
	
	for i := 0; i < len(album); i += MaxAlbumSize {
		end := i + MaxAlbumSize
		if end > len(album) {
			end = len(album)
		}

		batch := album[i:end]
		batchInfos := uploadedInfos[i:end]
		
		logger.Info("Sending batch", "start", i, "end", end, "count", len(batch))
		
		userName := getUserName(e, msg)
		captionHTML := h.buildCaption(batchInfos[0], providerName, time.Since(startTime), url, userName)
		captionText, entities := h.parseCaptionEntities(captionHTML)

		if len(batch) == 1 {
			_, err := api.MessagesSendMedia(ctx, &tg.MessagesSendMediaRequest{
				Peer:     inputPeer,
				ReplyTo:  &tg.InputReplyToMessage{ReplyToMsgID: msg.ID},
				Media:    batch[0],
				Message:  captionText,
				Entities: entities,
				RandomID: time.Now().UnixNano(),
			})
			if err != nil {
				logger.Error("Failed to send single media", "error", err)
				editMsg(fmt.Sprintf("❌ Upload Error (Single): %v", err))
			}
		} else {
			var multiMedia []tg.InputSingleMedia
			for j, media := range batch {
				single := tg.InputSingleMedia{
					RandomID: time.Now().UnixNano() + int64(j),
					Media:    media,
				}
				if j == 0 {
					single.Message = captionText
					single.Entities = entities
				}
				multiMedia = append(multiMedia, single)
			}

			_, err := api.MessagesSendMultiMedia(ctx, &tg.MessagesSendMultiMediaRequest{
				Peer:       inputPeer,
				ReplyTo:    &tg.InputReplyToMessage{ReplyToMsgID: msg.ID},
				MultiMedia: multiMedia,
			})
			
			if err != nil {
				logger.Error("Failed to send album batch", "error", err)
				editMsg(fmt.Sprintf("❌ Upload Error (Batch %d): %v", i/MaxAlbumSize+1, err))
			}
		}
		
		if end < len(album) {
			time.Sleep(1 * time.Second)
		}
	}

	if sentMsgID != 0 {
		api.MessagesDeleteMessages(ctx, &tg.MessagesDeleteMessagesRequest{
			ID: []int{sentMsgID},
			Revoke: true,
		})
	}
	
	if len(infos) > 1 {
	} 

	stats.TrackDownload()
	return nil
}

func (h *DownloadHandler) createInputMedia(input streaming.StreamInput, fileID int64, parts int) tg.InputMediaClass {
	inputFile := &tg.InputFileBig{
		ID:    fileID,
		Parts: parts,
		Name:  input.Filename,
	}

	mime := input.MIME
	if mime == "" {
		mime = "video/mp4" 
	}

	if strings.HasPrefix(mime, "image/") {
		return &tg.InputMediaUploadedPhoto{
			File: inputFile,
		}
	} else {
		w, h := input.Width, input.Height
		if w == 0 || h == 0 {
			w, h = 1280, 720
			logger.Warn("Missing video dimensions, using default", "file", input.Filename)
		}

		logger.Info("Creating video attributes", 
			"file", input.Filename,
			"w", w, "h", h, 
			"dur", input.Duration,
		)

		return &tg.InputMediaUploadedDocument{
			File: inputFile,
			MimeType: mime,
			Attributes: []tg.DocumentAttributeClass{
				&tg.DocumentAttributeVideo{
					SupportsStreaming: true,
					Duration:          float64(input.Duration),
					W:                 w,
					H:                 h,
				},
				&tg.DocumentAttributeFilename{
					FileName: input.Filename,
				},
			},
		}
	}
}


func (h *DownloadHandler) buildCaption(info provider.VideoInfo, providerName string, duration time.Duration, sourceURL string, userName string) string {
	sizeMB := float64(info.FileSize) / 1024 / 1024
	
	const MaxCaptionLen = 1024
	
	cleanTitle := html.UnescapeString(info.Title)
	
	displayTitle := cleanTitle
	if len(displayTitle) > 100 {
		displayTitle = displayTitle[:97] + "..."
	}
	
	safeTitle := html.EscapeString(displayTitle)
	safeProvider := html.EscapeString(providerName)
	safeUser := html.EscapeString(userName)
	
	srcLink := fmt.Sprintf(`<a href="%s">%s</a>`, sourceURL, safeProvider)
	
	baseText := fmt.Sprintf("<b>%s</b>\n"+
		"🔗 Source : %s\n"+
		"💾 Size : <code>%.2f MB</code>\n"+
		"⏱️ Processing Time : <code>%s</code>\n"+
		"👤 By : %s", 
		safeTitle,
		srcLink,
		sizeMB,
		duration.Round(time.Second),
		safeUser,
	)

	return baseText
}

func (h *DownloadHandler) parseCaptionEntities(text string) (string, []tg.MessageEntityClass) {
	re := regexp.MustCompile(`(?s)<(b|code|a)(?: href="([^"]+)")?>([^<]+)</(?:b|code|a)>`)
	
	var cleanText strings.Builder
	var entities []tg.MessageEntityClass
	
	matches := re.FindAllStringSubmatchIndex(text, -1)
	
	lastIdx := 0
	for _, m := range matches {
		pre := text[lastIdx:m[0]]
		cleanText.WriteString(pre)
		
		offset := len(utf16.Encode([]rune(cleanText.String())))
		tagEnd := m[1]
		
		tagName := text[m[2]:m[3]]
		href := ""
		if m[4] != -1 {
			href = text[m[4]:m[5]]
		}
		content := text[m[6]:m[7]]
		
		cleanText.WriteString(content)
		length := len(utf16.Encode([]rune(content)))
		
		var ent tg.MessageEntityClass
		switch tagName {
		case "b":
			ent = &tg.MessageEntityBold{Offset: offset, Length: length}
		case "code":
			ent = &tg.MessageEntityCode{Offset: offset, Length: length}
		case "a":
			ent = &tg.MessageEntityTextURL{Offset: offset, Length: length, URL: href}
		}
		
		if ent != nil {
			entities = append(entities, ent)
		}
		
		lastIdx = tagEnd
	}
	
	cleanText.WriteString(text[lastIdx:])
	
	return cleanText.String(), entities
}

func getMsgID(updates tg.UpdatesClass) int {
	switch u := updates.(type) {
	case *tg.UpdateShortSentMessage:
		return u.ID
	case *tg.Updates:
		for _, update := range u.Updates {
			if msg, ok := update.(*tg.UpdateNewMessage); ok {
				if m, ok := msg.Message.(*tg.Message); ok {
					return m.ID
				}
			}
			if msg, ok := update.(*tg.UpdateNewChannelMessage); ok {
				if m, ok := msg.Message.(*tg.Message); ok {
					return m.ID
				}
			}
		}
	}
	return 0
}

func getUserName(e tg.Entities, msg *tg.Message) string {
	var userID int64
	if from, ok := msg.GetFromID(); ok {
		if u, ok := from.(*tg.PeerUser); ok {
			userID = u.UserID
		}
	} else {
		if u, ok := msg.GetPeerID().(*tg.PeerUser); ok {
			userID = u.UserID
		}
	}

	if userID != 0 {
		if user, ok := e.Users[userID]; ok {
			if user.Username != "" {
				return "@" + user.Username
			}
			name := strings.TrimSpace(user.FirstName + " " + user.LastName)
			if name != "" {
				return name
			}
		}
	}
	return "User"
}
