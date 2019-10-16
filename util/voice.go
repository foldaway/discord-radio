package util

import (
	"errors"

	"github.com/bwmarrin/discordgo"
)

// FindUserVoiceState find the voice state of a user
func FindUserVoiceState(session *discordgo.Session, guildID string, userid string) (*discordgo.VoiceState, error) {
	guild, err := session.Guild(guildID)
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
func GetUsersInVoiceChannel(session *discordgo.Session, guildID, channelID string) ([]*discordgo.VoiceState, error) {
	var results []*discordgo.VoiceState
	guild, err := session.Guild(guildID)
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
