package youtube

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"github.com/andersfylling/disgord/json"
	"log"
	"math/rand"
	"net/url"
	"os"
	"os/exec"
	"sort"
	"strings"
	"time"

	"github.com/go-co-op/gocron"
	"github.com/joho/godotenv"

	cryptoRand "crypto/rand"
)

var (
	autoPlaylistItemCache = make([]PlaylistItem, 0)
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

	var scheduler = gocron.NewScheduler(time.Local)
	_, err = scheduler.Every(6).Hours().StartImmediately().StartAt(time.Now().Add(time.Second * 2)).Do(cacheAutoPlaylistItems)
	if err != nil {
		log.Println(err)
	}

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
		var titlePartsI = strings.Split(autoPlaylistItemCache[i].Title, " - ")
		var titlePartsJ = strings.Split(autoPlaylistItemCache[j].Title, " - ")

		return titlePartsI[0] < titlePartsJ[0]
	})

	log.Printf("[AP] Cached %d items\n", len(autoPlaylistItemCache))
}

// FetchAllPlaylistItems get all the items of a playlist
func FetchAllPlaylistItems(playlistURL *url.URL) ([]PlaylistItem, error) {
	var playlist Playlist

	var cmdOutput bytes.Buffer

	var youtubeDL = exec.Command(
		"youtube-dl",
		playlistURL.String(),
		"--dump-single-json",
		"--skip-download",
		"--flat-playlist",
	)

	youtubeDL.Stdout = &cmdOutput

	var err = youtubeDL.Run()
	if err != nil {
		return playlist.Entries, err
	}

	err = json.Unmarshal(cmdOutput.Bytes(), &playlist)
	if err != nil {
		return playlist.Entries, err
	}

	log.Printf("Fetching from YOUTUBE. We now have %d listings\n", len(playlist.Entries))

	return playlist.Entries, nil
}

func Search(term string, maxResults int64) ([]PlaylistItem, error) {
	var playlist Playlist
	var cmdOutput bytes.Buffer

	var youtubeDL = exec.Command(
		"youtube-dl",
		fmt.Sprintf("ytsearch%d:%s", maxResults, term),
		"--dump-single-json",
		"--skip-download",
		"--flat-playlist",
	)

	youtubeDL.Stdout = &cmdOutput

	var err = youtubeDL.Run()
	if err != nil {
		return playlist.Entries, err
	}

	err = json.Unmarshal(cmdOutput.Bytes(), &playlist)
	if err != nil {
		return playlist.Entries, err
	}

	log.Printf("Searching YOUTUBE. We now have %d results\n", len(playlist.Entries))

	return playlist.Entries, nil
}

// GenerateAutoPlaylistQueueItem get a new item from the auto playlist
func GenerateAutoPlaylistQueueItem(videoIdsToAvoid []string) (PlaylistItem, error) {
	var chosenListing PlaylistItem

	if len(autoPlaylistItemCache) == 0 {
		return chosenListing, fmt.Errorf("Nothing in cache, unable to generate")
	}

	for {
		chosenListing = autoPlaylistItemCache[rand.Intn(len(autoPlaylistItemCache))]

		if !stringInSlice(chosenListing.Id, videoIdsToAvoid) {
			break
		}

		// Played before
		log.Println("Reshuffling, played before")
	}

	log.Printf("[AP] Chosen video '%s'\n", chosenListing.Title)

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
