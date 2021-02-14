package util

import (
	"encoding/binary"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/go-co-op/gocron"
	"google.golang.org/api/googleapi/transport"

	"google.golang.org/api/youtube/v3"

	"github.com/joho/godotenv"

	cryptoRand "crypto/rand"
)

var (
	youtubeService        *youtube.Service
	autoPlaylistItemCache = make([]*youtube.PlaylistItem, 0)
)

func init() {
	godotenv.Load()

  // https://stackoverflow.com/a/54491783
  var b [8]byte
  var err error
  _, err = cryptoRand.Read(b[:])
  if err != nil {
    log.Println("Could not seed with crypto/rand")
    rand.Seed(time.Now().UnixNano())
  } else {
    rand.Seed(int64(binary.LittleEndian.Uint64(b[:])))
  }

	client := &http.Client{
		Transport: &transport.APIKey{Key: os.Getenv("GOOGLE_API_KEY")},
	}

	youtubeService, err = youtube.New(client)
	if err != nil {
		log.Println(err)
	}

	var scheduler = gocron.NewScheduler(time.Local)
	scheduler.Every(6).Hours().StartImmediately().StartAt(time.Now().Add(time.Second * 2)).Do(cacheAutoPlaylistItems)

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
	sort.Slice(autoPlaylistItemCache, func(i, j int) bool {
		var titlePartsI = strings.Split(autoPlaylistItemCache[i].Snippet.Title, " - ")
		var titlePartsJ = strings.Split(autoPlaylistItemCache[j].Snippet.Title, " - ")

		return titlePartsI[0] < titlePartsJ[0]
	})

	log.Printf("[AP] Cached %d items\n", len(autoPlaylistItemCache))
}

// FetchAllPlaylistItems get all the items of a playlist
func FetchAllPlaylistItems(playlistURL *url.URL) ([]*youtube.PlaylistItem, error) {
	var listings []*youtube.PlaylistItem
	var pageToken string
	for {
		log.Printf("Fetching from YOUTUBE. We now have %d listings\n", len(listings))
		youtubeListings, err := youtubeService.
			PlaylistItems.
			List("contentDetails,snippet").
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

// GenerateAutoPlaylistQueueItem get a new item from the auto playlist
func GenerateAutoPlaylistQueueItem(videoIdsToAvoid []string) (*youtube.PlaylistItem, error) {
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

		if stringInSlice(snippetsResp.Items[0].Id, videoIdsToAvoid) {
			// Played before
			log.Println("Reshuffling, played before")
			continue
		}

		chosenListing.Snippet = &youtube.PlaylistItemSnippet{
			Title:        snippetsResp.Items[0].Snippet.Title,
			ChannelTitle: snippetsResp.Items[0].Snippet.ChannelTitle,
			Thumbnails:   snippetsResp.Items[0].Snippet.Thumbnails,
		}
		break

	}

	log.Printf("[AP] Chosen video '%s' by '%s'\n", chosenListing.Snippet.Title, chosenListing.Snippet.ChannelTitle)

	return chosenListing, nil
}

func GetAutoPlaylistCacheLength() int {
	return len(autoPlaylistItemCache)
}

// https://stackoverflow.com/questions/15323767/does-go-have-if-x-in-construct-similar-to-python
func stringInSlice(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}
