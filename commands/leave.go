package commands

import (
	"fmt"

	"github.com/bottleneckco/discord-radio/util"
	"github.com/bwmarrin/discordgo"
)

func leave(s *discordgo.Session, m *discordgo.MessageCreate) {
	if guildSession, ok := GuildSessionMap[m.GuildID]; ok {
		voiceState, err := util.FindUserVoiceState(s, m.GuildID, m.Author.ID)
		if err != nil {
			s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("%s you are not in a voice channel", m.Author.Mention()))
			return
		}
		channel, err := s.Channel(voiceState.ChannelID)
		if err != nil {
			s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("%s error occurred: %s", m.Author.Mention(), err))
			return
		}
		// Actual disconnect code
		var tempVoiceConn = guildSession.VoiceConnection
		guildSession.VoiceConnection = nil

		guildSession.Mutex.Lock()
		guildSession.Queue = guildSession.Queue[0:0]
		guildSession.Mutex.Unlock()
		guildSession.MusicPlayer.Close <- struct{}{}
		tempVoiceConn.Disconnect()
		delete(GuildSessionMap, m.GuildID)

		s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("%s left '%s'", m.Author.Mention(), channel.Name))
	} else {
		s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("%s not in voice channel", m.Author.Mention()))
	}
}
