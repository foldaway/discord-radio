package models

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
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

	ytdlbuf := bufio.NewReaderSize(ytdlout, 16384)
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
func (mp *MusicPlayer) PlayStream(stream *bufio.Reader) error {
	var ffmpeg = exec.Command(
		"ffmpeg",
		"-hide_banner",
		"-nostats",
		"-loglevel",
		"error",
		"-i",
		"pipe:0",
		"-f",
		"s16le",
		"-ar",
		"48000",
		"-ac",
		"2",
		"-af",
		fmt.Sprintf("dynaudnorm=f=500:g=31:n=0:p=%f", volume),
		"-b:a",
		"256k",
		"pipe:1",
	)
	ffmpeg.Stdin = stream
	ffmpeg.Stderr = os.Stderr
	ffmpegout, err := ffmpeg.StdoutPipe()
	if err != nil {
		log.Println("ffmpeg StdoutPipe err:", err)
		return err
	}
	ffmpegbuf := bufio.NewReaderSize(ffmpegout, 1000000)

	dca := exec.Command("dca")
	dca.Stdin = ffmpegbuf
	dca.Stderr = os.Stderr
	dcaout, err := dca.StdoutPipe()
	if err != nil {
		log.Println("dca StdoutPipe err:", err)
		return err
	}

	var dcabuf = bufio.NewReaderSize(dcaout, 1000000)

	mp.PlaybackState = PlaybackStatePlaying
	defer func() {
		mp.PlaybackState = PlaybackStateStopped
	}()

	err = ffmpeg.Start()
	if err != nil {
		log.Println("ffmpeg Start error", err)
		return err
	}

	err = dca.Start()
	if err != nil {
		log.Println("dca Start error", err)
		return err
	}

	defer func() {
		dca.Process.Kill()
		ffmpeg.Process.Kill()
	}()

	// header "buffer"
	var opuslen int16

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

		if ffmpeg.ProcessState != nil && ffmpeg.ProcessState.Exited() {
			return nil
		} else if dca.ProcessState != nil && dca.ProcessState.Exited() {
			log.Println("DCA exited early. Something wrong during playback")
			return nil
		}

		// read dca opus length header
		err = binary.Read(dcabuf, binary.LittleEndian, &opuslen)
		if err == io.EOF || err == io.ErrUnexpectedEOF {
			return err
		}
		if err != nil {
			log.Println("read opus length from dca err:", err)
			return err
		}

		// read opus data from dca
		opus := make([]byte, opuslen)
		err = binary.Read(dcabuf, binary.LittleEndian, &opus)
		if err == io.EOF || err == io.ErrUnexpectedEOF {
			return err
		}
		if err != nil {
			log.Println("read opus from dca err:", err)
			return err
		}

		mp.PlaybackChannel <- opus
	}
}
