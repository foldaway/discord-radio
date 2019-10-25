package commands

import (
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/andersfylling/disgord"
	"github.com/bottleneckco/discord-radio/models"
	"github.com/joho/godotenv"
	"google.golang.org/api/googleapi/transport"
	youtube "google.golang.org/api/youtube/v3"
)

// CommandsMap a map of all the command handlers
var CommandsMap = make(map[string]func(disgord.Session, *disgord.MessageCreate))

func newGuildSession(guildID disgord.Snowflake) models.GuildSession {
	return models.GuildSession{
		GuildID: guildID,
		RWMutex: sync.RWMutex{},
		MusicPlayer: models.MusicPlayer{
			Close:   make(chan struct{}),
			Control: make(chan models.MusicPlayerAction),
		},
	}
}

// GuildSessionMap a map of all the guild sessions
var GuildSessionMap = make(map[disgord.Snowflake]*models.GuildSession)

func safeGetGuildSession(guildID disgord.Snowflake) *models.GuildSession {
	if session, ok := GuildSessionMap[guildID]; ok {
		return session
	}
	session := newGuildSession(guildID)
	GuildSessionMap[guildID] = &session
	return &session
}

var youtubeService *youtube.Service

func init() {
	godotenv.Load()
	var err error
	client := &http.Client{
		Transport: &transport.APIKey{Key: os.Getenv("GOOGLE_API_KEY")},
	}

	youtubeService, err = youtube.New(client)
	if err != nil {
		log.Println(err)
	}

	CommandsMap["ping"] = ping
	CommandsMap["q"] = queue
	CommandsMap["queue"] = queue
	CommandsMap["play"] = play
	CommandsMap["suicide"] = suicide
	CommandsMap["skip"] = skip
	CommandsMap["join"] = join
	CommandsMap["pause"] = pause
	CommandsMap["resume"] = resume
	CommandsMap["help"] = help
	CommandsMap["leave"] = leave
}

func deleteMessageDelayed(s disgord.Session, msg *disgord.Message) {
	time.Sleep(20 * time.Second)
	s.DeleteFromDiscord(msg)
}
