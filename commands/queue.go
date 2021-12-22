package commands

import (
	"context"
	"fmt"
	"github.com/andersfylling/disgord"
	"strings"
)

func queue(s disgord.Session, m *disgord.MessageCreate) {
	guildSession := safeGetGuildSession(s, m.Message.GuildID)
	guildSession.RWMutex.RLock()
	if len(guildSession.Queue) == 0 {
		m.Message.Reply(
			context.Background(),
			s,
			fmt.Sprintf("%s nothing in the queue.", m.Message.Author.Mention()),
		)
		guildSession.RWMutex.RUnlock()
		return
	}
	var b strings.Builder
	b.WriteString(fmt.Sprintf("%s here is the queue:\n", m.Message.Author.Mention()))

	b.WriteString(fmt.Sprintf("⏯ **%s <%s>**   ⏫%s\n", guildSession.Queue[0].Title, fmt.Sprintf("https://youtube.com/watch?v=%s", guildSession.Queue[0].VideoID), guildSession.Queue[0].Author))
	for index, queueItem := range guildSession.Queue[1:] {
		b.WriteString(fmt.Sprintf("`️%d.` **%s <%s>**   ⏫%s\n", index+2, queueItem.Title, fmt.Sprintf("https://youtube.com/watch?v=%s", queueItem.VideoID), queueItem.Author))
	}
	guildSession.RWMutex.RUnlock()
	m.Message.Reply(
		context.Background(),
		s,
		b.String(),
	)
}
