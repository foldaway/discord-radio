package commands

import (
	"os"

	"github.com/bwmarrin/discordgo"
)

func suicide(s *discordgo.Session, m *discordgo.MessageCreate) {
	guildSession := safeGetGuildSession(m.GuildID)
	s.ChannelMessageSend(m.ChannelID, "Goodbye, cruel world!")
	if guildSession.VoiceConnection != nil {
		guildSession.VoiceConnection.Disconnect()
	}
	os.Exit(1)
}
