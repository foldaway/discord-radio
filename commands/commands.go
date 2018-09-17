package commands

import (
	"log"
	"net/http"
	"os"

	"github.com/bottleneckco/radio-clerk/models"
	"github.com/bwmarrin/discordgo"
	"github.com/joho/godotenv"
	"google.golang.org/api/googleapi/transport"
	youtube "google.golang.org/api/youtube/v3"
)

var CommandsMap = make(map[string]func(*discordgo.Session, *discordgo.MessageCreate))
var Queue []models.QueueItem // current item = index 0
var VoiceConnection *discordgo.VoiceConnection
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
}
