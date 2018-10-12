package commands

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
)

func resume(s *discordgo.Session, m *discordgo.MessageCreate) {
	if !MusicPlayer.IsPlaying {
		s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("%s nothing to resume", m.Author.Mention()))
		return
	}
	MusicPlayer.Control <- Resume
	s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("%s resumed", m.Author.Mention()))
}
