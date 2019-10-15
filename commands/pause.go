package commands

import (
	"fmt"

	"github.com/bottleneckco/discord-radio/models"
	"github.com/bwmarrin/discordgo"
)

func pause(s *discordgo.Session, m *discordgo.MessageCreate) {
	guildSession := safeGetGuildSession(m.GuildID)
	if !guildSession.MusicPlayer.IsPlaying {
		s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("%s nothing to pause", m.Author.Mention()))
		return
	}
	guildSession.MusicPlayer.Control <- models.MusicPlayerActionPause
	s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("%s paused", m.Author.Mention()))
}
