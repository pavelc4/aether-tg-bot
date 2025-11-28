package utils

import (
	"fmt"
	"strings"
)

const (
	progressBarFilled = "█"
	progressBarEmpty  = "░"
	progressBarLength = 20
)

var markdownV2Replacer = strings.NewReplacer(
	"_", "\\_", "*", "\\*", "[", "\\[", "]", "\\]",
	"(", "\\(", ")", "\\)", "~", "\\~", "`", "\\`",
	">", "\\>", "#", "\\#", "+", "\\+", "-", "\\-",
	"=", "\\=", "|", "\\|", "{", "\\{", "}", "\\}",
	".", "\\.", "!", "\\!",
)

func FormatFileSize(size int64) string {
	const unit = 1024
	if size < unit {
		return fmt.Sprintf("%d B", size)
	}

	div, exp := float64(unit), 0
	for n := size / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}

	units := []string{"KB", "MB", "GB", "TB"}
	return fmt.Sprintf("%.2f %s", float64(size)/div, units[exp])
}

func FormatDuration(s uint64) string {
	d := s / 86400
	h := (s % 86400) / 3600
	m := (s % 3600) / 60
	ss := s % 60

	switch {
	case d > 0:
		return fmt.Sprintf("%dd %dh %dm %ds", d, h, m, ss)
	case h > 0:
		return fmt.Sprintf("%dh %dm %ds", h, m, ss)
	case m > 0:
		return fmt.Sprintf("%dm %ds", m, ss)
	default:
		return fmt.Sprintf("%ds", ss)
	}
}

func EscapeMarkdownV2(s string) string {
	return markdownV2Replacer.Replace(s)
}

func FormatProgressBar(p float64) string {
	if p < 0 {
		p = 0
	}
	if p > 100 {
		p = 100
	}

	filled := int(p / 100 * progressBarLength)
	empty := progressBarLength - filled

	return fmt.Sprintf(
		"%s%s %.1f%%",
		strings.Repeat(progressBarFilled, filled),
		strings.Repeat(progressBarEmpty, empty),
		p,
	)
}
