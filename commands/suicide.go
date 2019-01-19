package commands

import (
	"os"

	"github.com/bwmarrin/discordgo"
)

func suicide(s *discordgo.Session, m *discordgo.MessageCreate) {
	s.ChannelMessageSend(m.ChannelID, "Goodbye, cruel world!")
	if VoiceConnection != nil {
		VoiceConnection.Disconnect()
	}
	os.Exit(1)
}
