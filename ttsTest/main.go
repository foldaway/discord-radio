package main

import (
	"log"
	"unicode"

	"github.com/bottleneckco/discord-radio/util"

	"github.com/chrisport/go-lang-detector/langdet"
)

func main() {
	text := util.SanitiseSongTitleTTS("周杰倫Jay Chou X aMEI【不該 Shouldn't Be】Official MV")

	detector := langdet.NewDetector()
	clc := langdet.UnicodeRangeLanguageComparator{"zh-TW", unicode.Han}
	jlc := langdet.UnicodeRangeLanguageComparator{"ja", unicode.Katakana}
	klc := langdet.UnicodeRangeLanguageComparator{"ko", unicode.Hangul}
	eng := langdet.UnicodeRangeLanguageComparator{"en", unicode.ASCII_Hex_Digit}
	detector.AddLanguageComparators(&clc, &jlc, &klc, &eng)

	log.Printf("'%s' = %+v\n", text, detector.GetLanguages(text))
}
