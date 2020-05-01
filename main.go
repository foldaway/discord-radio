package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"time"

	"github.com/bottleneckco/discord-radio/commands"
	"github.com/bottleneckco/discord-radio/models"
	"github.com/bottleneckco/discord-radio/util"
	"github.com/bwmarrin/discordgo"
	"github.com/evalphobia/google-tts-go/googletts"
	"github.com/go-co-op/gocron"
	"github.com/joho/godotenv"
)

func main() {
	godotenv.Load()
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	var session *discordgo.Session
	var err error

	session, err = discordgo.New(fmt.Sprintf("Bot %s", os.Getenv("DISCORD_TOKEN")))
	if err != nil {
		log.Panic(err)
	}

	var scheduler = gocron.NewScheduler(time.Local)
	scheduler.Every(1).Day().Do(func() {
		log.Println("Clearing youtube-dl cache dir")
		var ytdl = exec.Command("youtube-dl", "--rm-cache-dir")
		ytdl.Stdout = os.Stdout
		ytdl.Stderr = os.Stderr
		ytdl.Run()
	})
	scheduler.Every(5).Seconds().Do(func() {
		var sb strings.Builder
		for _, guildSession := range commands.GuildSessionMap {
			if len(guildSession.Queue) > 0 && guildSession.MusicPlayer.IsPlaying {
				song := guildSession.Queue[0]
				sb.WriteString(fmt.Sprintf("[%s] (1 of %d) %s | ", util.GenerateAcronym(guildSession.GuildName), len(guildSession.Queue), song.Title))
			}
		}
		if sb.Len() > 0 {
			session.UpdateStatus(0, sb.String())
		} else {
			session.UpdateStatus(1, "")
		}

	})
	scheduler.Start()

	//gameStatusQuitChannel := make(chan bool)

	session.AddHandler(func(s *discordgo.Session, event *discordgo.MessageCreate) {
		log.Printf("[MESSAGE] %s: '%s'\n", event.Author.Username, event.Message.Content)

		var parts = strings.Split(event.Message.Content, " ")

		var command string
		var args []string

		var isMessageHasMentions = len(event.Message.Mentions) >= 1
		var isBotMentioned = isMessageHasMentions && event.Message.Mentions[0].ID == s.State.User.ID
		var isCommandPrefixed = strings.HasPrefix(parts[0], os.Getenv("BOT_COMMAND_PREFIX"))

		if isBotMentioned && len(parts) >= 2 {
			command = parts[1]
			args = parts[2:]
		} else if isCommandPrefixed {
			command = parts[0][1:]
			args = parts[1:]
		}

		if len(command) != 0 {
			if commandHandler, ok := commands.CommandsMap[command]; ok {
				log.Printf("[COMMAND] Processing command '%s %+v'\n", command, args)
				event.Message.Content = strings.Join(args, " ")
				commandHandler(s, event)
			} else {
				log.Printf("[COMMAND] Unknown command '%s %+v'\n", command, args)
			}
		}
	})

	session.AddHandler(func(s *discordgo.Session, event *discordgo.Ready) {

	})

	var voiceStateCache = make(map[string]discordgo.VoiceState)

	session.AddHandler(func(s *discordgo.Session, event *discordgo.VoiceStateUpdate) {
		previousVoiceState, hasPreviousVoiceState := voiceStateCache[event.UserID]
		guildSession, ok := commands.GuildSessionMap[event.GuildID]

		if event.VoiceState != nil {
			voiceStateCache[event.UserID] = *event.VoiceState
		} else {
			delete(voiceStateCache, event.UserID)
		}

		if !ok || guildSession.VoiceConnection == nil {
			log.Println("[VSU] Not handling, guild has no voice connection")
			return
		}

		if event.UserID == s.State.User.ID {
			guildSession.VoiceChannelID = event.ChannelID
			log.Println("[VSU] Updated internal cache of GuildSession.VoiceChannelID")
		}

		if hasPreviousVoiceState &&
			(previousVoiceState.Deaf != event.Deaf ||
				previousVoiceState.Mute != event.Mute ||
				previousVoiceState.SelfDeaf != event.SelfDeaf ||
				previousVoiceState.SelfMute != event.SelfMute) {
			log.Println("[VSU] Not handling, it's only a deaf/mute state change")
			return
		}

		var ttsMsg string
		guildMember, err := s.GuildMember(event.GuildID, event.UserID)
		if err != nil {
			log.Println(err)
			return
		}
		username := guildMember.Nick
		if username == "" {
			username = guildMember.User.Username
		}
		// userVoiceState, err := util.FindUserVoiceState(s, vsu.UserID)
		if hasPreviousVoiceState && previousVoiceState.ChannelID != event.ChannelID && event.ChannelID != guildSession.VoiceChannelID {
			// User left this bot's channel
			ttsMsg = fmt.Sprintf("Goodbye, %s", username)
			log.Printf("[VSU] User '%s' left channel '%s'\n", guildMember.User.Username, event.ChannelID)
		} else if guildSession.VoiceChannelID == event.ChannelID { // User joined this channel
			ttsMsg = fmt.Sprintf("Welcome, %s", username)
		} else if hasPreviousVoiceState && previousVoiceState.ChannelID != event.ChannelID {
			log.Printf("[VSU] User '%s' joined channel '%s'\n", guildMember.User.Username, event.ChannelID)
		}
		if len(ttsMsg) > 0 {
			url, _ := googletts.GetTTSURL(ttsMsg, "en")
			var isSomethingPlaying = guildSession.MusicPlayer.IsPlaying
			if isSomethingPlaying {
				guildSession.MusicPlayer.Control <- models.MusicPlayerActionPause
			}
			guildSession.PlayURL(url, 0.5)
			if isSomethingPlaying {
				guildSession.MusicPlayer.Control <- models.MusicPlayerActionResume
				log.Println("[MAIN] Patching MusicPlayer IsPlaying=true")
				guildSession.MusicPlayer.IsPlaying = true
			}
		}

		guildSessionVoiceChannelUsers := util.GetChannelVoiceStates(s, event.GuildID, guildSession.VoiceChannelID)
		log.Printf("[VSU] Currently left %d players in voice channel\n", len(guildSessionVoiceChannelUsers))

		if len(guildSessionVoiceChannelUsers) == 1 {
			// Only bot left
			log.Println("Leaving, only me left in voice channel.")
			s.UpdateStatus(1, "")
			var tempVoiceConn = guildSession.VoiceConnection

			guildSession.RWMutex.Lock()
			guildSession.Queue = guildSession.Queue[0:0]
			guildSession.RWMutex.Unlock()
			guildSession.MusicPlayer.Close <- struct{}{}
			tempVoiceConn.Close()
			guildSession.VoiceConnection = nil
		}
	})

	err = session.Open()
	if err != nil {
		log.Panic(err)
	}

	// Wait here until CTRL-C or other term signal is received.
	log.Println("Bot is now running.  Press CTRL-C to exit.")

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)

	select {
	case <-stop:
		log.Println("Closing connections")

		for _, guildSession := range commands.GuildSessionMap {
			if guildSession.VoiceConnection != nil {
				guildSession.VoiceConnection.Close()
			}
		}

		err = session.Close()
		if err != nil {
			log.Panic(err)
		}
	}

}
