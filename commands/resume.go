package commands

import (
	"fmt"

	"github.com/bottleneckco/discord-radio/models"
	"github.com/bwmarrin/discordgo"
)

func resume(s *discordgo.Session, m *discordgo.MessageCreate) {
	guildSession := safeGetGuildSession(s, m.GuildID)
	if !guildSession.MusicPlayer.IsPlaying {
		s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("%s nothing to resume", m.Author.Mention()))
		return
	}
	guildSession.MusicPlayer.Control <- models.MusicPlayerActionResume
	s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("%s resumed", m.Author.Mention()))
}
