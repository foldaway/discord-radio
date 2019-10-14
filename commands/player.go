package commands

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/url"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"github.com/evalphobia/google-tts-go/googletts"
	"github.com/masatana/go-textdistance"

	"google.golang.org/api/youtube/v3"

	"github.com/bottleneckco/discord-radio/models"
)

type ControlMessage int

const (
	Skip ControlMessage = iota
	Pause
	Resume
)

// Huge thanks to https://github.com/iopred/bruxism/blob/master/musicplugin/musicplugin.go

func (guildSession *GuildSession) Play(url, volume string) {
	log.Printf("[PLAYER] Playing URL '%s'\n", url)
	log.Println("[PLAYER] IsPlaying=true")
	guildSession.MusicPlayer.IsPlaying = true

	defer func() {
		log.Println("[PLAYER] IsPlaying=false")
		guildSession.MusicPlayer.IsPlaying = false
	}()

	args := []string{"-q"}
	if strings.Contains(url, "youtube.com") {
		args = append(args, "-f", "bestaudio[abr>=130],best")
	}
	args = append(args, "-o", "-", url)
	ytdl := exec.Command("youtube-dl", args...)
	ytdl.Stderr = os.Stderr
	ytdlout, err := ytdl.StdoutPipe()
	if err != nil {
		log.Println("ytdl StdoutPipe err:", err)
		return
	}
	ytdlbuf := bufio.NewReaderSize(ytdlout, 16384)
	ffmpeg := exec.Command("ffmpeg", "-hide_banner", "-nostats", "-loglevel", "error", "-i", "pipe:0", "-f", "s16le", "-ar", "48000", "-ac", "2", "-af", fmt.Sprintf("dynaudnorm=f=500:g=31:n=0:p=0.95,volume=%s", volume), "-b:a", "256k", "pipe:1")
	ffmpeg.Stdin = ytdlbuf
	ffmpeg.Stderr = os.Stderr
	ffmpegout, err := ffmpeg.StdoutPipe()
	if err != nil {
		log.Println("ffmpeg StdoutPipe err:", err)
		return
	}
	ffmpegbuf := bufio.NewReaderSize(ffmpegout, 16384)

	dca := exec.Command("dca")
	dca.Stdin = ffmpegbuf
	dca.Stderr = os.Stderr
	dcaout, err := dca.StdoutPipe()
	if err != nil {
		log.Println("dca StdoutPipe err:", err)
		return
	}
	dcabuf := bufio.NewReaderSize(dcaout, 16384)

	err = ytdl.Start()
	if err != nil {
		log.Println("ytdl Start err:", err)
		return
	}
	defer func() {
		go ytdl.Wait()
	}()

	err = ffmpeg.Start()
	if err != nil {
		log.Println("ffmpeg Start err:", err)
		return
	}
	defer func() {
		go ffmpeg.Wait()
	}()

	err = dca.Start()
	if err != nil {
		log.Println("dca Start err:", err)
		return
	}
	defer func() {
		go dca.Wait()
	}()

	// header "buffer"
	var opuslen int16

	// Send "speaking" packet over the voice websocket
	if guildSession.VoiceConnection != nil {
		guildSession.VoiceConnection.Speaking(true)
	}

	// Send not "speaking" packet over the websocket when we finish
	defer func() {
		if guildSession.VoiceConnection != nil {
			guildSession.VoiceConnection.Speaking(false)
		}
	}()

	guildSession.MusicPlayer.StartTime = time.Now()

	for {
		select {
		case <-guildSession.MusicPlayer.Close:
			log.Println("play() exited due to close channel.")
			return
		default:
		}

		select {
		case ctl := <-guildSession.MusicPlayer.Control:
			switch ctl {
			case Skip:
				log.Println("received skip")
				return
			case Pause:
				done := false
				for {

					ctl, ok := <-guildSession.MusicPlayer.Control
					if !ok {
						return
					}
					switch ctl {
					case Skip:
						return
					case Resume:
						done = true
						break
					}

					if done {
						break
					}

				}
			}
		default:
		}

		// read dca opus length header
		err = binary.Read(dcabuf, binary.LittleEndian, &opuslen)
		if err == io.EOF || err == io.ErrUnexpectedEOF {
			return
		}
		if err != nil {
			log.Println("read opus length from dca err:", err)
			return
		}

		// read opus data from dca
		opus := make([]byte, opuslen)
		err = binary.Read(dcabuf, binary.LittleEndian, &opus)
		if err == io.EOF || err == io.ErrUnexpectedEOF {
			return
		}
		if err != nil {
			log.Println("read opus from dca err:", err)
			return
		}

		// Send received PCM to the sendPCM channel
		if guildSession.VoiceConnection != nil {
			guildSession.VoiceConnection.OpusSend <- opus
		} else {
			log.Println("[PLAYER] VoiceConnection nil, terminating OPUS transmission")
			return
		}
	}
}

func GenerateAutoPlaylistQueueItem(guildSession *GuildSession) (models.QueueItem, error) {
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

		if guildSession.previousAutoPlaylistListing != nil && textdistance.LevenshteinDistance(guildSession.previousAutoPlaylistListing.Snippet.Title, chosenListingSnippet.Snippet.Title) > 20 {
			guildSession.previousAutoPlaylistListing = chosenListing
			guildSession.previousAutoPlaylistListing.Snippet = &youtube.PlaylistItemSnippet{Title: chosenListingSnippet.Snippet.Title}
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

func SafeCheckPlay(guildSession *GuildSession) {
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
		guildSession.Play(ttsMsgURL, "0.5")
	}
	log.Println("[PLAYER] Playing the actual song data")
	guildSession.Play(fmt.Sprintf("https://www.youtube.com/watch?v=%s", song.VideoID), os.Getenv("BOT_VOLUME"))
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
