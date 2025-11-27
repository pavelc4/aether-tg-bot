package providers

import "regexp"

var (
	unsafeCharsRegex = regexp.MustCompile(`[<>:"/\\|?*\x00-\x1f]`)
	spacesRegex      = regexp.MustCompile(`\s+`)
)
