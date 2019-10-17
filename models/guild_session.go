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
	"strconv"
	"sync"
	"time"
	"unicode"

	"github.com/andersfylling/disgord"
	"github.com/bottleneckco/discord-radio/util"
	"github.com/chrisport/go-lang-detector/langdet"
	"github.com/evalphobia/google-tts-go/googletts"
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
	GuildID                     disgord.Snowflake
	Mutex                       sync.Mutex
	Queue                       []QueueItem // current item = index 0
	VoiceConnection             disgord.VoiceConnection
	VoiceChannelID              disgord.Snowflake
	PreviousAutoPlaylistListing *youtube.PlaylistItem
	MusicPlayer                 MusicPlayer
}

// Loop session management loop
func (guildSession *GuildSession) Loop() {
	for {
		log.Println("LOOP")
		if guildSession.VoiceConnection == nil {
			time.Sleep(1 * time.Second)
			continue
		}
		if guildSession.MusicPlayer.IsPlaying {
			log.Println("[SCP] currently playing something!")
			time.Sleep(1 * time.Second)
			continue
		}
		if len(guildSession.Queue) == 0 && len(os.Getenv("BOT_AUTO_PLAYLIST")) == 0 {
			log.Println("[SCP] no items in queue")
			time.Sleep(1 * time.Second)
			continue
		} else if len(guildSession.Queue) == 0 {
			log.Println("[SCP] Getting from auto playlist")
			playlistItem, err := util.GenerateAutoPlaylistQueueItem(guildSession.PreviousAutoPlaylistListing)
			if err != nil {
				log.Printf("[SCP] Error generating auto playlist item: %s\n", err)
				time.Sleep(1 * time.Second)
				continue
			}
			queueItem := ConvertYouTubePlaylistItem(playlistItem)
			guildSession.Mutex.Lock()
			guildSession.Queue = append(guildSession.Queue, queueItem)
			guildSession.Mutex.Unlock()
		}
		guildSession.Mutex.Lock()
		var song = guildSession.Queue[0]
		guildSession.Mutex.Unlock()

		// Announce music title
		songTitle := util.SanitiseSongTitleTTS(song.Title)

		detector := langdet.NewDetector()
		clc := langdet.UnicodeRangeLanguageComparator{"zh-TW", unicode.Han}
		jlc := langdet.UnicodeRangeLanguageComparator{"ja", unicode.Katakana}
		klc := langdet.UnicodeRangeLanguageComparator{"ko", unicode.Hangul}
		eng := langdet.UnicodeRangeLanguageComparator{"en", unicode.ASCII_Hex_Digit}
		detector.AddLanguageComparators(&clc, &jlc, &klc, &eng)

		if ttsMsgURL, err := googletts.GetTTSURL(fmt.Sprintf("Music: %s", songTitle), detector.GetLanguages(songTitle)[0].Name); err == nil {
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
	}
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
		guildSession.VoiceConnection.StartSpeaking()
	}

	// Send not "speaking" packet over the websocket when we finish
	defer func() {
		if guildSession.VoiceConnection != nil {
			guildSession.VoiceConnection.StopSpeaking()
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
			guildSession.VoiceConnection.SendOpusFrame(opus)
		} else {
			log.Println("[PLAYER] VoiceConnection nil, terminating OPUS transmission")
		}
	}
}
