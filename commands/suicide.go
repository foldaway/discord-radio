package commands

import (
	"os"

	"github.com/andersfylling/disgord"
)

func suicide(s disgord.Session, m *disgord.MessageCreate) {
	guildSession := safeGetGuildSession(m.Message.GuildID)
	s.SendMsg(m.Message.ChannelID, "Goodbye, cruel world!")
	if guildSession.VoiceConnection != nil {
		guildSession.VoiceConnection.Close()
	}
	os.Exit(1)
}
