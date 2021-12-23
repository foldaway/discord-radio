package commands

import (
	"context"
	"fmt"
	"github.com/andersfylling/disgord"
	"github.com/bottleneckco/discord-radio/session"
)

func resume(s disgord.Session, m *disgord.MessageCreate) {
	guildSession := findOrCreateGuildSession(s, m.Message.GuildID)
	if guildSession.MusicPlayer.PlaybackState == session.PlaybackStateStopped {
		m.Message.Reply(
			context.Background(),
			s,
			fmt.Sprintf("%s nothing to resume", m.Message.Author.Mention()),
		)
		return
	}
	guildSession.MusicPlayer.Control <- session.MusicPlayerActionResume
	m.Message.Reply(
		context.Background(),
		s,
		fmt.Sprintf("%s resumed", m.Message.Author.Mention()),
	)
}
