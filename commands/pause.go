package commands

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
)

func pause(s *discordgo.Session, m *discordgo.MessageCreate) {
	if !player.IsPlaying {
		s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("%s nothing to pause", m.Author.Mention()))
		return
	}
	player.Control <- Pause
	s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("%s paused", m.Author.Mention()))
}
