package commands

import (
	"context"
	"fmt"
	"github.com/andersfylling/disgord"
	"github.com/bottleneckco/discord-radio/session"
	"log"
	"strconv"

	"github.com/bottleneckco/discord-radio/models"
)

func skip(s disgord.Session, m *disgord.MessageCreate) {
	guildSession := safeGetGuildSession(s, m.Message.GuildID)
	guildSession.RWMutex.RLock()
	if len(guildSession.Queue) == 0 || guildSession.MusicPlayer.PlaybackState == session.PlaybackStateStopped {
		m.Message.Reply(
			context.Background(),
			s,
			fmt.Sprintf("%s nothing to skip", m.Message.Author.Mention()),
		)
		guildSession.RWMutex.RUnlock()
		return
	}
	guildSession.RWMutex.RUnlock()
	guildSession.RWMutex.Lock()
	var skippedItem models.QueueItem
	if len(m.Message.Content) == 0 {
		// No args, skip current
		skippedItem = guildSession.Queue[0]
		// Queue = append(Queue[:0], Queue[1:]...)
		guildSession.MusicPlayer.Control <- session.MusicPlayerActionStop
	} else {
		choice, err := strconv.ParseInt(m.Message.Content, 10, 64)
		if err == nil && (choice-1 >= 0 && choice-1 < int64(len(guildSession.Queue))) {
			skippedItem = guildSession.Queue[choice-1]
			guildSession.Queue = append(guildSession.Queue[:choice-1], guildSession.Queue[choice:]...)
		} else {
			m.Message.Reply(
				context.Background(),
				s,
				fmt.Sprintf("%s invalid choice", m.Message.Author.Mention()),
			)
			guildSession.RWMutex.Unlock()
			return
		}
	}
	guildSession.RWMutex.Unlock()

	avatarURL, err := m.Message.Author.AvatarURL(32, false)
	if err != nil {
		log.Println(err)
		m.Message.Reply(
			context.Background(),
			s,
			"An error occurred",
		)
		return
	}

	m.Message.Reply(
		context.Background(),
		s,
		&disgord.Embed{
			Author: &disgord.EmbedAuthor{
				Name:    "Removed from queue",
				IconURL: avatarURL,
			},
			Title: skippedItem.Title,
			Thumbnail: &disgord.EmbedThumbnail{
				URL: skippedItem.Thumbnail,
			},
			URL: fmt.Sprintf("https://www.youtube.com/watch?v=%s", skippedItem.VideoID),
			Fields: []*disgord.EmbedField{
				{
					Name:  "Queued by",
					Value: skippedItem.Author,
				},
			},
		})
}
