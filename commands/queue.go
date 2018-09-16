package commands

import (
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
)

func queue(s *discordgo.Session, m *discordgo.MessageCreate) {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("%s here is the queue:\n", m.Author.Mention()))

	if len(Queue) > 0 {
		b.WriteString(fmt.Sprintf("`️▶.` **%s** `%s` [%s]\n", Queue[0].Title, Queue[0].ChannelTitle, Queue[0].Author))
		for index, queueItem := range Queue[1:] {
			b.WriteString(fmt.Sprintf("`️%d` **%s** `%s` [%s]\n", index+1, queueItem.Title, queueItem.ChannelTitle, queueItem.Author))
		}
	} else {
		b.WriteString("Nothing in queue")
	}
	s.ChannelMessageSend(m.ChannelID, b.String())
}
