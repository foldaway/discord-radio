package commands

import (
	"context"
	"github.com/andersfylling/disgord"
)

func ping(s disgord.Session, m *disgord.MessageCreate) {
	m.Message.Reply(context.Background(), s, "pong")
}
