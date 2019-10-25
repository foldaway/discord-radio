package commands

import (
	"fmt"
	"strconv"

	"github.com/andersfylling/disgord"
	"github.com/bottleneckco/discord-radio/models"
)

func skip(s disgord.Session, m *disgord.MessageCreate) {
	guildSession := safeGetGuildSession(m.Message.GuildID)
	guildSession.RWMutex.RLock()
	if len(guildSession.Queue) == 0 || !guildSession.MusicPlayer.IsPlaying {
		s.SendMsg(m.Message.ChannelID, fmt.Sprintf("%s nothing to skip", m.Message.Author.Mention()))
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
		guildSession.MusicPlayer.Control <- models.MusicPlayerActionSkip
	} else {
		choice, err := strconv.ParseInt(m.Message.Content, 10, 64)
		if err == nil && (choice-1 >= 0 && choice-1 < int64(len(guildSession.Queue))) {
			skippedItem = guildSession.Queue[choice-1]
			guildSession.Queue = append(guildSession.Queue[:choice-1], guildSession.Queue[choice:]...)
		} else {
			s.SendMsg(m.Message.ChannelID, fmt.Sprintf("%s invalid choice", m.Message.Author.Mention()))
			guildSession.RWMutex.Unlock()
			return
		}
	}
	guildSession.RWMutex.Unlock()

	s.SendMsg(m.Message.ChannelID, &disgord.CreateMessageParams{
		Embed: &disgord.Embed{
			Author: &disgord.EmbedAuthor{
				Name:    "Removed from queue",
				IconURL: fmt.Sprintf("https://cdn.discordapp.com/embed/avatars/%d.png?size=%d", m.Message.Author.Discriminator%5, 32),
			},
			Title: skippedItem.Title,
			Thumbnail: &disgord.EmbedThumbnail{
				URL: skippedItem.Thumbnail,
			},
			URL: fmt.Sprintf("https://www.youtube.com/watch?v=%s", skippedItem.VideoID),
			Fields: []*disgord.EmbedField{
				&disgord.EmbedField{
					Name:  "Channel",
					Value: skippedItem.ChannelTitle,
				},
			},
		},
	})
}
