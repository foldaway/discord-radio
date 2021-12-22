package commands

import (
	"context"
	"fmt"
	"github.com/andersfylling/disgord"
	"github.com/bottleneckco/discord-radio/session"
)

func pause(s disgord.Session, m *disgord.MessageCreate) {
	guildSession := findOrCreateGuildSession(s, m.Message.GuildID)
	switch guildSession.MusicPlayer.PlaybackState {
	case session.PlaybackStatePaused:
		m.Message.Reply(
			context.Background(),
			s,
			fmt.Sprintf("%s already paused", m.Message.Author.Mention()),
		)
		return
	case session.PlaybackStateStopped:
		m.Message.Reply(
			context.Background(),
			s,
			fmt.Sprintf("%s nothing to pause", m.Message.Author.Mention()),
		)
		return
	}
	guildSession.MusicPlayer.Control <- session.MusicPlayerActionPause
	m.Message.Reply(
		context.Background(),
		s,
		fmt.Sprintf("%s paused", m.Message.Author.Mention()),
	)
}
