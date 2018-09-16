package commands

import (
	"github.com/bottleneckco/radio-clerk/models"
	"github.com/bwmarrin/discordgo"
)

var CommandsMap = make(map[string]func(*discordgo.Session, *discordgo.MessageCreate))
var Queue []models.QueueItem // current item = index 0
var VoiceConnection *discordgo.VoiceConnection

func init() {
	CommandsMap["ping"] = ping
	CommandsMap["q"] = queue
	CommandsMap["queue"] = queue
}
