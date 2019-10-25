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
		"official video",
		"M/?V",
		"colou?r coded lyrics",
		"1080p",
		"720p",
		"(Audio)",
	}
)

func ttsBannedWordsRegex() *regexp.Regexp {
	var sanitisedWords []string
	for _, word := range ttsBannedWords {
		sanitisedWords = append(sanitisedWords, regexp.QuoteMeta(word))
	}
	return regexp.MustCompile(fmt.Sprintf("(?i)(%s)", strings.Join(sanitisedWords, "|")))
}

// SanitiseSongTitleTTS process a song title for TTS reading
func SanitiseSongTitleTTS(title string) string {
	return ttsBannedWordsRegex().
		ReplaceAllString(
			title,
			"",
		)
}
