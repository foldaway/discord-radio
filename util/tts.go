package util

import (
	"regexp"
	"strings"
)

var (
	// Special cases
	regExes = []*regexp.Regexp{
		regexp.MustCompile(`(?i)Han.+?Rom.+?Eng`),
		regexp.MustCompile(`(?i)music video`),
		regexp.MustCompile(`(?i)special video`),
		regexp.MustCompile(`(?i)lyric video`),
		regexp.MustCompile(`(?i)official video`),
		regexp.MustCompile(`(?i)official audio`),
		regexp.MustCompile(`(?i)color coded lyrics`),
		regexp.MustCompile(`(?i)colour coded lyrics`),
		regexp.MustCompile(`(?i)1080p`),
		regexp.MustCompile(`(?i)720p`),
		regexp.MustCompile(`(?i)official`),
		regexp.MustCompile(`(?i)(Audio)`),
		regexp.MustCompile(`(?i)(Lyrics)`),
		regexp.MustCompile(`(?i)(Lyric)`),
	}

	// Case sensitive replacement
	bannedTerms = []string{
		"官方",
		"完整版",
		"MV",
		"M/V",

		// Remove empty brackets
		"()",
		"[]",
	}
)

// SanitiseSongTitleTTS process a song title for TTS reading
func SanitiseSongTitleTTS(title string) string {
	var result = title
	for _, rgx := range regExes {
		result = rgx.ReplaceAllString(result, "")
	}
	for _, bannedTerm := range bannedTerms {
		result = strings.ReplaceAll(result, bannedTerm, "")
	}
	return result
}
