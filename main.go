package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/andersfylling/disgord"
	"github.com/bottleneckco/discord-radio/commands"
	"github.com/bottleneckco/discord-radio/ctx"
	"github.com/bottleneckco/discord-radio/models"
	"github.com/bottleneckco/discord-radio/util"
	"github.com/bottleneckco/discord-radio/vscache"
	"github.com/evalphobia/google-tts-go/googletts"
	"github.com/joho/godotenv"
)

func main() {
	godotenv.Load()
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	client := disgord.New(disgord.Config{
		BotToken: os.Getenv("DISCORD_TOKEN"),
		Logger:   disgord.DefaultLogger(false),
		CacheConfig: &disgord.CacheConfig{
			DisableVoiceStateCaching: true,
		},
	})

	gameStatusQuitChannel := make(chan bool)

	go func() {
		for {
			select {
			case <-gameStatusQuitChannel:
				return
			default:
				// Update music status
				if time.Now().Second()%20 == 0 {
					if len(commands.GuildSessionMap) == 0 {
						client.UpdateStatus(&disgord.UpdateStatusPayload{
							AFK: true,
						})
					} else {
						var sb strings.Builder
						for _, guildSession := range commands.GuildSessionMap {
							if len(guildSession.Queue) > 0 && guildSession.MusicPlayer.IsPlaying {
								guild, err := client.GetGuild(ctx.Ctx, guildSession.GuildID)
								if err != nil {
									log.Println(err)
									continue
								}
								song := guildSession.Queue[0]
								sb.WriteString(fmt.Sprintf("[%s] (1 of %d) %s | ", util.GenerateAcronym(guild.Name), len(guildSession.Queue), song.Title))
							}
						}
						client.UpdateStatusString(sb.String())
					}
				}
			}
			time.Sleep(500 * time.Millisecond)
		}
	}()

	client.On(disgord.EvtMessageCreate, func(s disgord.Session, m *disgord.MessageCreate) {
		// Ignore all messages created by the bot itself
		botUser, err := client.GetCurrentUser(ctx.Ctx)
		if err != nil {
			log.Println(err)
			return
		}
		if m.Message.Author.ID == botUser.ID {
			return
		}

		log.Printf("[MESSAGE] '%s' - '%s'\n", m.Message.Content, m.Message.Author.Username)
		parts := strings.Split(m.Message.Content, " ")

		var command string
		var args []string

		if len(m.Message.Mentions) >= 1 && m.Message.Mentions[0].ID == botUser.ID && len(parts) >= 2 {
			command = parts[1]
			args = parts[2:]
		} else if strings.HasPrefix(parts[0], os.Getenv("BOT_COMMAND_PREFIX")) {
			command = parts[0][1:]
			args = parts[1:]
		}

		if command != "" {
			if handler, ok := commands.CommandsMap[command]; ok {
				log.Printf("[COMMAND] Processing command '%s'\n", parts[0][1:])
				m.Message.Content = strings.Join(args, " ")
				handler(s, m)
			}
		}
	})

	client.Ready(func() {
		vscache.PreloadGuilds(client)
	})

	client.On(disgord.EvtVoiceStateUpdate, vscache.HandleVSU)

	client.On(disgord.EvtVoiceStateUpdate, func(s disgord.Session, vsu *disgord.VoiceStateUpdate) {
		voiceStateCache, isCached := vscache.FindUserVoiceState(vsu.UserID)
		hasPreviousVoiceState := isCached && voiceStateCache.Previous != nil
		guildSession, ok := commands.GuildSessionMap[vsu.GuildID]

		if !ok || guildSession.VoiceConnection == nil {
			log.Println("[VSU] Not handling, guild has no voice connection")
			return
		}

		botUser, err := client.GetCurrentUser(ctx.Ctx)
		if err != nil {
			log.Println(err)
			return
		}

		if vsu.UserID == botUser.ID {
			guildSession.VoiceChannelID = vsu.ChannelID
			log.Println("[VSU] Updated internal cache of GuildSession.VoiceChannelID")
		}

		if hasPreviousVoiceState &&
			(voiceStateCache.Previous.Deaf != vsu.Deaf ||
				voiceStateCache.Previous.Mute != vsu.Mute ||
				voiceStateCache.Previous.SelfDeaf != vsu.SelfDeaf ||
				voiceStateCache.Previous.SelfMute != vsu.SelfMute) {
			log.Println("[VSU] Not handling, it's only a deaf/mute state change")
			return
		}

		var ttsMsg string
		guildMember, err := client.GetMember(ctx.Ctx, vsu.GuildID, vsu.UserID)
		if err != nil {
			log.Println(err)
			return
		}
		username := guildMember.Nick
		if username == "" {
			username = guildMember.User.Username
		}
		// userVoiceState, err := util.FindUserVoiceState(s, vsu.UserID)
		if hasPreviousVoiceState && voiceStateCache.Previous.ChannelID != vsu.ChannelID && vsu.ChannelID != guildSession.VoiceChannelID {
			// User left this bot's channel
			ttsMsg = fmt.Sprintf("Goodbye, %s", username)
			log.Printf("[VSU] User '%s' left channel '%s'\n", vsu.Member.Nick, vsu.ChannelID)
		} else if guildSession.VoiceChannelID == vsu.ChannelID { // User joined this channel
			ttsMsg = fmt.Sprintf("Welcome, %s", username)
		} else if hasPreviousVoiceState && voiceStateCache.Previous.ChannelID != vsu.ChannelID {
			log.Printf("[VSU] User '%s' joined channel '%s'\n", vsu.Member.Nick, vsu.ChannelID)
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

		guildSessionVoiceChannelUsers := vscache.GetUsersInVoiceChannel(vsu.GuildID, guildSession.VoiceChannelID)
		log.Printf("[VSU] Currently left %d players in voice channel\n", len(guildSessionVoiceChannelUsers))

		if len(guildSessionVoiceChannelUsers) == 1 {
			// Only bot left
			log.Println("Leaving, only me left in voice channel.")
			s.UpdateStatus(&disgord.UpdateStatusPayload{AFK: true})
			var tempVoiceConn = guildSession.VoiceConnection

			guildSession.RWMutex.Lock()
			guildSession.Queue = guildSession.Queue[0:0]
			guildSession.RWMutex.Unlock()
			guildSession.MusicPlayer.Close <- struct{}{}
			tempVoiceConn.Close()
			guildSession.VoiceConnection = nil
		}
	})

	err := client.Connect(ctx.Ctx)
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
			guildSession.VoiceConnection.Close()
		}
	}

	err = client.Disconnect()
	if err != nil {
		log.Panic(err)
	}

	gameStatusQuitChannel <- true
}
