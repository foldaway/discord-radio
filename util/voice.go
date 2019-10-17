package util

import (
	"errors"

	"github.com/andersfylling/disgord"
)

// FindUserVoiceState find the voice state of a user
func FindUserVoiceState(session disgord.Session, guildID, userid disgord.Snowflake) (*disgord.VoiceState, error) {
	guild, err := session.GetGuild(guildID)
	if err != nil {
		return nil, err
	}
	for _, vs := range guild.VoiceStates {
		if vs.UserID == userid {
			return vs, nil
		}
	}
	return nil, errors.New("Could not find user's voice state")
}

// GetUsersInVoiceChannel get users in a voice channel
func GetUsersInVoiceChannel(session disgord.Session, guildID, channelID disgord.Snowflake) ([]*disgord.VoiceState, error) {
	var results []*disgord.VoiceState
	guild, err := session.GetGuild(guildID)
	if err != nil {
		return results, err
	}
	for _, vs := range guild.VoiceStates {
		if vs.ChannelID == channelID {
			results = append(results, vs)
		}
	}
	return results, nil
}
