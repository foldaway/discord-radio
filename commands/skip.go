package commands

import (
	"fmt"
	"strconv"

	"github.com/bottleneckco/discord-radio/models"
	"github.com/bwmarrin/discordgo"
)

func skip(s *discordgo.Session, m *discordgo.MessageCreate) {
	guildSession := safeGetGuildSession(s, m.Message.GuildID)
	guildSession.RWMutex.RLock()
	if len(guildSession.Queue) == 0 || guildSession.MusicPlayer.PlaybackState == models.PlaybackStateStopped {
		s.ChannelMessageSend(m.Message.ChannelID, fmt.Sprintf("%s nothing to skip", m.Message.Author.Mention()))
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
		guildSession.MusicPlayer.Control <- models.MusicPlayerActionStop
	} else {
		choice, err := strconv.ParseInt(m.Message.Content, 10, 64)
		if err == nil && (choice-1 >= 0 && choice-1 < int64(len(guildSession.Queue))) {
			skippedItem = guildSession.Queue[choice-1]
			guildSession.Queue = append(guildSession.Queue[:choice-1], guildSession.Queue[choice:]...)
		} else {
			s.ChannelMessageSend(m.Message.ChannelID, fmt.Sprintf("%s invalid choice", m.Message.Author.Mention()))
			guildSession.RWMutex.Unlock()
			return
		}
	}
	guildSession.RWMutex.Unlock()

	avatarURL := m.Message.Author.AvatarURL("32")

	s.ChannelMessageSendEmbed(
		m.Message.ChannelID,
		&discordgo.MessageEmbed{
			Author: &discordgo.MessageEmbedAuthor{
				Name:    "Removed from queue",
				IconURL: avatarURL,
			},
			Title: skippedItem.Title,
			Thumbnail: &discordgo.MessageEmbedThumbnail{
				URL: skippedItem.Thumbnail,
			},
			URL: fmt.Sprintf("https://www.youtube.com/watch?v=%s", skippedItem.VideoID),
			Fields: []*discordgo.MessageEmbedField{
				&discordgo.MessageEmbedField{
					Name:  "Channel",
					Value: skippedItem.ChannelTitle,
				},
			},
		})
}
