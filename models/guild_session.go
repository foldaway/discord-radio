package models

import (
	"fmt"
	"log"
	"os"
	"sync"
	"time"
	"unicode"

	"github.com/bottleneckco/discord-radio/util"
	"github.com/bwmarrin/discordgo"
	"github.com/chrisport/go-lang-detector/langdet"
)

// GuildSession represents a guild voice session
type GuildSession struct {
	GuildID         string
	GuildName       string
	RWMutex         sync.RWMutex
	Queue           []QueueItem // current item = index 0
	VoiceConnection *discordgo.VoiceConnection
	VoiceChannelID  string
	History         []string // Youtube IDs
	MusicPlayer     MusicPlayer
}

var (
	isAutoPlaylistEnabled = len(os.Getenv("BOT_AUTO_PLAYLIST")) > 0
)

// Loop session management loop
func (guildSession *GuildSession) Loop() {
	for {
		if guildSession.VoiceConnection == nil {
			time.Sleep(1 * time.Second)
			return
		}
		if guildSession.MusicPlayer.PlaybackState != PlaybackStateStopped {
			log.Println("[SCP] currently playing something!")
			time.Sleep(1 * time.Second)
			continue
		}
		if len(guildSession.Queue) == 0 && !isAutoPlaylistEnabled {
			log.Println("[SCP] no items in queue")
			time.Sleep(1 * time.Second)
			continue
		}
		if len(guildSession.Queue) == 0 {
			log.Println("[SCP] Getting from auto playlist")
			playlistItem, err := util.GenerateAutoPlaylistQueueItem(guildSession.History)
			if err != nil {
				log.Printf("[SCP] Error generating auto playlist item: %s\n", err)
				time.Sleep(1 * time.Second)
				continue
			}

			// Clear history automatically
			if len(guildSession.History) >= util.GetAutoPlaylistCacheLength() {
				guildSession.History = make([]string, 0)
			}

			queueItem := ConvertYouTubePlaylistItem(playlistItem)
			guildSession.RWMutex.Lock()
			guildSession.Queue = append(guildSession.Queue, queueItem)
			guildSession.RWMutex.Unlock()
		}
		guildSession.RWMutex.RLock()
		var song = guildSession.Queue[0]
		guildSession.RWMutex.RUnlock()

		// Announce music title
		// songTitle := util.SanitiseSongTitleTTS(song.Title)

		detector := langdet.NewDetector()
		clc := langdet.UnicodeRangeLanguageComparator{
			Name:       "zh-TW",
			RangeTable: unicode.Han,
		}
		jlc := langdet.UnicodeRangeLanguageComparator{
			Name:       "ja",
			RangeTable: unicode.Katakana,
		}
		klc := langdet.UnicodeRangeLanguageComparator{
			Name:       "ko",
			RangeTable: unicode.Hangul,
		}
		eng := langdet.UnicodeRangeLanguageComparator{
			Name:       "en",
			RangeTable: unicode.ASCII_Hex_Digit,
		}
		detector.AddLanguageComparators(&clc, &jlc, &klc, &eng)

		// if ttsMsgURL, err := googletts.GetTTSURL(fmt.Sprintf("Music: %s", songTitle), detector.GetLanguages(songTitle)[0].Name); err == nil {
		// 	log.Printf("[PLAYER] Announcing upcoming song title: '%s'\n", songTitle)

		// 	err = guildSession.MusicPlayer.PlayURL(ttsMsgURL)
		// 	if err != nil {
		// 		log.Println("Playback error", err)
		// 	}
		// }
		log.Println("[PLAYER] Playing the actual song data")

		guildSession.History = append(guildSession.History, song.VideoID)

		// NOTE: Only YouTube is supported for now
		var err = guildSession.MusicPlayer.PlayYouTubeVideo(fmt.Sprintf("https://www.youtube.com/watch?v=%s", song.VideoID))
		if err != nil {
			log.Println("Playback error", err)
		}

		guildSession.RWMutex.Lock()
		if len(guildSession.Queue) > 0 {
			guildSession.Queue = guildSession.Queue[1:]
		}
		guildSession.RWMutex.Unlock()
	}
}
