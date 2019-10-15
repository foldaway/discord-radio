package commands

import (
	"fmt"
	"log"
	"os"
	"regexp"
	"strconv"

	"github.com/evalphobia/google-tts-go/googletts"

	"github.com/bottleneckco/discord-radio/models"
	"github.com/bottleneckco/discord-radio/util"
)

func SafeCheckPlay(guildSession *models.GuildSession) {
	if guildSession.VoiceConnection == nil {
		log.Println("[SCP] no voice connection")
		return
	}
	if guildSession.MusicPlayer.IsPlaying {
		log.Println("[SCP] currently playing something!")
		return
	}
	if len(guildSession.Queue) == 0 && len(os.Getenv("BOT_AUTO_PLAYLIST")) == 0 {
		log.Println("[SCP] no items in queue")
		return
	} else if len(guildSession.Queue) == 0 {
		log.Println("[SCP] Getting from auto playlist")
		playlistItem, err := util.GenerateAutoPlaylistQueueItem(guildSession.PreviousAutoPlaylistListing)
		if err != nil {
			log.Printf("[SCP] Error generating auto playlist item: %s\n", err)
			return
		}
		queueItem := util.ConvertYouTubePlaylistItem(playlistItem)
		guildSession.Mutex.Lock()
		guildSession.Queue = append(guildSession.Queue, queueItem)
		guildSession.Mutex.Unlock()
	}
	guildSession.Mutex.Lock()
	var song = guildSession.Queue[0]
	guildSession.Mutex.Unlock()

	if ttsMsgURL, err := googletts.GetTTSURL(fmt.Sprintf("Music: %s", sanitiseSongTitle(song.Title)), "en"); err == nil {
		log.Println("[PLAYER] Announcing upcoming song title")
		guildSession.PlayURL(ttsMsgURL, 0.5)
	}
	log.Println("[PLAYER] Playing the actual song data")
	volume := 0.5
	volumeConv, err := strconv.ParseFloat(os.Getenv("BOT_VOLUME"), 64)
	if err == nil {
		volume = volumeConv
	}
	guildSession.PlayYouTube(fmt.Sprintf("https://www.youtube.com/watch?v=%s", song.VideoID), volume)
	guildSession.Mutex.Lock()
	if len(guildSession.Queue) > 0 {
		guildSession.Queue = guildSession.Queue[1:]
	}
	guildSession.Mutex.Unlock()
	if guildSession.VoiceConnection != nil {
		go SafeCheckPlay(guildSession)
	}
}

func sanitiseSongTitle(title string) string {
	parenthesisRegex := regexp.MustCompile(`(\(.+?\)|\[.+?\])`)
	alphabetNumberOnly := regexp.MustCompile(`[^a-zA-Z0-9\s&]+`)
	bannedWordsRegex := regexp.MustCompile(`(official|music video|special video|lyric video)`)
	return alphabetNumberOnly.ReplaceAllString(bannedWordsRegex.ReplaceAllString(parenthesisRegex.ReplaceAllString(title, ""), ""), "")
}
