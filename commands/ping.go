package commands

import "github.com/andersfylling/disgord"

func ping(s disgord.Session, m *disgord.MessageCreate) {
	s.SendMsg(m.Message.ChannelID, "pong")
}
