package commands

import (
	"fmt"

	"github.com/bottleneckco/discord-radio/models"
	"github.com/bwmarrin/discordgo"
)

func pause(s *discordgo.Session, m *discordgo.MessageCreate) {
	guildSession := safeGetGuildSession(m.Message.GuildID)
	if !guildSession.MusicPlayer.IsPlaying {
		s.ChannelMessageSend(m.Message.ChannelID, fmt.Sprintf("%s nothing to pause", m.Message.Author.Mention()))
		return
	}
	guildSession.MusicPlayer.Control <- models.MusicPlayerActionPause
	s.ChannelMessageSend(m.Message.ChannelID, fmt.Sprintf("%s paused", m.Message.Author.Mention()))
}
