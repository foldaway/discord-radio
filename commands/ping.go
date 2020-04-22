package commands

import (
	"github.com/andersfylling/disgord"
	"github.com/bottleneckco/discord-radio/ctx"
)

func ping(s disgord.Session, m *disgord.MessageCreate) {
	s.SendMsg(ctx.Ctx, m.Message.ChannelID, "pong")
}
