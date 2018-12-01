package commands

import (
	"fmt"
	"os"
	"strings"

	"github.com/bwmarrin/discordgo"
)

func help(s *discordgo.Session, m *discordgo.MessageCreate) {
	var b strings.Builder
	for k := range CommandsMap {
		b.WriteString(fmt.Sprintf("%s%s\n", os.Getenv("BOT_COMMAND_PREFIX"), k))
	}
	s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("%s commands list:\n%s", m.Author.Mention(), b.String()))
	SafeCheckPlay()
}
