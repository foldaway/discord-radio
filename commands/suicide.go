package commands

import (
	"context"
	"github.com/andersfylling/disgord"
	"os"
)

func suicide(s disgord.Session, m *disgord.MessageCreate) {
	guildSession := findOrCreateGuildSession(s, m.Message.GuildID)
	m.Message.Reply(
		context.Background(),
		s,
		"Goodbye, cruel world!",
	)
	if guildSession.VoiceConnection != nil {
		var voiceConnection = *guildSession.VoiceConnection

		voiceConnection.Close()
	}
	os.Exit(1)
}
