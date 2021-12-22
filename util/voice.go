package util

import (
	"github.com/andersfylling/disgord"
	"sync"
)

var (
	GlobalVoiceStateCache = VoiceStateCache{
		VoiceStates: make(map[disgord.Snowflake]*disgord.VoiceState),
	}
)

// VoiceStateCache manual implementation of a voice state cache
// adapted from https://github.com/andersfylling/disgord/issues/288#issuecomment-657728535
type VoiceStateCache struct {
	VoiceStates map[disgord.Snowflake]*disgord.VoiceState
	mutex       sync.RWMutex
}

func (c *VoiceStateCache) Handle(s disgord.Session, voiceState *disgord.VoiceState) {
	user, err := s.User(voiceState.UserID).Get()
	if err != nil || user.Bot {
		return
	}

	if voiceState.ChannelID.IsZero() {
		// This is a leave voice channel event
		delete(c.VoiceStates, voiceState.UserID)
	} else {
		// Either freshly joined or switched voice channel
		c.VoiceStates[voiceState.UserID] = voiceState
	}
}
