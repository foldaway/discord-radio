package commands

import (
	"fmt"

	"github.com/bottleneckco/discord-radio/models"
	"github.com/bwmarrin/discordgo"
)

func resume(s *discordgo.Session, m *discordgo.MessageCreate) {
	guildSession := safeGetGuildSession(s, m.Message.GuildID)
	if guildSession.MusicPlayer.PlaybackState == models.PlaybackStateStopped {
		s.ChannelMessageSend(m.Message.ChannelID, fmt.Sprintf("%s nothing to resume", m.Message.Author.Mention()))
		return
	}
	guildSession.MusicPlayer.Control <- models.MusicPlayerActionResume
	s.ChannelMessageSend(m.Message.ChannelID, fmt.Sprintf("%s resumed", m.Message.Author.Mention()))
}
