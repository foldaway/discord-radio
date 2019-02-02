package commands

import (
	"fmt"
	"strconv"

	"github.com/bottleneckco/radio-clerk/models"

	"github.com/bwmarrin/discordgo"
)

func skip(s *discordgo.Session, m *discordgo.MessageCreate) {
	if len(Queue) == 0 || !MusicPlayer.IsPlaying {
		s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("%s nothing to skip", m.Author.Mention()))
		return
	}
	var skippedItem models.QueueItem
	if len(m.Content) == 0 {
		// No args, skip current
		skippedItem = Queue[0]
		// Queue = append(Queue[:0], Queue[1:]...)
		MusicPlayer.Control <- Skip
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

	s.ChannelMessageSendEmbed(m.ChannelID, &discordgo.MessageEmbed{
		Author: &discordgo.MessageEmbedAuthor{
			Name:    "Removed from queue",
			IconURL: m.Author.AvatarURL("32"),
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
