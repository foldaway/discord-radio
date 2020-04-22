package commands

import (
	"fmt"

	"github.com/andersfylling/disgord"
	"github.com/bottleneckco/discord-radio/ctx"
	"github.com/bottleneckco/discord-radio/models"
)

func pause(s disgord.Session, m *disgord.MessageCreate) {
	guildSession := safeGetGuildSession(m.Message.GuildID)
	if !guildSession.MusicPlayer.IsPlaying {
		s.SendMsg(ctx.Ctx, m.Message.ChannelID, fmt.Sprintf("%s nothing to pause", m.Message.Author.Mention()))
		return
	}
	guildSession.MusicPlayer.Control <- models.MusicPlayerActionPause
	s.SendMsg(ctx.Ctx, m.Message.ChannelID, fmt.Sprintf("%s paused", m.Message.Author.Mention()))
}
