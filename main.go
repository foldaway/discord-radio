package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/bottleneckco/discord-radio/commands"
	"github.com/bottleneckco/discord-radio/util"
	"github.com/bwmarrin/discordgo"
	"github.com/evalphobia/google-tts-go/googletts"
	"github.com/joho/godotenv"
)

func main() {
	godotenv.Load()
	dg, err := discordgo.New(fmt.Sprintf("Bot %s", os.Getenv("DISCORD_TOKEN")))
	if err != nil {
		log.Panic(err)
	}

	dg.AddHandler(func(s *discordgo.Session, m *discordgo.MessageCreate) {
		// Ignore all messages created by the bot itself
		if m.Author.ID == s.State.User.ID {
			return
		}

		log.Printf("[MESSAGE] '%s' - '%s'\n", m.Content, m.Author.Username)
		parts := strings.Split(m.Content, " ")

		if strings.HasPrefix(parts[0], os.Getenv("BOT_COMMAND_PREFIX")) {
			if handler, ok := commands.CommandsMap[parts[0][1:]]; ok {
				log.Printf("[COMMAND] Processing command '%s'\n", parts[0][1:])
				m.Content = strings.Join(parts[1:], " ")
				handler(s, m)
			}
		}
	})

	dg.AddHandler(func(s *discordgo.Session, vsu *discordgo.VoiceStateUpdate) {
		guildSession, ok := commands.GuildSessionMap[vsu.GuildID]
		if !ok || guildSession.VoiceConnection == nil {
			return
		}
		channel, err := s.Channel(guildSession.VoiceConnection.ChannelID)
		if err != nil {
			log.Println(err)
			return
		}
		if channel.ID != guildSession.VoiceConnection.ChannelID {
			// Not my voice channel
			return
		}

		var ttsMsg string
		guildMember, err := s.GuildMember(vsu.GuildID, vsu.UserID)
		if err != nil {
			log.Println(err)
			return
		}
		userVoiceState, err := util.FindUserVoiceState(s, vsu.UserID)
		if userVoiceState == nil || err != nil || vsu.ChannelID == "" {
			// User disconnected from all voice channels
			ttsMsg = fmt.Sprintf("Goodbye, %s", guildMember.Nick)
		} else if channel.ID == userVoiceState.ChannelID && !userVoiceState.SelfMute && !userVoiceState.SelfDeaf { // User joined this channel
			ttsMsg = fmt.Sprintf("Welcome, %s", guildMember.Nick)
		}
		if len(ttsMsg) > 0 {
			url, _ := googletts.GetTTSURL(ttsMsg, "en")
			var isSomethingPlaying = guildSession.MusicPlayer.IsPlaying
			if isSomethingPlaying {
				guildSession.MusicPlayer.Control <- commands.Pause
			}
			guildSession.Play(url, "0.5")
			if isSomethingPlaying {
				guildSession.MusicPlayer.Control <- commands.Resume
				log.Println("[MAIN] Patching MusicPlayer IsPlaying=true")
				guildSession.MusicPlayer.IsPlaying = true
			}
		}

		if len(util.GetUsersInVoiceChannel(s, guildSession.VoiceConnection.ChannelID)) == 1 {
			// Only bot left
			log.Println("Leaving, only me left in voice channel.")
			s.UpdateStatus(1, "")
			var tempVoiceConn = guildSession.VoiceConnection
			guildSession.VoiceConnection = nil

			guildSession.Mutex.Lock()
			guildSession.Queue = guildSession.Queue[0:0]
			guildSession.Mutex.Unlock()
			guildSession.MusicPlayer.Close <- struct{}{}
			tempVoiceConn.Disconnect()
		}
	})

	commands.GameUpdateFunc = func(game string) {
		dg.UpdateStatus(0, game)
	}

	err = dg.Open()
	if err != nil {
		log.Panic(err)
	}

	// Wait here until CTRL-C or other term signal is received.
	fmt.Println("Bot is now running.  Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc

	for _, guildSession := range commands.GuildSessionMap {
		if guildSession.VoiceConnection != nil {
			guildSession.VoiceConnection.Disconnect()
		}
	}

	// Cleanly close down the Discord session.
	dg.Close()
}
