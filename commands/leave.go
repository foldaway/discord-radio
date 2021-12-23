package commands

import (
	"context"
	"fmt"
	"github.com/andersfylling/disgord"
	"github.com/bottleneckco/discord-radio/session"
	"log"

	"github.com/bottleneckco/discord-radio/util"
)

func leave(s disgord.Session, m *disgord.MessageCreate) {
	if guildSession, ok := GuildSessionMap[m.Message.GuildID]; ok {
		userVoiceState, ok := util.GlobalVoiceStateCache.VoiceStates[m.Message.Author.ID]
		if !ok {
			log.Println("No voice state cached")
			m.Message.Reply(
				context.Background(),
				s,
				fmt.Sprintf("%s you are not in a voice channel", m.Message.Author.Mention()),
			)
			return
		}
		channel, err := s.Channel(userVoiceState.ChannelID).Get()
		if err != nil {
			m.Message.Reply(
				context.Background(),
				s,
				fmt.Sprintf("%s error occurred: %s", m.Message.Author.Mention(), err),
			)
			return
		}

		// Actual disconnect code

		guildSession.RWMutex.Lock()
		guildSession.Queue = guildSession.Queue[0:0]
		guildSession.RWMutex.Unlock()
		if guildSession.MusicPlayer.PlaybackState == session.PlaybackStatePlaying {
			guildSession.MusicPlayer.Control <- session.MusicPlayerActionStop
		}

		if guildSession.VoiceConnection != nil {
			var voiceConnection = *guildSession.VoiceConnection
			voiceConnection.Close()
		}

		delete(GuildSessionMap, m.Message.GuildID)

		m.Message.Reply(
			context.Background(),
			s,
			fmt.Sprintf("%s left '%s'", m.Message.Author.Mention(), channel.Name),
		)
	} else {
		m.Message.Reply(
			context.Background(),
			s,
			fmt.Sprintf("%s not in voice channel", m.Message.Author.Mention()),
		)
	}
}
