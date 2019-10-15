package commands

import (
	"fmt"
	"log"
	"strings"

	"github.com/bwmarrin/discordgo"
)

func queue(s *discordgo.Session, m *discordgo.MessageCreate) {
	guildSession := safeGetGuildSession(m.GuildID)
	guildSession.Mutex.Lock()
	if len(guildSession.Queue) == 0 {
		s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("%s nothing in the queue.", m.Author.Mention()))
		guildSession.Mutex.Unlock()
		return
	}
	var b strings.Builder
	b.WriteString(fmt.Sprintf("%s here is the queue:\n", m.Author.Mention()))

	b.WriteString(fmt.Sprintf("⏯ **%s**    ▶️️%s   ⏫%s\n", guildSession.Queue[0].Title, guildSession.Queue[0].ChannelTitle, guildSession.Queue[0].Author))
	for index, queueItem := range guildSession.Queue[1:] {
		b.WriteString(fmt.Sprintf("`️%d.` **%s**   ⬆️%s   ⏫%s\n", index+2, queueItem.Title, queueItem.ChannelTitle, queueItem.Author))
	}
	guildSession.Mutex.Unlock()
	log.Println("Off")
	s.ChannelMessageSend(m.ChannelID, b.String())
}
