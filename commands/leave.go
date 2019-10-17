package commands

import (
	"fmt"

	"github.com/andersfylling/disgord"
	"github.com/bottleneckco/discord-radio/util"
)

func leave(s disgord.Session, m *disgord.MessageCreate) {
	if guildSession, ok := GuildSessionMap[m.Message.GuildID]; ok {
		voiceState, err := util.FindUserVoiceState(s, m.Message.GuildID, m.Message.Author.ID)
		if err != nil {
			s.SendMsg(m.Message.ChannelID, fmt.Sprintf("%s you are not in a voice channel", m.Message.Author.Mention()))
			return
		}
		channel, err := s.GetChannel(voiceState.ChannelID)
		if err != nil {
			s.SendMsg(m.Message.ChannelID, fmt.Sprintf("%s error occurred: %s", m.Message.Author.Mention(), err))
			return
		}
		// Actual disconnect code
		var tempVoiceConn = guildSession.VoiceConnection
		guildSession.VoiceConnection = nil

		guildSession.Mutex.Lock()
		guildSession.Queue = guildSession.Queue[0:0]
		guildSession.Mutex.Unlock()
		guildSession.MusicPlayer.Close <- struct{}{}
		tempVoiceConn.Close()
		delete(GuildSessionMap, m.Message.GuildID)

		s.SendMsg(m.Message.ChannelID, fmt.Sprintf("%s left '%s'", m.Message.Author.Mention(), channel.Name))
	} else {
		s.SendMsg(m.Message.ChannelID, fmt.Sprintf("%s not in voice channel", m.Message.Author.Mention()))
	}
}
