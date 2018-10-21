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
	"time"

	"google.golang.org/api/youtube/v3"

	"github.com/bottleneckco/radio-clerk/models"
)

type ControlMessage int

const (
	Skip ControlMessage = iota
	Pause
	Resume
)

type Player struct {
	StartTime time.Time
	IsPlaying bool
	Close     chan struct{}
	Control   chan ControlMessage
}

var debug = false

// Huge thanks to https://github.com/iopred/bruxism/blob/master/musicplugin/musicplugin.go

func (p *Player) Play(url string) {
	p.IsPlaying = true
	ytdl := exec.Command("youtube-dl", "-v", "-f", "bestaudio[abr>=130]", "-o", "-", url)
	if debug {
		ytdl.Stderr = os.Stderr
	}
	ytdlout, err := ytdl.StdoutPipe()
	if err != nil {
		log.Println("ytdl StdoutPipe err:", err)
		return
	}
	ytdlbuf := bufio.NewReaderSize(ytdlout, 16384)
	ffmpeg := exec.Command("ffmpeg", "-i", "pipe:0", "-f", "s16le", "-ar", "48000", "-ac", "2", "-af", fmt.Sprintf("dynaudnorm=f=500:g=31:n=0:p=0.95,volume=%s", os.Getenv("BOT_VOLUME")), "-b:a", "256k", "pipe:1")
	ffmpeg.Stdin = ytdlbuf
	if debug {
		ffmpeg.Stderr = os.Stderr
	}
	ffmpegout, err := ffmpeg.StdoutPipe()
	if err != nil {
		log.Println("ffmpeg StdoutPipe err:", err)
		return
	}
	ffmpegbuf := bufio.NewReaderSize(ffmpegout, 16384)

	dca := exec.Command("dca")
	dca.Stdin = ffmpegbuf
	if debug {
		dca.Stderr = os.Stderr
	}
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

	defer func() {
		p.IsPlaying = false
	}()

	// header "buffer"
	var opuslen int16

	// Send "speaking" packet over the voice websocket
	if VoiceConnection != nil {
		VoiceConnection.Speaking(true)
	}

	// Send not "speaking" packet over the websocket when we finish
	defer func() {
		if VoiceConnection != nil {
			VoiceConnection.Speaking(false)
		}
	}()

	p.StartTime = time.Now()

	for {
		select {
		case <-p.Close:
			log.Println("play() exited due to close channel.")
			return
		default:
		}

		select {
		case ctl := <-p.Control:
			switch ctl {
			case Skip:
				log.Println("received skip")
				return
			case Pause:
				done := false
				for {

					ctl, ok := <-p.Control
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
		if VoiceConnection != nil {
			VoiceConnection.OpusSend <- opus
		} else {
			log.Println("[PLAYER] VoiceConnection nil, terminating OPUS transmission")
			return
		}
	}
}

func GenerateAutoPlaylistQueueItem() (models.QueueItem, error) {
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
		for {
			chosenListing = listings[rand.Intn(len(listings))]
			if previousAutoPlaylistListing != nil && previousAutoPlaylistListing.ContentDetails.VideoId != chosenListing.ContentDetails.VideoId {
				previousAutoPlaylistListing = chosenListing
				break
			} else {
				break
			}
		}

		log.Printf("[AP] Chosen v='%s'\n", chosenListing.ContentDetails.VideoId)

		chosenListingSnippets, err := youtubeService.Videos.List("snippet").Id(chosenListing.ContentDetails.VideoId).Do()
		if err != nil {
			return data, err
		}
		if len(chosenListingSnippets.Items) == 0 {
			continue
		}
		chosenListingSnippet = chosenListingSnippets.Items[0]
		break
	}

	log.Printf("[AP] Chosen video '%s' by '%s'\n", chosenListingSnippet.Snippet.Title, chosenListingSnippet.Snippet.ChannelTitle)

	data = models.QueueItem{
		Title:        chosenListingSnippet.Snippet.Title,
		ChannelTitle: chosenListingSnippet.Snippet.ChannelTitle,
		Author:       "AutoPlaylist",
		VideoID:      chosenListing.ContentDetails.VideoId,
	}
	return data, nil
}

func SafeCheckPlay() {
	if VoiceConnection == nil {
		log.Println("[SCP] no voice connection")
		return
	}
	if MusicPlayer.IsPlaying {
		log.Println("[SCP] currently playing something!")
		return
	}
	if len(Queue) == 0 && len(os.Getenv("BOT_AUTO_PLAYLIST")) == 0 {
		log.Println("[SCP] no items in queue")
		return
	} else if len(Queue) == 0 {
		log.Println("[SCP] Getting from auto playlist")
		queueItem, err := GenerateAutoPlaylistQueueItem()
		if err != nil {
			log.Printf("[SCP] Error generating auto playlist item: %s\n", err)
			return
		}
		Queue = append(Queue, queueItem)
	}
	var song = Queue[0]
	GameUpdateFunc(fmt.Sprintf("%s (%s)", song.Title, song.ChannelTitle))
	MusicPlayer.Play(fmt.Sprintf("https://www.youtube.com/watch?v=%s", song.VideoID))
	if len(Queue) > 0 {
		Queue = Queue[1:]
	}
	if VoiceConnection != nil {
		go SafeCheckPlay()
	}
}
