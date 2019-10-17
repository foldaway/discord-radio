package commands

import (
	"fmt"
	"os"
	"strings"

	"github.com/andersfylling/disgord"
)

func help(s disgord.Session, m *disgord.MessageCreate) {
	var b strings.Builder
	for k := range CommandsMap {
		b.WriteString(fmt.Sprintf("%s%s\n", os.Getenv("BOT_COMMAND_PREFIX"), k))
	}
	s.SendMsg(m.Message.ChannelID, fmt.Sprintf("%s commands list:\n%s", m.Message.Author.Mention(), b.String()))
}
