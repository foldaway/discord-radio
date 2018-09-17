package commands

import (
	"fmt"
	"strconv"

	"github.com/bottleneckco/radio-clerk/models"

	"github.com/bwmarrin/discordgo"
)

func skip(s *discordgo.Session, m *discordgo.MessageCreate) {
	if len(Queue) == 0 {
		s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("%s nothing to skip", m.Author.Mention()))
		return
	}
	var skippedItem models.QueueItem
	if len(m.Content) == 0 {
		// No args, skip current
		skippedItem = Queue[0]
		Queue = append(Queue[:0], Queue[1:]...)
	} else {
		choice, err := strconv.ParseInt(m.Content, 10, 64)
		if err == nil && (choice-1 >= 0 && choice-1 < int64(len(Queue))) {
			skippedItem = Queue[choice-1]
			Queue = append(Queue[:choice-1], Queue[choice:]...)
		} else {
			s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("%s invalid choice", m.Author.Mention()))
			return
		}
	}

	s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("%s skipped **%s** `%s` [%s]\n", m.Author.Mention(), skippedItem.Title, skippedItem.ChannelTitle, skippedItem.Author))
}
