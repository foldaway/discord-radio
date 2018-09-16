package commands

import "github.com/bwmarrin/discordgo"

var CommandsMap = make(map[string]func(*discordgo.Session, *discordgo.MessageCreate))

func init() {
	CommandsMap["ping"] = ping
}
