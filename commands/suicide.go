package commands

import (
	"context"
	"github.com/andersfylling/disgord"
	"os"
)

func suicide(s disgord.Session, m *disgord.MessageCreate) {
	guildSession := safeGetGuildSession(s, m.Message.GuildID)
	m.Message.Reply(
		context.Background(),
		s,
		"Goodbye, cruel world!",
	)
	if guildSession.VoiceConnection != nil {
		guildSession.VoiceConnection.Close()
	}
	os.Exit(1)
}
