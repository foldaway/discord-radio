package commands

import (
	"fmt"

	"github.com/andersfylling/disgord"
	"github.com/bottleneckco/discord-radio/models"
)

func resume(s disgord.Session, m *disgord.MessageCreate) {
	guildSession := safeGetGuildSession(m.Message.GuildID)
	if !guildSession.MusicPlayer.IsPlaying {
		s.SendMsg(m.Message.ChannelID, fmt.Sprintf("%s nothing to resume", m.Message.Author.Mention()))
		return
	}
	guildSession.MusicPlayer.Control <- models.MusicPlayerActionResume
	s.SendMsg(m.Message.ChannelID, fmt.Sprintf("%s resumed", m.Message.Author.Mention()))
}
