package commands

import (
	"fmt"
	"log"
	"math/rand"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"time"

	"github.com/evalphobia/google-tts-go/googletts"
	"github.com/masatana/go-textdistance"

	"google.golang.org/api/youtube/v3"

	"github.com/bottleneckco/discord-radio/models"
)

func GenerateAutoPlaylistQueueItem(guildSession *models.GuildSession) (models.QueueItem, error) {
	var data models.QueueItem
	parsedURI, err := url.ParseRequestURI(os.Getenv("BOT_AUTO_PLAYLIST"))
	if err != nil {
		return data, err
	}
	var listings []*youtube.PlaylistItem
	var pageToken string
	for {
		youtubeListings, err := youtubeService.PlaylistItems.List("contentDetails").PlaylistId(parsedURI.Query().Get("list")).MaxResults(50).PageToken(pageToken).Do()
		if err != nil {
			return data, err
		}
		listings = append(listings, youtubeListings.Items...)
		pageToken = youtubeListings.NextPageToken
		if len(pageToken) == 0 {
			break
		}
	}

	log.Printf("[AP] Fetched %d items\n", len(listings))

	rand.Seed(time.Now().Unix())

	var chosenListing *youtube.PlaylistItem
	var chosenListingSnippet *youtube.Video

	for {
		chosenListing = listings[rand.Intn(len(listings))]
		chosenListingSnippets, err := youtubeService.Videos.List("snippet").Id(chosenListing.ContentDetails.VideoId).Do()
		if err != nil {
			return data, err
		}
		if len(chosenListingSnippets.Items) == 0 {
			continue
		}
		chosenListingSnippet = chosenListingSnippets.Items[0]

		if guildSession.PreviousAutoPlaylistListing != nil && textdistance.LevenshteinDistance(guildSession.PreviousAutoPlaylistListing.Snippet.Title, chosenListingSnippet.Snippet.Title) > 20 {
			guildSession.PreviousAutoPlaylistListing = chosenListing
			guildSession.PreviousAutoPlaylistListing.Snippet = &youtube.PlaylistItemSnippet{Title: chosenListingSnippet.Snippet.Title}
			break
		} else {
			break
		}
	}

	log.Printf("[AP] Chosen video '%s' by '%s'\n", chosenListingSnippet.Snippet.Title, chosenListingSnippet.Snippet.ChannelTitle)

	data = models.QueueItem{
		Title:        chosenListingSnippet.Snippet.Title,
		ChannelTitle: chosenListingSnippet.Snippet.ChannelTitle,
		Author:       "AutoPlaylist",
		VideoID:      chosenListing.ContentDetails.VideoId,
		Thumbnail:    chosenListingSnippet.Snippet.Thumbnails.Default.Url,
	}
	return data, nil
}

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
		queueItem, err := GenerateAutoPlaylistQueueItem(guildSession)
		if err != nil {
			log.Printf("[SCP] Error generating auto playlist item: %s\n", err)
			return
		}
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
