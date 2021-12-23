package main

import (
	"fmt"
	"github.com/andersfylling/disgord"
	"github.com/andersfylling/disgord/std"
	"github.com/bottleneckco/discord-radio/commands"
	"github.com/bottleneckco/discord-radio/session"
	"github.com/bottleneckco/discord-radio/util"
	"github.com/go-co-op/gocron"
	"github.com/joho/godotenv"
	"log"
	"os"
	"strings"
	"time"
)

var (
	commandPrefix = os.Getenv("BOT_COMMAND_PREFIX")

	previousCommandMap = make(map[disgord.Snowflake]string)

	client *disgord.Client
)

func handleMsg(s disgord.Session, data *disgord.MessageCreate) {
	var message = data.Message
	var authorID = message.Author.ID
	log.Printf("[MSG] '%s'\n", message.Content)

	var messageParts = strings.Split(message.Content, " ")

	var isCommand = strings.HasPrefix(messageParts[0], commandPrefix)

	if isCommand {
		var command = messageParts[0][len(commandPrefix):]

		if commandHandler, isCommandHandlerExists := commands.PrimaryCommandMap[command]; isCommandHandlerExists {
			commandHandler(s, data)

			if _, isSecondaryCommandHandlerExists := commands.SecondaryCommandMap[command]; isSecondaryCommandHandlerExists {
				previousCommandMap[authorID] = command
			}
		} else {
			log.Println("Unsupported command", command)
		}
	} else {
		if previousCommand, ok := previousCommandMap[authorID]; ok {
			var messageHandler = commands.SecondaryCommandMap[previousCommand]
			messageHandler(s, data)
			delete(previousCommandMap, authorID)
		}
	}
}

func handleGuildCreate(s disgord.Session, data *disgord.GuildCreate) {
	for _, vs := range data.Guild.VoiceStates {
		util.GlobalVoiceStateCache.Handle(s, vs)
	}
}

func handleVoiceStateUpdate(s disgord.Session, data *disgord.VoiceStateUpdate) {
	util.GlobalVoiceStateCache.Handle(s, data.VoiceState)
}

func updateBotStatus() {
	var guildStatuses []string

	for _, guildSession := range commands.GuildSessionMap {
		if len(guildSession.Queue) > 0 && guildSession.MusicPlayer.PlaybackState == session.PlaybackStatePlaying {
			song := guildSession.Queue[0]
			guildStatuses = append(
				guildStatuses,
				fmt.Sprintf("[%s] (1 of %d) %s", util.GenerateAcronym(guildSession.GuildName), len(guildSession.Queue), song.Title),
			)
		}
	}

	var statusString = strings.Join(guildStatuses, " | ")

	var err = client.UpdateStatusString(statusString)

	if err != nil {
		log.Println("Error updating bot status", err)
	}
}

func main() {
	godotenv.Load()
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	client = disgord.New(disgord.Config{
		ProjectName: os.Getenv("BOT_NICKNAME"),
		BotToken:    os.Getenv("DISCORD_TOKEN"),
		RejectEvents: []string{
			disgord.EvtTypingStart,
			disgord.EvtPresenceUpdate,
			disgord.EvtGuildMemberAdd,
			disgord.EvtGuildMemberUpdate,
			disgord.EvtGuildMemberRemove,
		},
	})

	defer client.Gateway().StayConnectedUntilInterrupted()

	logFilter, _ := std.NewLogFilter(client)

	var handlerRegistrar = client.Gateway().WithMiddleware(
		logFilter.LogMsg,
	)

	handlerRegistrar.MessageCreate(handleMsg)
	handlerRegistrar.GuildCreate(handleGuildCreate)
	handlerRegistrar.VoiceStateUpdate(handleVoiceStateUpdate)

	client.Gateway().BotReady(func() {
		log.Println("Bot is ready")
	})

	var scheduler = gocron.NewScheduler(time.Local)

	scheduler.Every(15).Seconds().Do(updateBotStatus)
	scheduler.Start()

	// Wait here until CTRL-C or other term signal is received.
	log.Println("Bot is now running.  Press CTRL-C to exit.")

}
