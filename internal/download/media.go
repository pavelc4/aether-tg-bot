package download

import (
	"strings"

	"github.com/gotd/td/tg"
	"github.com/pavelc4/aether-tg-bot/internal/streaming"
	"github.com/pavelc4/aether-tg-bot/pkg/logger"
)

// CreateInputMedia creates a tg.InputMediaClass based on the stream input and file details.
func CreateInputMedia(input streaming.StreamInput, fileID int64, parts int, isBig bool, md5sum string, isAudio bool) tg.InputMediaClass {
	var inputFile tg.InputFileClass
	var mime string

	if isBig {
		inputFile = &tg.InputFileBig{
			ID:    fileID,
			Parts: parts,
			Name:  input.Filename,
		}
	} else {
		inputFile = &tg.InputFile{
			ID:          fileID,
			Parts:       parts,
			Name:        input.Filename,
			MD5Checksum: md5sum,
		}
	}

	mime = input.MIME
	if mime == "" {
		mime = "video/mp4"
	}

	isMimeAudio := strings.HasPrefix(mime, "audio/")
	isExtAudio := strings.HasSuffix(input.Filename, ".mp3") || strings.HasSuffix(input.Filename, ".m4a") || strings.HasSuffix(input.Filename, ".ogg")

	if (isAudio || isExtAudio) && !isMimeAudio {
		if strings.HasSuffix(input.Filename, ".m4a") {
			mime = "audio/mp4"
		} else if strings.HasSuffix(input.Filename, ".mp3") {
			mime = "audio/mpeg"
		} else {
			mime = "audio/mpeg" // Fallback
		}
	}

	logger.Info("Creating media input",
		"file", input.Filename,
		"mime", mime,
		"parts", parts,
		"isBig", isBig,
		"isAudio", isAudio,
	)

	isPhoto := strings.HasPrefix(mime, "image/") ||
		strings.HasSuffix(strings.ToLower(input.Filename), ".jpg") ||
		strings.HasSuffix(strings.ToLower(input.Filename), ".jpeg") ||
		strings.HasSuffix(strings.ToLower(input.Filename), ".png") ||
		strings.HasSuffix(strings.ToLower(input.Filename), ".webp") ||
		strings.HasSuffix(strings.ToLower(input.Filename), ".gif")

	if isPhoto {
		if isBig {
			logger.Error("CRITICAL: Photo cannot use InputFileBig!", "file", input.Filename)
			return nil
		}

		logger.Info("Creating InputMediaUploadedPhoto", "file", input.Filename)
		return &tg.InputMediaUploadedPhoto{
			File: inputFile, // Must be InputFile (not InputFileBig)
		}
	}

	if strings.HasPrefix(mime, "audio/") || isAudio {
		logger.Info("Creating audio document", "file", input.Filename)
		return &tg.InputMediaUploadedDocument{
			File:     inputFile,
			MimeType: mime,
			Attributes: []tg.DocumentAttributeClass{
				&tg.DocumentAttributeAudio{
					Duration:  int(input.Duration),
					Title:     input.Filename,
					Performer: "AetherBot",
				},
				&tg.DocumentAttributeFilename{
					FileName: input.Filename,
				},
			},
		}
	}

	// Video files
	if strings.HasPrefix(mime, "video/") {
		w, h := input.Width, input.Height
		if w == 0 || h == 0 {
			w, h = 1280, 720
			logger.Warn("Missing video dimensions, using default", "file", input.Filename)
		}

		logger.Info("Creating video document",
			"file", input.Filename,
			"w", w, "h", h,
			"dur", input.Duration,
		)

		return &tg.InputMediaUploadedDocument{
			File:     inputFile,
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

	// Default: generic document
	logger.Info("Creating generic document", "file", input.Filename)
	return &tg.InputMediaUploadedDocument{
		File:     inputFile,
		MimeType: mime,
		Attributes: []tg.DocumentAttributeClass{
			&tg.DocumentAttributeFilename{
				FileName: input.Filename,
			},
		},
	}
}
