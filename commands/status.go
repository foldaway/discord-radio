package commands

import (
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
)

func status(s *discordgo.Session, m *discordgo.MessageCreate) {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Guild Count: %d\n", len(s.State.Guilds)))

	var vsCount = 0
	var botVsCount = 0
	for _, guild := range s.State.Guilds {
		vsCount += len(guild.VoiceStates)
		for _, vs := range guild.VoiceStates {
			if vs.UserID == s.State.User.ID {
				botVsCount += 1
			}
		}
	}
	sb.WriteString(fmt.Sprintf("Total Voice Sessions: %d\n", vsCount))
	sb.WriteString(fmt.Sprintf("Active Bot Voice Sessions: %d\n", botVsCount))
	s.ChannelMessageSend(m.Message.ChannelID, fmt.Sprintf("%s current status:\n%s", m.Message.Author.Mention(), sb.String()))
}
