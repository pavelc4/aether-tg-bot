package telegram

import (
	"fmt"
	"time"
)

func FormatStart() string {
	return " **Welcome to Aether!**\n\nSend me a link from TikTok, Cobalt, or YouTube, and I'll stream it to you directly!"
}

func FormatError(err error) string {
	return fmt.Sprintf(" **Error:** %v", err)
}

func FormatSuccess(filename string, size int64, duration time.Duration) string {
	return fmt.Sprintf(" **Completed**\nFile: `%s`\nSize: %s\nTime: %v", filename, formatBytes(uint64(size)), duration)
}
