package commands

import (
	"fmt"

	"github.com/bottleneckco/discord-radio/models"
	"github.com/bwmarrin/discordgo"
)

func pause(s *discordgo.Session, m *discordgo.MessageCreate) {
	guildSession := safeGetGuildSession(s, m.Message.GuildID)
	switch guildSession.MusicPlayer.PlaybackState {
	case models.PlaybackStatePaused:
		s.ChannelMessageSend(m.Message.ChannelID, fmt.Sprintf("%s already paused", m.Message.Author.Mention()))
		return
	case models.PlaybackStateStopped:
		s.ChannelMessageSend(m.Message.ChannelID, fmt.Sprintf("%s nothing to pause", m.Message.Author.Mention()))
		return
	}
	guildSession.MusicPlayer.Control <- models.MusicPlayerActionPause
	s.ChannelMessageSend(m.Message.ChannelID, fmt.Sprintf("%s paused", m.Message.Author.Mention()))
}
