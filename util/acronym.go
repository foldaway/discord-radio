package util

import (
	"strings"
)

// GenerateAcronym generate acronym from a sentence
func GenerateAcronym(text string) string {
	words := strings.Split(text, " ")

	res := ""

	for _, word := range words {
		res = res + string([]rune(word)[0])
	}

	return res
}
