package commands

import (
	"context"
	"fmt"
	"github.com/andersfylling/disgord"
	"github.com/bottleneckco/discord-radio/util"
	"strings"
)

func status(s disgord.Session, m *disgord.MessageCreate) {
	var sb strings.Builder

	var connectedGuilds = s.GetConnectedGuilds()
	sb.WriteString(fmt.Sprintf("Guild Count: %d\n", len(connectedGuilds)))

	var vsCount = 0

	for range util.GlobalVoiceStateCache.VoiceStates {
		vsCount++
	}

	sb.WriteString(fmt.Sprintf("Total Voice Sessions: %d\n", vsCount))
	m.Message.Reply(
		context.Background(),
		s,
		fmt.Sprintf("%s current status:\n%s", m.Message.Author.Mention(), sb.String()),
	)
}
