package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/bottleneckco/discord-radio/commands"
	"github.com/bottleneckco/discord-radio/models"
	"github.com/bottleneckco/discord-radio/util"
	"github.com/bwmarrin/discordgo"
	"github.com/evalphobia/google-tts-go/googletts"
	"github.com/joho/godotenv"
)

var userVoiceStateCache = make(map[string]discordgo.VoiceState)

func main() {
	godotenv.Load()
	dg, err := discordgo.New(fmt.Sprintf("Bot %s", os.Getenv("DISCORD_TOKEN")))
	if err != nil {
		log.Panic(err)
	}

	gameStatusQuitChannel := make(chan bool)

	go func() {
		for {
			select {
			case <-gameStatusQuitChannel:
				return
			default:
				// Update music status
				if time.Now().Second()%5 == 0 {
					if len(commands.GuildSessionMap) == 0 {
						dg.UpdateStatus(1, "")
					} else {
						var sb strings.Builder
						for _, guildSession := range commands.GuildSessionMap {
							if len(guildSession.Queue) > 0 && guildSession.MusicPlayer.IsPlaying {
								guild, err := dg.Guild(guildSession.GuildID)
								if err != nil {
									log.Println(err)
									continue
								}
								song := guildSession.Queue[0]
								sb.WriteString(fmt.Sprintf("[%s] %s (%s) | ", guild.Name, song.Title, song.ChannelTitle))
							}
						}
						dg.UpdateStatus(0, sb.String())
					}
				}
			}
			time.Sleep(500 * time.Millisecond)
		}
	}()

	dg.AddHandler(func(s *discordgo.Session, m *discordgo.MessageCreate) {
		// Ignore all messages created by the bot itself
		if m.Author.ID == s.State.User.ID {
			return
		}

		log.Printf("[MESSAGE] '%s' - '%s'\n", m.Content, m.Author.Username)
		parts := strings.Split(m.Content, " ")

		var command string
		var args []string

		if len(m.Mentions) >= 1 && m.Mentions[0].ID == s.State.User.ID && len(parts) >= 2 {
			command = parts[1]
			args = parts[2:]
		} else if strings.HasPrefix(parts[0], os.Getenv("BOT_COMMAND_PREFIX")) {
			command = parts[0][1:]
			args = parts[1:]
		}

		if command != "" {
			if handler, ok := commands.CommandsMap[command]; ok {
				log.Printf("[COMMAND] Processing command '%s'\n", parts[0][1:])
				m.Content = strings.Join(args, " ")
				handler(s, m)
			}
		}
	})

	dg.AddHandler(func(s *discordgo.Session, vsu *discordgo.VoiceStateUpdate) {
		// Populate all voice states
		voiceChannelUsers, err := util.GetUsersInVoiceChannel(s, vsu.GuildID, vsu.ChannelID)
		if err != nil {
			log.Println(err)
			return
		}

		for _, user := range voiceChannelUsers {
			if _, isExist := userVoiceStateCache[user.UserID]; !isExist {
				userVoiceStateCache[user.UserID] = *user
			}
		}

		previousVoiceState, hasPreviousVoiceState := userVoiceStateCache[vsu.UserID]
		userVoiceStateCache[vsu.UserID] = *vsu.VoiceState
		guildSession, ok := commands.GuildSessionMap[vsu.GuildID]

		if !ok || guildSession.VoiceConnection == nil {
			log.Println("[VSU] Not handling, guild has no voice connection")
			return
		}

		if hasPreviousVoiceState &&
			(previousVoiceState.Deaf != vsu.Deaf ||
				previousVoiceState.Mute != vsu.Mute ||
				previousVoiceState.SelfDeaf != vsu.SelfDeaf ||
				previousVoiceState.SelfMute != vsu.SelfMute) {
			log.Println("[VSU] Not handling, it's only a deaf/mute state change")
			return
		}

		var ttsMsg string
		guildMember, err := s.GuildMember(vsu.GuildID, vsu.UserID)
		if err != nil {
			log.Println(err)
			return
		}
		username := guildMember.Nick
		if username == "" {
			username = guildMember.User.Username
		}
		// userVoiceState, err := util.FindUserVoiceState(s, vsu.UserID)
		if hasPreviousVoiceState && previousVoiceState.ChannelID != vsu.ChannelID && vsu.ChannelID != guildSession.VoiceConnection.ChannelID {
			// User left this bot's channel
			ttsMsg = fmt.Sprintf("Goodbye, %s", username)
		} else if guildSession.VoiceConnection.ChannelID == vsu.ChannelID { // User joined this channel
			ttsMsg = fmt.Sprintf("Welcome, %s", username)
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

		guildSessionVoiceChannelUsers, err := util.GetUsersInVoiceChannel(s, vsu.GuildID, guildSession.VoiceConnection.ChannelID)
		if err != nil {
			log.Println(err)
			return
		}

		if len(guildSessionVoiceChannelUsers) == 1 {
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

	gameStatusQuitChannel <- true

	// Cleanly close down the Discord session.
	dg.Close()
}
