package commands

import (
	"fmt"
	"log"

	"github.com/bottleneckco/discord-radio/util"
	"github.com/bwmarrin/discordgo"
)

func leave(s *discordgo.Session, m *discordgo.MessageCreate) {
	if guildSession, ok := GuildSessionMap[m.Message.GuildID]; ok {
		userVoiceState, err := util.FindUserVoiceState(s, m.Message.Author.ID)
		if err != nil {
			log.Println("No voice state cached")
			s.ChannelMessageSend(m.Message.ChannelID, fmt.Sprintf("%s you are not in a voice channel", m.Message.Author.Mention()))
			return
		}
		channel, err := s.Channel(userVoiceState.ChannelID)
		if err != nil {
			s.ChannelMessageSend(m.Message.ChannelID, fmt.Sprintf("%s error occurred: %s", m.Message.Author.Mention(), err))
			return
		}

		// Actual disconnect code
		var tempVoiceConn = guildSession.VoiceConnection
		guildSession.VoiceConnection = nil

		guildSession.RWMutex.Lock()
		guildSession.Queue = guildSession.Queue[0:0]
		guildSession.RWMutex.Unlock()
		guildSession.MusicPlayer.Close <- struct{}{}
		tempVoiceConn.Disconnect()
		delete(GuildSessionMap, m.Message.GuildID)

		s.ChannelMessageSend(m.Message.ChannelID, fmt.Sprintf("%s left '%s'", m.Message.Author.Mention(), channel.Name))
	} else {
		s.ChannelMessageSend(m.Message.ChannelID, fmt.Sprintf("%s not in voice channel", m.Message.Author.Mention()))
	}
}
