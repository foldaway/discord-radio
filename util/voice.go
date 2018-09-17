package util

import (
	"errors"

	"github.com/bwmarrin/discordgo"
)

func FindUserVoiceState(session *discordgo.Session, userid string) (*discordgo.VoiceState, error) {
	for _, guild := range session.State.Guilds {
		for _, vs := range guild.VoiceStates {
			if vs.UserID == userid {
				return vs, nil
			}
		}
	}
	return nil, errors.New("Could not find user's voice state")
}

func GetUsersInVoiceChannel(session *discordgo.Session, channelID string) []*discordgo.VoiceState {
	var results []*discordgo.VoiceState
	for _, guild := range session.State.Guilds {
		for _, vs := range guild.VoiceStates {
			if vs.ChannelID == channelID {
				results = append(results, vs)
			}
		}
	}
	return results
}
