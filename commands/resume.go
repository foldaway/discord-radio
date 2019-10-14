package commands

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
)

func resume(s *discordgo.Session, m *discordgo.MessageCreate) {
	guildSession := safeGetGuildSession(m.GuildID)
	if !guildSession.MusicPlayer.IsPlaying {
		s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("%s nothing to resume", m.Author.Mention()))
		return
	}
	guildSession.MusicPlayer.Control <- Resume
	s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("%s resumed", m.Author.Mention()))
}
