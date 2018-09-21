package commands

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"time"
)

type controlMessage int

const (
	Skip controlMessage = iota
	Pause
	Resume
)

type Player struct {
	StartTime time.Time
	IsPlaying bool
	Close     chan struct{}
	Control   chan controlMessage
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
	ffmpeg := exec.Command("ffmpeg", "-i", "pipe:0", "-f", "s16le", "-ar", "48000", "-ac", "2", "pipe:1")
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
	VoiceConnection.Speaking(true)

	// Send not "speaking" packet over the websocket when we finish
	defer VoiceConnection.Speaking(false)

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
		VoiceConnection.OpusSend <- opus
	}
}

func SafeCheckPlay() {
	if VoiceConnection == nil {
		log.Println("[SCP] no voice connection")
		return
	}
	if len(Queue) == 0 {
		log.Println("[SCP] no items in queue")
		return
	}
	if player.IsPlaying {
		log.Println("[SCP] currently playing something!")
		return
	}
	var song = Queue[0]
	player.Play(fmt.Sprintf("https://www.youtube.com/watch?v=%s", song.VideoID))
	Queue = Queue[1:]
	go SafeCheckPlay()
}
