package util

import (
	"fmt"
	"regexp"
	"strings"
)

var (
	ttsBannedWords = []string{
		"official",
		"music video",
		"special video",
		"lyric video",
		"M/?V",
		"colou?r coded lyrics",
		"1080p",
		"720p",
	}
	ttsBannedWordsRegex = regexp.MustCompile(fmt.Sprintf("(?i)(%s)", strings.Join(ttsBannedWords, "|")))
)

// SanitiseSongTitleTTS process a song title for TTS reading
func SanitiseSongTitleTTS(title string) string {
	return ttsBannedWordsRegex.
		ReplaceAllString(
			title,
			"",
		)
}
