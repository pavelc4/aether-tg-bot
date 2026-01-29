package messaging

import (
	"fmt"
	"html"
	"regexp"
	"strings"
	"time"
	"unicode/utf16"

	"github.com/gotd/td/tg"
	"github.com/pavelc4/aether-tg-bot/internal/provider"
)

func BuildCaption(info provider.VideoInfo, providerName string, duration time.Duration, sourceURL string, userName string) string {
	sizeMB := float64(info.FileSize) / 1024 / 1024

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

func ParseCaptionEntities(text string) (string, []tg.MessageEntityClass) {
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

func GetUserName(e tg.Entities, msg *tg.Message) string {
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
