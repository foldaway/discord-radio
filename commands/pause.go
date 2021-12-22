package commands

import (
	"context"
	"fmt"
	"github.com/andersfylling/disgord"

	"github.com/bottleneckco/discord-radio/models"
)

func pause(s disgord.Session, m *disgord.MessageCreate) {
	guildSession := safeGetGuildSession(s, m.Message.GuildID)
	switch guildSession.MusicPlayer.PlaybackState {
	case models.PlaybackStatePaused:
		m.Message.Reply(
			context.Background(),
			s,
			fmt.Sprintf("%s already paused", m.Message.Author.Mention()),
		)
		return
	case models.PlaybackStateStopped:
		m.Message.Reply(
			context.Background(),
			s,
			fmt.Sprintf("%s nothing to pause", m.Message.Author.Mention()),
		)
		return
	}
	guildSession.MusicPlayer.Control <- models.MusicPlayerActionPause
	m.Message.Reply(
		context.Background(),
		s,
		fmt.Sprintf("%s paused", m.Message.Author.Mention()),
	)
}
