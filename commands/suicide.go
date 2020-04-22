package commands

import (
	"os"

	"github.com/andersfylling/disgord"
	"github.com/bottleneckco/discord-radio/ctx"
)

func suicide(s disgord.Session, m *disgord.MessageCreate) {
	guildSession := safeGetGuildSession(m.Message.GuildID)
	s.SendMsg(ctx.Ctx, m.Message.ChannelID, "Goodbye, cruel world!")
	if guildSession.VoiceConnection != nil {
		guildSession.VoiceConnection.Close()
	}
	os.Exit(1)
}
