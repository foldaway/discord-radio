package commands

import (
	"context"
	"fmt"
	"github.com/andersfylling/disgord"
	"os"
	"strings"
)

func help(s disgord.Session, m *disgord.MessageCreate) {
	var b strings.Builder
	for k := range PrimaryCommandMap {
		b.WriteString(fmt.Sprintf("%s%s\n", os.Getenv("BOT_COMMAND_PREFIX"), k))
	}
	m.Message.Reply(
		context.Background(),
		s,
		fmt.Sprintf("%s commands list:\n%s", m.Message.Author.Mention(), b.String()),
	)
}
