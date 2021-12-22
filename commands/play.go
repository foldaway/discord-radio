package commands

import (
	"context"
	"fmt"
	"github.com/andersfylling/disgord"
	"html"
	"log"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/bottleneckco/discord-radio/models"
	"google.golang.org/api/youtube/v3"
)

var tempSearchResultsCache = make(map[disgord.Snowflake][]*youtube.SearchResult)

func play(s disgord.Session, m *disgord.MessageCreate) {
	guildSession := findOrCreateGuildSession(s, m.Message.GuildID)
	var isNotInVoiceChannel = guildSession.VoiceConnection == nil
	if isNotInVoiceChannel {
		voiceChannelInit(s, m)
	}

	var messageParts = strings.Split(m.Message.Content, " ")

	if len(messageParts) == 1 {
		m.Message.Reply(
			context.Background(),
			s,
			fmt.Sprintf("%s invalid syntax", m.Message.Author.Mention()),
		)
		return
	}

	var b strings.Builder
	// Terminate if user has a pending search, forgets and uses /play again.
	if items, ok := tempSearchResultsCache[m.Message.Author.ID]; ok {
		b.WriteString(fmt.Sprintf("%s, you **already have** a pending search query:\n", m.Message.Author.Mention()))

		for index, item := range items {
			b.WriteString(fmt.Sprintf("`%d.` **%s** `%s`\n", index+1, item.Snippet.Title, item.Snippet.ChannelTitle))
		}
		b.WriteString("\nreply with a single number, no command needed. Anything else will cancel the search.")
		sentMsg, err := m.Message.Reply(
			context.Background(),
			s,
			b.String(),
		)
		if err == nil {
			go deleteMessageDelayed(s, sentMsg)
		}
		return
	}
	if url, err := url.ParseRequestURI(messageParts[1]); err == nil {
		// URL
		var videoIDs []string
		if len(url.Query().Get("list")) != 0 {
			// Playlist URL
			playlistResponse, err := youtubeService.PlaylistItems.List("contentDetails").PlaylistId(url.Query().Get("list")).MaxResults(50).Do()
			if err != nil {
				m.Message.Reply(
					context.Background(),
					s,
					fmt.Sprintf("Error occurred: %s", err),
				)
				return
			}
			for _, playlistItem := range playlistResponse.Items {
				videoIDs = append(videoIDs, playlistItem.ContentDetails.VideoId)
			}
		} else if len(url.Query().Get("v")) != 0 {
			// Plain video URL
			videoIDs = append(videoIDs, url.Query().Get("v"))
		} else {
			m.Message.Reply(
				context.Background(),
				s,
				"Unsupported URL",
			)
			return
		}
		youtubeListings, err := youtubeService.Videos.List("snippet").Id(strings.Join(videoIDs, ",")).Do()
		if err != nil {
			m.Message.Reply(
				context.Background(),
				s,
				fmt.Sprintf("Error occurred: %s", err),
			)
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
		m.Message.Reply(
			context.Background(),
			s,
			fmt.Sprintf("%s enqueued %d videos`\n", m.Message.Author.Mention(), len(videoIDs)),
		)
	} else {
		maxResults, _ := strconv.ParseInt(os.Getenv("BOT_NUM_RESULTS"), 10, 64)

		var query = strings.Join(messageParts[1:], " ")

		call := youtubeService.Search.List("snippet").Q(query).MaxResults(maxResults)
		response, err := call.Do()
		if err != nil {
			m.Message.Reply(
				context.Background(),
				s,
				fmt.Sprintf("Error occurred: %s", err),
			)
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
	}

	sentMsg, err := m.Message.Reply(
		context.Background(),
		s,
		b.String(),
	)
	if err == nil {
		go deleteMessageDelayed(s, sentMsg)
	}
}

func playSecondaryHandler(s disgord.Session, m *disgord.MessageCreate) {
	guildSession := findOrCreateGuildSession(s, m.Message.GuildID)
	if strings.HasPrefix(m.Message.Content, os.Getenv("BOT_COMMAND_PREFIX")) {
		return
	}
	choice, err := strconv.ParseInt(m.Message.Content, 10, 64)
	if err == nil && (choice-1 < 0 || choice-1 >= int64(len(tempSearchResultsCache[m.Message.Author.ID]))) {
		// Valid integer but out of bounds
		m.Message.Reply(
			context.Background(),
			s,
			fmt.Sprintf("%s Invalid choice, try again", m.Message.Author.Mention()),
		)
		return
	} else if err == nil {
		chosenItem := tempSearchResultsCache[m.Message.Author.ID][choice-1]
		guildSession.RWMutex.Lock()
		guildSession.Queue = append(guildSession.Queue, models.QueueItem{
			Title:        html.UnescapeString(chosenItem.Snippet.Title),
			ChannelTitle: chosenItem.Snippet.ChannelTitle,
			Author:       m.Message.Author.Username,
			VideoID:      chosenItem.Id.VideoId,
			Thumbnail:    chosenItem.Snippet.Thumbnails.Default.Url,
		})
		guildSession.RWMutex.Unlock()
		avatarURL, err := m.Message.Author.AvatarURL(32, false)
		if err != nil {
			log.Println(err)
			m.Message.Reply(
				context.Background(),
				s,
				fmt.Sprintf("%s Error occurred: %s", m.Message.Author.Mention(), err),
			)
			return
		}
		m.Message.Reply(
			context.Background(),
			s,
			disgord.Embed{
				Author: &disgord.EmbedAuthor{
					Name:    "Added to queue",
					IconURL: avatarURL,
				},
				Title: html.UnescapeString(chosenItem.Snippet.Title),
				Thumbnail: &disgord.EmbedThumbnail{
					URL: chosenItem.Snippet.Thumbnails.Default.Url,
				},
				URL: fmt.Sprintf("https://www.youtube.com/watch?v=%s", chosenItem.Id.VideoId),
				Fields: []*disgord.EmbedField{
					{
						Name:  "Channel",
						Value: chosenItem.Snippet.ChannelTitle,
					},
				},
			})
	} else {
		m.Message.Reply(
			context.Background(),
			s,
			fmt.Sprintf("%s Search cancelled", m.Message.Author.Mention()),
		)
	}
	delete(tempSearchResultsCache, m.Message.Author.ID)

	var isNotInVoiceChannel = guildSession.VoiceConnection == nil
	if isNotInVoiceChannel {
		go guildSession.Loop()
	}
}
