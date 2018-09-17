package commands

import (
	"os"

	"github.com/bwmarrin/discordgo"
)

func suicide(s *discordgo.Session, m *discordgo.MessageCreate) {
	s.ChannelMessageSend(m.ChannelID, "Goodbye, cruel world!")
	os.Exit(1)
}
