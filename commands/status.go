package commands

import (
	"context"
	"fmt"
	"github.com/andersfylling/disgord"
	"strings"
)

func status(s disgord.Session, m *disgord.MessageCreate) {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Guild Count: %d\n", len(s.GetConnectedGuilds())))

	//var vsCount = 0
	//var botVsCount = 0
	//for _, guild := range s.GetConnectedGuilds() {
	//	vsCount += len(guild.VoiceStates)
	//	for _, vs := range guild.VoiceStates {
	//		if vs.UserID == s.State.User.ID {
	//			botVsCount += 1
	//		}
	//	}
	//}
	//sb.WriteString(fmt.Sprintf("Total Voice Sessions: %d\n", vsCount))
	//sb.WriteString(fmt.Sprintf("Active Bot Voice Sessions: %d\n", botVsCount))
	m.Message.Reply(
		context.Background(),
		s,
		fmt.Sprintf("%s current status:\n%s", m.Message.Author.Mention(), sb.String()),
	)
}
