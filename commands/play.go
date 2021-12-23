package commands

import (
	"context"
	"fmt"
	"github.com/andersfylling/disgord"
	"github.com/bottleneckco/discord-radio/youtube"
	"html"
	"log"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/bottleneckco/discord-radio/models"
)

var tempSearchResultsCache = make(map[disgord.Snowflake][]youtube.PlaylistItem)

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
			b.WriteString(fmt.Sprintf("`%d.` **%s** `%s`\n", index+1, item.Title, item.Uploader))
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

	var playlistItems []youtube.PlaylistItem

	if inputURL, err := url.ParseRequestURI(messageParts[1]); err == nil {
		var isYouTubeURL = strings.HasSuffix(inputURL.Host, "youtube.com")
		var isVideoURL = isYouTubeURL && inputURL.Path == "/watch" && inputURL.Query().Has("v")
		var isPlaylistURL = isYouTubeURL && inputURL.Path == "/playlist" && inputURL.Query().Has("list")

		if isVideoURL {
			playlistItem, err := youtube.FetchSingleVideo(inputURL.String())
			if err != nil {
				m.Message.Reply(
					context.Background(),
					s,
					fmt.Sprintf("Error occurred: %s", err),
				)
				return
			}

			playlistItems = []youtube.PlaylistItem{
				playlistItem,
			}
		} else if isPlaylistURL {
			// URL
			playlistItems, err = youtube.FetchAllPlaylistItems(inputURL)
			if err != nil {
				m.Message.Reply(
					context.Background(),
					s,
					fmt.Sprintf("Error occurred: %s", err),
				)
				return
			}
		}

		guildSession.RWMutex.Lock()
		for _, playlistItem := range playlistItems {
			var queueItem = models.ConvertYouTubePlaylistItem(playlistItem)

			guildSession.Queue = append(
				guildSession.Queue,
				queueItem,
			)
		}
		guildSession.RWMutex.Unlock()

		m.Message.Reply(
			context.Background(),
			s,
			fmt.Sprintf("%s enqueued %d videos\n", m.Message.Author.Mention(), len(playlistItems)),
		)
	} else {
		maxResults, _ := strconv.ParseInt(os.Getenv("BOT_NUM_RESULTS"), 10, 64)

		var query = strings.Join(messageParts[1:], " ")

		playlistItems, err = youtube.Search(query, maxResults)
		if err != nil {
			m.Message.Reply(
				context.Background(),
				s,
				fmt.Sprintf("Error occurred: %s", err),
			)
			return
		}

		log.Printf("[PLAY] Found %d results", len(playlistItems))

		b.WriteString(fmt.Sprintf("%s, here are your search results:\n", m.Message.Author.Mention()))

		for index, item := range playlistItems {
			b.WriteString(fmt.Sprintf("`%d.` **%s** `%s`\n", index+1, html.UnescapeString(item.Title), item.Uploader))
		}
		b.WriteString("\nreply with a single number, no command needed. Anything else will cancel the search.")

		// Cache for response
		tempSearchResultsCache[m.Message.Author.ID] = playlistItems
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

		var queueItem = models.ConvertYouTubePlaylistItem(chosenItem)

		guildSession.Queue = append(guildSession.Queue, queueItem)
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
				Title: html.UnescapeString(chosenItem.Title),
				URL:   fmt.Sprintf("https://www.youtube.com/watch?v=%s", chosenItem.Id),
				Fields: []*disgord.EmbedField{
					{
						Name:  "Channel",
						Value: chosenItem.Uploader,
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
}
