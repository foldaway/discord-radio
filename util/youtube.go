package util

import (
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/masatana/go-textdistance"
	"google.golang.org/api/googleapi/transport"

	"google.golang.org/api/youtube/v3"

	"github.com/bottleneckco/discord-radio/models"
	"github.com/joho/godotenv"
)

var youtubeService *youtube.Service

func init() {
	godotenv.Load()
	var err error
	client := &http.Client{
		Transport: &transport.APIKey{Key: os.Getenv("GOOGLE_API_KEY")},
	}

	youtubeService, err = youtube.New(client)
	if err != nil {
		log.Println(err)
	}
}

// GenerateAutoPlaylistQueueItem get a new item from the auto playlist (with optional parameter for item to ignore)
func GenerateAutoPlaylistQueueItem(ignoreItem *youtube.PlaylistItem) (*youtube.PlaylistItem, error) {
	parsedURI, err := url.ParseRequestURI(os.Getenv("BOT_AUTO_PLAYLIST"))
	if err != nil {
		return nil, err
	}
	var listings []*youtube.PlaylistItem
	var pageToken string
	for {
		youtubeListings, err := youtubeService.
			PlaylistItems.
			List("contentDetails").
			PlaylistId(parsedURI.Query().Get("list")).
			MaxResults(50).
			PageToken(pageToken).
			Do()
		if err != nil {
			return nil, err
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

	for {
		chosenListing = listings[rand.Intn(len(listings))]

		snippetsResp, err := youtubeService.
			Videos.
			List("snippet").
			Id(chosenListing.ContentDetails.VideoId).
			Do()
		if err != nil {
			return nil, err
		}
		if len(snippetsResp.Items) == 0 {
			continue
		}

		if ignoreItem == nil || textdistance.LevenshteinDistance(ignoreItem.Snippet.Title, snippetsResp.Items[0].Snippet.Title) > 20 {
			// Use youtube.Video to populate youtube.PlaylistItemSnippet
			chosenListing.Snippet = &youtube.PlaylistItemSnippet{
				Title:        snippetsResp.Items[0].Snippet.Title,
				ChannelTitle: snippetsResp.Items[0].Snippet.ChannelTitle,
				Thumbnails:   snippetsResp.Items[0].Snippet.Thumbnails,
			}
			break
		}
	}

	log.Printf("[AP] Chosen video '%s' by '%s'\n", chosenListing.Snippet.Title, chosenListing.Snippet.ChannelTitle)

	return chosenListing, nil
}

// ConvertYouTubePlaylistItem convert a YouTube playlist item into a local QueueItem model
func ConvertYouTubePlaylistItem(playlistItem *youtube.PlaylistItem) models.QueueItem {
	return models.QueueItem{
		Title:        playlistItem.Snippet.Title,
		ChannelTitle: playlistItem.Snippet.ChannelTitle,
		Author:       "AutoPlaylist",
		VideoID:      playlistItem.ContentDetails.VideoId,
		Thumbnail:    playlistItem.Snippet.Thumbnails.Default.Url,
	}
}
