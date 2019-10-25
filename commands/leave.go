package commands

import (
	"fmt"
	"log"

	"github.com/andersfylling/disgord"
	"github.com/bottleneckco/discord-radio/vscache"
)

func leave(s disgord.Session, m *disgord.MessageCreate) {
	if guildSession, ok := GuildSessionMap[m.Message.GuildID]; ok {
		voiceStateCache, ok := vscache.FindUserVoiceState(m.Message.Author.ID)
		if !ok {
			log.Println("No voice state cached")
			s.SendMsg(m.Message.ChannelID, fmt.Sprintf("%s you are not in a voice channel", m.Message.Author.Mention()))
			return
		}
		channel, err := s.GetChannel(voiceStateCache.Current.ChannelID)
		if err != nil {
			s.SendMsg(m.Message.ChannelID, fmt.Sprintf("%s error occurred: %s", m.Message.Author.Mention(), err))
			return
		}
		// Actual disconnect code
		var tempVoiceConn = guildSession.VoiceConnection
		guildSession.VoiceConnection = nil

		guildSession.RWMutex.Lock()
		guildSession.Queue = guildSession.Queue[0:0]
		guildSession.RWMutex.Unlock()
		guildSession.MusicPlayer.Close <- struct{}{}
		tempVoiceConn.Close()
		delete(GuildSessionMap, m.Message.GuildID)

		s.SendMsg(m.Message.ChannelID, fmt.Sprintf("%s left '%s'", m.Message.Author.Mention(), channel.Name))
	} else {
		s.SendMsg(m.Message.ChannelID, fmt.Sprintf("%s not in voice channel", m.Message.Author.Mention()))
	}
}
