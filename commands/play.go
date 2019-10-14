package commands

import (
	"fmt"
	"log"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/bottleneckco/discord-radio/models"

	"google.golang.org/api/youtube/v3"

	"github.com/bwmarrin/discordgo"
)

var tempSearchResultsCache = make(map[string][]*youtube.SearchResult)

func play(s *discordgo.Session, m *discordgo.MessageCreate) {
	guildSession := safeGetGuildSession(m.GuildID)
	if guildSession.VoiceConnection == nil {
		s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("%s I am not in any voice channel", m.Author.Mention()))
		return
	}
	var b strings.Builder
	// Terminate if user has a pending search, forgets and uses /play again.
	if items, ok := tempSearchResultsCache[m.Author.ID]; ok {
		b.WriteString(fmt.Sprintf("%s, you **already have** a pending search query:\n", m.Author.Mention()))

		for index, item := range items {
			b.WriteString(fmt.Sprintf("`%d.` **%s** `%s`\n", index+1, item.Snippet.Title, item.Snippet.ChannelTitle))
		}
		b.WriteString("\nreply with a single number, no command needed. Anything else will cancel the search.")
		sentMsg, err := s.ChannelMessageSend(m.ChannelID, b.String())
		if err == nil {
			go deleteMessageDelayed(s, sentMsg)
		}
		return
	}
	if url, err := url.ParseRequestURI(m.Content); err == nil {
		// URL
		var videoIDs []string
		if len(url.Query().Get("list")) != 0 {
			// Playlist URL
			playlistResponse, err := youtubeService.PlaylistItems.List("contentDetails").PlaylistId(url.Query().Get("list")).MaxResults(50).Do()
			if err != nil {
				s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("Error occurred: %s", err))
				return
			}
			for _, playlistItem := range playlistResponse.Items {
				videoIDs = append(videoIDs, playlistItem.ContentDetails.VideoId)
			}
		} else if len(url.Query().Get("v")) != 0 {
			// Plain video URL
			videoIDs = append(videoIDs, url.Query().Get("v"))
		} else {
			s.ChannelMessageSend(m.ChannelID, "Unsupported URL")
			return
		}
		youtubeListings, err := youtubeService.Videos.List("snippet").Id(strings.Join(videoIDs, ",")).Do()
		if err != nil {
			s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("Error occurred: %s", err))
			return
		}
		guildSession.Mutex.Lock()
		for _, youtubeListing := range youtubeListings.Items {
			guildSession.Queue = append(guildSession.Queue, models.QueueItem{
				Title:        youtubeListing.Snippet.Title,
				ChannelTitle: youtubeListing.Snippet.ChannelTitle,
				Author:       m.Author.Username,
				VideoID:      youtubeListing.Id,
				Thumbnail:    youtubeListing.Snippet.Thumbnails.Default.Url,
			})
		}
		guildSession.Mutex.Unlock()
		SafeCheckPlay(guildSession)
		s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("%s enqueued %d videos`\n", m.Author.Mention(), len(videoIDs)))
	} else {
		maxResults, _ := strconv.ParseInt(os.Getenv("BOT_NUM_RESULTS"), 10, 64)
		call := youtubeService.Search.List("snippet").Q(m.Content).MaxResults(maxResults)
		response, err := call.Do()
		if err != nil {
			s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("Error occurred: %s", err))
			return
		}

		log.Printf("[PLAY] Found %d results", len(response.Items))

		b.WriteString(fmt.Sprintf("%s, here are your search results:\n", m.Author.Mention()))

		for index, item := range response.Items {
			b.WriteString(fmt.Sprintf("`%d.` **%s** `%s`\n", index+1, item.Snippet.Title, item.Snippet.ChannelTitle))
		}
		b.WriteString("\nreply with a single number, no command needed. Anything else will cancel the search.")

		// Cache for response
		tempSearchResultsCache[m.Author.ID] = response.Items

		var awaitFuncRemove func()

		awaitFuncRemove = s.AddHandler(func(ss *discordgo.Session, mm *discordgo.MessageCreate) {
			if m.Author.ID != mm.Author.ID {
				return
			}
			if strings.HasPrefix(mm.Content, os.Getenv("BOT_COMMAND_PREFIX")) {
				return
			}
			choice, err := strconv.ParseInt(mm.Content, 10, 64)
			if err == nil && (choice-1 < 0 || choice-1 >= int64(len(tempSearchResultsCache[mm.Author.ID]))) {
				// Valid integer but out of bounds
				ss.ChannelMessageSend(mm.ChannelID, fmt.Sprintf("%s Invalid choice, try again", mm.Author.Mention()))
				return
			} else if err == nil {
				chosenItem := tempSearchResultsCache[mm.Author.ID][choice-1]
				guildSession.Mutex.Lock()
				guildSession.Queue = append(guildSession.Queue, models.QueueItem{
					Title:        chosenItem.Snippet.Title,
					ChannelTitle: chosenItem.Snippet.ChannelTitle,
					Author:       mm.Author.Username,
					VideoID:      chosenItem.Id.VideoId,
					Thumbnail:    chosenItem.Snippet.Thumbnails.Default.Url,
				})
				guildSession.Mutex.Unlock()
				ss.ChannelMessageSendEmbed(mm.ChannelID, &discordgo.MessageEmbed{
					Author: &discordgo.MessageEmbedAuthor{
						Name:    "Added to queue",
						IconURL: m.Author.AvatarURL("32"),
					},
					Title: chosenItem.Snippet.Title,
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
				ss.ChannelMessageSend(mm.ChannelID, fmt.Sprintf("%s Search cancelled", mm.Author.Mention()))
			}
			delete(tempSearchResultsCache, mm.Author.ID)
			awaitFuncRemove()
			SafeCheckPlay(guildSession)
		})
	}

	sentMsg, err := s.ChannelMessageSend(m.ChannelID, b.String())
	if err == nil {
		go deleteMessageDelayed(s, sentMsg)
	}
}
