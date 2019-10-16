package util

import "regexp"

// SanitiseSongTitleTTS process a song title for TTS reading
func SanitiseSongTitleTTS(title string) string {
	parenthesisRegex := regexp.MustCompile(`(\(.+?\)|\[.+?\])`)
	// alphabetNumberOnly := regexp.MustCompile(`[^a-zA-Z0-9\s&]+`)
	bannedWordsRegex := regexp.MustCompile(`(official|music video|special video|lyric video)`)
	return bannedWordsRegex.ReplaceAllString(parenthesisRegex.ReplaceAllString(title, ""), "")
}
