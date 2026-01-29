package messaging

import (
	"context"
	"fmt"
	"time"

	"github.com/gotd/td/tg"
	"github.com/pavelc4/aether-tg-bot/internal/provider"
	"github.com/pavelc4/aether-tg-bot/pkg/logger"
)

type Sender struct {
	api *tg.Client
}

func NewSender(api *tg.Client) *Sender {
	return &Sender{api: api}
}

// SendSingle sends a single media item with caption.
func (s *Sender) SendSingle(ctx context.Context, peer tg.InputPeerClass, replyTo tg.InputReplyToClass, media tg.InputMediaClass, info provider.VideoInfo, providerName string, startTime time.Time, url string, userName string) (tg.UpdatesClass, error) {
	captionHTML := BuildCaption(info, providerName, time.Since(startTime), url, userName)
	captionText, entities := ParseCaptionEntities(captionHTML)

	updates, err := s.api.MessagesSendMedia(ctx, &tg.MessagesSendMediaRequest{
		Peer:     peer,
		ReplyTo:  replyTo,
		Media:    media,
		Message:  captionText,
		Entities: entities,
		RandomID: time.Now().UnixNano(),
	})

	if err != nil {
		return nil, fmt.Errorf("failed to send single media: %w", err)
	}
	return updates, nil
}

func (s *Sender) SendAlbum(ctx context.Context, peer tg.InputPeerClass, replyTo tg.InputReplyToClass, batch []tg.InputMediaClass, batchInfos []provider.VideoInfo, providerName string, startTime time.Time, url string, userName string, albumIndex int, totalAlbumLen int, offset int) error {
	prepareAlbum := func(batch []tg.InputMediaClass) ([]tg.InputSingleMedia, error) {
		return s.prepareAlbumHelper(ctx, batch)
	}

	multiMedia, err := prepareAlbum(batch)
	if err != nil {
		logger.Error("Failed to prepare album", "error", err)
	}

	if len(multiMedia) == len(batch) {
		end := offset + len(batch)
		isLastBatch := (end == totalAlbumLen)

		// Add caption to the last item of the batch if it's the last batch of the album
		if isLastBatch {
			lastIdx := len(multiMedia) - 1
			captionHTML := BuildCaption(batchInfos[lastIdx], providerName, time.Since(startTime), url, userName)
			captionText, entities := ParseCaptionEntities(captionHTML)
			multiMedia[lastIdx].Message = captionText
			multiMedia[lastIdx].Entities = entities
		}

		_, err = s.api.MessagesSendMultiMedia(ctx, &tg.MessagesSendMultiMediaRequest{
			Peer:       peer,
			ReplyTo:    replyTo,
			MultiMedia: multiMedia,
		})
	} else {
		if err == nil {
			err = fmt.Errorf("failed to prepare all media items")
		}
	}

	if err != nil {
		logger.Error("Failed to send album batch, trying individual sends",
			"error", err.Error(),
			"batch_size", len(batch),
		)

		for j, media := range batch {
			isLastImage := (offset + j == totalAlbumLen-1)
			
			var singleCaptionText string
			var singleEntities []tg.MessageEntityClass
			
			if isLastImage {
				captionHTML := BuildCaption(batchInfos[j], providerName, time.Since(startTime), url, userName)
				singleCaptionText, singleEntities = ParseCaptionEntities(captionHTML)
			}

			var singleReplyTo tg.InputReplyToClass
			if albumIndex == 0 && j == 0 {
				singleReplyTo = replyTo
			}

			_, singleErr := s.api.MessagesSendMedia(ctx, &tg.MessagesSendMediaRequest{
				Peer:     peer,
				ReplyTo:  singleReplyTo,
				Media:    media,
				Message:  singleCaptionText,
				Entities: singleEntities,
				RandomID: time.Now().UnixNano() + int64(j),
			})

			if singleErr != nil {
				logger.Error("Failed to send single image fallback", "index", j, "error", singleErr)
			} else {
				logger.Info("Successfully sent single image fallback", "index", j)
			}

			if j < len(batch)-1 {
				time.Sleep(500 * time.Millisecond)
			}
		}
	} else {
		logger.Info("Successfully sent album", "items", len(batch))
	}

	return nil
}

func (s *Sender) prepareAlbumHelper(ctx context.Context, batch []tg.InputMediaClass) ([]tg.InputSingleMedia, error) {
	multiMedia := make([]tg.InputSingleMedia, len(batch))
	type resType struct {
		idx   int
		media tg.InputSingleMedia
		err   error
	}
	resChan := make(chan resType, len(batch))

	for i, media := range batch {
		go func(idx int, m tg.InputMediaClass) {
			res, err := s.api.MessagesUploadMedia(ctx, &tg.MessagesUploadMediaRequest{
				Peer:  &tg.InputPeerSelf{},
				Media: m,	
			})
			if err != nil {
				resChan <- resType{idx: idx, err: err}
				return
			}

			var inputMedia tg.InputMediaClass
			switch m := res.(type) {
			case *tg.MessageMediaPhoto:
				if photo, ok := m.Photo.(*tg.Photo); ok {
					inputMedia = &tg.InputMediaPhoto{
						ID: &tg.InputPhoto{
							ID:            photo.ID,
							AccessHash:    photo.AccessHash,
							FileReference: photo.FileReference,
						},
					}
				}
			case *tg.MessageMediaDocument:
				if doc, ok := m.Document.(*tg.Document); ok {
					inputMedia = &tg.InputMediaDocument{
						ID: &tg.InputDocument{
							ID:            doc.ID,
							AccessHash:    doc.AccessHash,
							FileReference: doc.FileReference,
						},
					}
				}
			}

			if inputMedia == nil {
				resChan <- resType{idx: idx, err: fmt.Errorf("failed to get persistent media")}
				return
			}

			resChan <- resType{
				idx: idx,
				media: tg.InputSingleMedia{
					RandomID: time.Now().UnixNano() + int64(idx),
					Media:    inputMedia,
				},
			}
		}(i, media)
	}

	for i := 0; i < len(batch); i++ {
		r := <-resChan
		if r.err != nil {
			return nil, r.err
		}
		multiMedia[r.idx] = r.media
	}
	
	return multiMedia, nil
}
