package commands

import "github.com/bwmarrin/discordgo"

func ping(s *discordgo.Session, m *discordgo.MessageCreate) {
	s.ChannelMessageSend(m.Message.ChannelID, "pong")
}
