package session

import (
	"bufio"
	"bytes"
	"fmt"
	"github.com/fourthclasshonours/dca"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strconv"
)

type PlaybackState int

const (
	PlaybackStatePlaying = iota
	PlaybackStatePaused
	PlaybackStateStopped
)

var (
	volume = 0.5
)

func init() {
	var volumeStr = os.Getenv("BOT_VOLUME")
	volumeConv, err := strconv.ParseFloat(volumeStr, 64)
	if err != nil {
		log.Println("Error parsing invalid volume:", volumeStr, ", using default 0.5")
		return
	}
	volume = volumeConv
}

// MusicPlayer represents a music player
type MusicPlayer struct {
	PlaybackState   PlaybackState
	Control         chan MusicPlayerAction
	PlaybackChannel chan []byte
	playStream      *bytes.Buffer
	Volume          float64
}

// MusicPlayerAction an action to be issued to MusicPlayer
type MusicPlayerAction int

const (
	// MusicPlayerActionStop stop this track
	MusicPlayerActionStop MusicPlayerAction = iota

	// MusicPlayerActionPause pause this track
	MusicPlayerActionPause

	// MusicPlayerActionResume resume this track
	MusicPlayerActionResume
)

func (mp *MusicPlayer) PlayYouTubeVideo(youtubeURL string) error {
	args := []string{"-q", "-f", "bestaudio[abr>=130],best", "-o", "-", youtubeURL}
	ytdl := exec.Command("youtube-dl", args...)
	ytdl.Stderr = os.Stderr
	ytdlout, err := ytdl.StdoutPipe()
	if err != nil {
		log.Println("ytdl StdoutPipe err:", err)
		return err
	}

	ytdlbuf := bufio.NewReaderSize(ytdlout, 5*1000*1000)
	err = ytdl.Start()
	if err != nil {
		return err
	}
	return mp.PlayStream(ytdlbuf)
}

// PlayURL play a URL to an audio/video file
func (mp *MusicPlayer) PlayURL(url string) error {
	log.Printf("[PLAYER] Playing URL '%s'\n", url)

	resp, err := http.Get(url)
	if err != nil {
		return err
	}

	return mp.PlayStream(bufio.NewReader(resp.Body))
}

// Huge thanks to https://github.com/iopred/bruxism/blob/master/musicplugin/musicplugin.go
func (mp *MusicPlayer) PlayStream(stream io.Reader) error {
	encoder, err := dca.EncodeMem(
		stream,
		&dca.EncodeOptions{
			Volume:        256,
			Channels:      2,
			FrameRate:     48000,
			FrameDuration: 20,
			Bitrate:       256,
			Threads:       0,
			VBR:           true,
			Application:   dca.AudioApplicationAudio,
			CoverFormat:   "jpeg",
			AudioFilter:   fmt.Sprintf("dynaudnorm=f=500:g=31:n=0:p=%f", volume),
		},
	)

	if err != nil {
		return err
	}

	defer encoder.Cleanup()

	mp.PlaybackState = PlaybackStatePlaying
	defer func() {
		mp.PlaybackState = PlaybackStateStopped
	}()

	for {
		select {
		case ctl := <-mp.Control:
			switch ctl {
			case MusicPlayerActionStop:
				return nil
			case MusicPlayerActionPause:
				var done = false
				mp.PlaybackState = PlaybackStatePaused
				for {
					ctl, ok := <-mp.Control
					if !ok {
						return nil
					}
					switch ctl {
					case MusicPlayerActionResume:
						mp.PlaybackState = PlaybackStatePlaying
						done = true
						break
					case MusicPlayerActionStop:
						return nil
					}

					if done {
						break
					}
				}
			}
		default:
		}

		var opusFrame []byte
		var err error

		opusFrame, err = encoder.OpusFrame()
		if err != nil {
			return err
		}

		mp.PlaybackChannel <- opusFrame
	}
}
