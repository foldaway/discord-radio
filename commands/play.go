package commands

import (
	"fmt"
	"html"
	"log"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/bottleneckco/discord-radio/models"
	"github.com/bwmarrin/discordgo"

	"google.golang.org/api/youtube/v3"
)

var tempSearchResultsCache = make(map[string][]*youtube.SearchResult)

func play(s *discordgo.Session, m *discordgo.MessageCreate) {
	guildSession := safeGetGuildSession(m.Message.GuildID)
	var isNotInVoiceChannel = guildSession.VoiceConnection == nil
	if isNotInVoiceChannel {
		voiceChannelInit(s, m)
	}
	var b strings.Builder
	// Terminate if user has a pending search, forgets and uses /play again.
	if items, ok := tempSearchResultsCache[m.Message.Author.ID]; ok {
		b.WriteString(fmt.Sprintf("%s, you **already have** a pending search query:\n", m.Message.Author.Mention()))

		for index, item := range items {
			b.WriteString(fmt.Sprintf("`%d.` **%s** `%s`\n", index+1, item.Snippet.Title, item.Snippet.ChannelTitle))
		}
		b.WriteString("\nreply with a single number, no command needed. Anything else will cancel the search.")
		sentMsg, err := s.ChannelMessageSend(m.Message.ChannelID, b.String())
		if err == nil {
			go deleteMessageDelayed(s, sentMsg)
		}
		return
	}
	if url, err := url.ParseRequestURI(m.Message.Content); err == nil {
		// URL
		var videoIDs []string
		if len(url.Query().Get("list")) != 0 {
			// Playlist URL
			playlistResponse, err := youtubeService.PlaylistItems.List("contentDetails").PlaylistId(url.Query().Get("list")).MaxResults(50).Do()
			if err != nil {
				s.ChannelMessageSend(m.Message.ChannelID, fmt.Sprintf("Error occurred: %s", err))
				return
			}
			for _, playlistItem := range playlistResponse.Items {
				videoIDs = append(videoIDs, playlistItem.ContentDetails.VideoId)
			}
		} else if len(url.Query().Get("v")) != 0 {
			// Plain video URL
			videoIDs = append(videoIDs, url.Query().Get("v"))
		} else {
			s.ChannelMessageSend(m.Message.ChannelID, "Unsupported URL")
			return
		}
		youtubeListings, err := youtubeService.Videos.List("snippet").Id(strings.Join(videoIDs, ",")).Do()
		if err != nil {
			s.ChannelMessageSend(m.Message.ChannelID, fmt.Sprintf("Error occurred: %s", err))
			return
		}
		guildSession.RWMutex.Lock()
		for _, youtubeListing := range youtubeListings.Items {
			guildSession.Queue = append(guildSession.Queue, models.QueueItem{
				Title:        youtubeListing.Snippet.Title,
				ChannelTitle: youtubeListing.Snippet.ChannelTitle,
				Author:       m.Message.Author.Username,
				VideoID:      youtubeListing.Id,
				Thumbnail:    youtubeListing.Snippet.Thumbnails.Default.Url,
			})
		}
		guildSession.RWMutex.Unlock()
		s.ChannelMessageSend(m.Message.ChannelID, fmt.Sprintf("%s enqueued %d videos`\n", m.Message.Author.Mention(), len(videoIDs)))
	} else {
		maxResults, _ := strconv.ParseInt(os.Getenv("BOT_NUM_RESULTS"), 10, 64)
		call := youtubeService.Search.List("snippet").Q(m.Message.Content).MaxResults(maxResults)
		response, err := call.Do()
		if err != nil {
			s.ChannelMessageSend(m.Message.ChannelID, fmt.Sprintf("Error occurred: %s", err))
			return
		}

		log.Printf("[PLAY] Found %d results", len(response.Items))

		b.WriteString(fmt.Sprintf("%s, here are your search results:\n", m.Message.Author.Mention()))

		for index, item := range response.Items {
			b.WriteString(fmt.Sprintf("`%d.` **%s** `%s`\n", index+1, html.UnescapeString(item.Snippet.Title), item.Snippet.ChannelTitle))
		}
		b.WriteString("\nreply with a single number, no command needed. Anything else will cancel the search.")

		// Cache for response
		tempSearchResultsCache[m.Message.Author.ID] = response.Items

		var removeSecondaryHandler func()

		removeSecondaryHandler = s.AddHandler(func(ss *discordgo.Session, mm *discordgo.MessageCreate) {
			if m.Message.Author.ID != mm.Message.Author.ID {
				return
			}
			if strings.HasPrefix(mm.Message.Content, os.Getenv("BOT_COMMAND_PREFIX")) {
				return
			}
			choice, err := strconv.ParseInt(mm.Message.Content, 10, 64)
			if err == nil && (choice-1 < 0 || choice-1 >= int64(len(tempSearchResultsCache[mm.Message.Author.ID]))) {
				// Valid integer but out of bounds
				ss.ChannelMessageSend(mm.Message.ChannelID, fmt.Sprintf("%s Invalid choice, try again", mm.Message.Author.Mention()))
				return
			} else if err == nil {
				chosenItem := tempSearchResultsCache[mm.Message.Author.ID][choice-1]
				guildSession.RWMutex.Lock()
				guildSession.Queue = append(guildSession.Queue, models.QueueItem{
					Title:        html.UnescapeString(chosenItem.Snippet.Title),
					ChannelTitle: chosenItem.Snippet.ChannelTitle,
					Author:       mm.Message.Author.Username,
					VideoID:      chosenItem.Id.VideoId,
					Thumbnail:    chosenItem.Snippet.Thumbnails.Default.Url,
				})
				guildSession.RWMutex.Unlock()
				avatarURL := m.Message.Author.AvatarURL("32")
				ss.ChannelMessageSendEmbed(
					mm.Message.ChannelID,
					&discordgo.MessageEmbed{
						Author: &discordgo.MessageEmbedAuthor{
							Name:    "Added to queue",
							IconURL: avatarURL,
						},
						Title: html.UnescapeString(chosenItem.Snippet.Title),
						Thumbnail: &discordgo.MessageEmbedThumbnail{
							URL: chosenItem.Snippet.Thumbnails.Default.Url,
						},
						URL: fmt.Sprintf("https://www.youtube.com/watch?v=%s", chosenItem.Id.VideoId),
						Fields: []*discordgo.MessageEmbedField{
							&discordgo.MessageEmbedField{
								Name:  "Channel",
								Value: chosenItem.Snippet.ChannelTitle,
							},
						},
					})
			} else {
				ss.ChannelMessageSend(mm.Message.ChannelID, fmt.Sprintf("%s Search cancelled", mm.Message.Author.Mention()))
			}
			delete(tempSearchResultsCache, mm.Message.Author.ID)
			removeSecondaryHandler()

			if isNotInVoiceChannel {
				go guildSession.Loop()
			}
		})
	}

	sentMsg, err := s.ChannelMessageSend(m.Message.ChannelID, b.String())
	if err == nil {
		go deleteMessageDelayed(s, sentMsg)
	}
}
