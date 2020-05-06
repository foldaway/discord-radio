package util

import (
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/go-co-op/gocron"
	"github.com/masatana/go-textdistance"
	"google.golang.org/api/googleapi/transport"

	"google.golang.org/api/youtube/v3"

	"github.com/joho/godotenv"
)

var (
	youtubeService        *youtube.Service
	autoPlaylistItemCache = make([]*youtube.PlaylistItem, 0)
)

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

	var scheduler = gocron.NewScheduler(time.Local)
	scheduler.Every(2).Hours().StartImmediately().StartAt(time.Now().Add(time.Second * 2)).Do(cacheAutoPlaylistItems)

	scheduler.Start()
}

func cacheAutoPlaylistItems() {
	var envURL = os.Getenv("BOT_AUTO_PLAYLIST")
	if len(envURL) == 0 {
		return
	}

	log.Println("[AP] Caching items...")

	var autoPlaylistURL *url.URL
	var err error

	autoPlaylistURL, err = url.ParseRequestURI(envURL)
	if err != nil {
		log.Println(err)
		return
	}

	autoPlaylistItemCache, err = FetchAllPlaylistItems(autoPlaylistURL)
	if err != nil {
		log.Println(err)
		return
	}

	log.Printf("[AP] Cached %d items\n", len(autoPlaylistItemCache))
}

// FetchAllPlaylistItems get all the items of a playlist
func FetchAllPlaylistItems(playlistURL *url.URL) ([]*youtube.PlaylistItem, error) {
	var listings []*youtube.PlaylistItem
	var pageToken string
	for {
		youtubeListings, err := youtubeService.
			PlaylistItems.
			List("contentDetails").
			PlaylistId(playlistURL.Query().Get("list")).
			MaxResults(50).
			PageToken(pageToken).
			Do()

		if err != nil {
			return listings, err
		}

		listings = append(listings, youtubeListings.Items...)
		pageToken = youtubeListings.NextPageToken
		if len(pageToken) == 0 {
			break
		}
	}

	return listings, nil
}

// GenerateAutoPlaylistQueueItem get a new item from the auto playlist (with optional parameter for item to ignore)
func GenerateAutoPlaylistQueueItem(ignoreItem *youtube.PlaylistItem) (*youtube.PlaylistItem, error) {
	rand.Seed(time.Now().Unix())

	var chosenListing *youtube.PlaylistItem

	if len(autoPlaylistItemCache) == 0 {
		return nil, fmt.Errorf("Nothing in cache, unable to generate")
	}

	for {
		chosenListing = autoPlaylistItemCache[rand.Intn(len(autoPlaylistItemCache))]

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
