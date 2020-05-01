package commands

import (
	"os"

	"github.com/bwmarrin/discordgo"
)

func suicide(s *discordgo.Session, m *discordgo.MessageCreate) {
	guildSession := safeGetGuildSession(s, m.Message.GuildID)
	s.ChannelMessageSend(m.Message.ChannelID, "Goodbye, cruel world!")
	if guildSession.VoiceConnection != nil {
		guildSession.VoiceConnection.Close()
	}
	os.Exit(1)
}
