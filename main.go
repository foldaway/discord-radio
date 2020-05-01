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
	"github.com/bwmarrin/discordgo"
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
	scheduler.StartImmediately()

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

	// client.On(disgord.EvtVoiceStateUpdate, func(s disgord.Session, vsu *disgord.VoiceStateUpdate) {
	// 	voiceStateCache, isCached := vscache.FindUserVoiceState(vsu.UserID)
	// 	hasPreviousVoiceState := isCached && voiceStateCache.Previous != nil
	// 	guildSession, ok := commands.GuildSessionMap[vsu.GuildID]

	// 	if !ok || guildSession.VoiceConnection == nil {
	// 		log.Println("[VSU] Not handling, guild has no voice connection")
	// 		return
	// 	}

	// 	botUser, err := client.GetCurrentUser(ctx.Ctx)
	// 	if err != nil {
	// 		log.Println(err)
	// 		return
	// 	}

	// 	if vsu.UserID == botUser.ID {
	// 		guildSession.VoiceChannelID = vsu.ChannelID
	// 		log.Println("[VSU] Updated internal cache of GuildSession.VoiceChannelID")
	// 	}

	// 	if hasPreviousVoiceState &&
	// 		(voiceStateCache.Previous.Deaf != vsu.Deaf ||
	// 			voiceStateCache.Previous.Mute != vsu.Mute ||
	// 			voiceStateCache.Previous.SelfDeaf != vsu.SelfDeaf ||
	// 			voiceStateCache.Previous.SelfMute != vsu.SelfMute) {
	// 		log.Println("[VSU] Not handling, it's only a deaf/mute state change")
	// 		return
	// 	}

	// 	var ttsMsg string
	// 	guildMember, err := client.GetMember(ctx.Ctx, vsu.GuildID, vsu.UserID)
	// 	if err != nil {
	// 		log.Println(err)
	// 		return
	// 	}
	// 	username := guildMember.Nick
	// 	if username == "" {
	// 		username = guildMember.User.Username
	// 	}
	// 	// userVoiceState, err := util.FindUserVoiceState(s, vsu.UserID)
	// 	if hasPreviousVoiceState && voiceStateCache.Previous.ChannelID != vsu.ChannelID && vsu.ChannelID != guildSession.VoiceChannelID {
	// 		// User left this bot's channel
	// 		ttsMsg = fmt.Sprintf("Goodbye, %s", username)
	// 		log.Printf("[VSU] User '%s' left channel '%s'\n", vsu.Member.Nick, vsu.ChannelID)
	// 	} else if guildSession.VoiceChannelID == vsu.ChannelID { // User joined this channel
	// 		ttsMsg = fmt.Sprintf("Welcome, %s", username)
	// 	} else if hasPreviousVoiceState && voiceStateCache.Previous.ChannelID != vsu.ChannelID {
	// 		log.Printf("[VSU] User '%s' joined channel '%s'\n", vsu.Member.Nick, vsu.ChannelID)
	// 	}
	// 	if len(ttsMsg) > 0 {
	// 		url, _ := googletts.GetTTSURL(ttsMsg, "en")
	// 		var isSomethingPlaying = guildSession.MusicPlayer.IsPlaying
	// 		if isSomethingPlaying {
	// 			guildSession.MusicPlayer.Control <- models.MusicPlayerActionPause
	// 		}
	// 		guildSession.PlayURL(url, 0.5)
	// 		if isSomethingPlaying {
	// 			guildSession.MusicPlayer.Control <- models.MusicPlayerActionResume
	// 			log.Println("[MAIN] Patching MusicPlayer IsPlaying=true")
	// 			guildSession.MusicPlayer.IsPlaying = true
	// 		}
	// 	}

	// 	guildSessionVoiceChannelUsers := vscache.GetUsersInVoiceChannel(vsu.GuildID, guildSession.VoiceChannelID)
	// 	log.Printf("[VSU] Currently left %d players in voice channel\n", len(guildSessionVoiceChannelUsers))

	// 	if len(guildSessionVoiceChannelUsers) == 1 {
	// 		// Only bot left
	// 		log.Println("Leaving, only me left in voice channel.")
	// 		s.UpdateStatus(&disgord.UpdateStatusPayload{AFK: true})
	// 		var tempVoiceConn = guildSession.VoiceConnection

	// 		guildSession.RWMutex.Lock()
	// 		guildSession.Queue = guildSession.Queue[0:0]
	// 		guildSession.RWMutex.Unlock()
	// 		guildSession.MusicPlayer.Close <- struct{}{}
	// 		tempVoiceConn.Close()
	// 		guildSession.VoiceConnection = nil
	// 	}
	// })

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
