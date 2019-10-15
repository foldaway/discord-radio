package models

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
	youtube "google.golang.org/api/youtube/v3"
)

// MusicPlayer represents a music player
type MusicPlayer struct {
	StartTime time.Time
	IsPlaying bool
	Close     chan struct{}
	Control   chan MusicPlayerAction
}

// MusicPlayerAction an action to be issued to MusicPlayer
type MusicPlayerAction int

const (
	// MusicPlayerActionSkip skip this track
	MusicPlayerActionSkip MusicPlayerAction = iota

	// MusicPlayerActionPause pause this track
	MusicPlayerActionPause

	// MusicPlayerActionResume resume this track
	MusicPlayerActionResume
)

// GuildSession represents a guild voice session
type GuildSession struct {
	GuildID                     string
	Mutex                       sync.Mutex
	Queue                       []QueueItem // current item = index 0
	VoiceConnection             *discordgo.VoiceConnection
	PreviousAutoPlaylistListing *youtube.PlaylistItem
	MusicPlayer                 MusicPlayer
}

// Huge thanks to https://github.com/iopred/bruxism/blob/master/musicplugin/musicplugin.go

func createYTPipe(youtubeURL string) (*bufio.Reader, error) {
	args := []string{"-q", "-f", "bestaudio[abr>=130],best", "-o", "-", youtubeURL}
	ytdl := exec.Command("youtube-dl", args...)
	ytdl.Stderr = os.Stderr
	ytdlout, err := ytdl.StdoutPipe()
	if err != nil {
		log.Println("ytdl StdoutPipe err:", err)
		return nil, err
	}
	err = ytdl.Start()
	if err != nil {
		log.Println("ytdl Start err:", err)
		return nil, err
	}
	defer func() {
		go ytdl.Wait()
	}()
	ytdlbuf := bufio.NewReaderSize(ytdlout, 16384)
	return ytdlbuf, nil
}

// PlayYouTube play a YouTube video
func (guildSession *GuildSession) PlayYouTube(youtubeURL string, volume float64) error {
	log.Printf("[PLAYER] Playing YouTube '%s'\n", youtubeURL)

	ytPipe, err := createYTPipe(youtubeURL)
	if err != nil {
		return err
	}
	return guildSession.play(ytPipe, volume)
}

// PlayURL play a URL to an audio/video file
func (guildSession *GuildSession) PlayURL(url string, volume float64) error {
	log.Printf("[PLAYER] Playing URL '%s'\n", url)

	resp, err := http.Get(url)
	if err != nil {
		return err
	}

	return guildSession.play(bufio.NewReader(resp.Body), volume)
}
func (guildSession *GuildSession) play(pipe *bufio.Reader, volume float64) error {

	log.Println("[PLAYER] IsPlaying=true")
	guildSession.MusicPlayer.IsPlaying = true

	defer func() {
		log.Println("[PLAYER] IsPlaying=false")
		guildSession.MusicPlayer.IsPlaying = false
	}()

	ffmpeg := exec.Command("ffmpeg", "-hide_banner", "-nostats", "-loglevel", "error", "-i", "pipe:0", "-f", "s16le", "-ar", "48000", "-ac", "2", "-af", fmt.Sprintf("dynaudnorm=f=500:g=31:n=0:p=0.95,volume=%f", volume), "-b:a", "256k", "pipe:1")
	ffmpeg.Stdin = pipe
	ffmpeg.Stderr = os.Stderr
	ffmpegout, err := ffmpeg.StdoutPipe()
	if err != nil {
		log.Println("ffmpeg StdoutPipe err:", err)
		return err
	}
	ffmpegbuf := bufio.NewReaderSize(ffmpegout, 16384)

	dca := exec.Command("dca")
	dca.Stdin = ffmpegbuf
	dca.Stderr = os.Stderr
	dcaout, err := dca.StdoutPipe()
	if err != nil {
		log.Println("dca StdoutPipe err:", err)
		return err
	}
	dcabuf := bufio.NewReaderSize(dcaout, 16384)

	err = ffmpeg.Start()
	if err != nil {
		log.Println("ffmpeg Start err:", err)
		return err
	}
	defer func() {
		go ffmpeg.Wait()
	}()

	err = dca.Start()
	if err != nil {
		log.Println("dca Start err:", err)
		return err
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
			return nil
		default:
		}

		select {
		case ctl := <-guildSession.MusicPlayer.Control:
			switch ctl {
			case MusicPlayerActionSkip:
				log.Println("received skip")
				return nil
			case MusicPlayerActionPause:
				done := false
				for {

					ctl, ok := <-guildSession.MusicPlayer.Control
					if !ok {
						return nil
					}
					switch ctl {
					case MusicPlayerActionSkip:
						return nil
					case MusicPlayerActionResume:
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

		// Send received PCM to the sendPCM channel
		if guildSession.VoiceConnection != nil {
			guildSession.VoiceConnection.OpusSend <- opus
		} else {
			log.Println("[PLAYER] VoiceConnection nil, terminating OPUS transmission")
		}
	}
}
