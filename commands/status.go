package commands

import (
	"fmt"
	"strings"

	"github.com/andersfylling/disgord"
	"github.com/bottleneckco/discord-radio/vscache"
)

func status(s disgord.Session, m *disgord.MessageCreate) {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Guild Count: %d\n", len(GuildSessionMap)))
	sb.WriteString(fmt.Sprintf("Active Voice Sessions: %d\n", vscache.ActiveVSCount()))
	s.SendMsg(m.Message.ChannelID, fmt.Sprintf("%s current status:\n%s", m.Message.Author.Mention(), sb.String()))
}
