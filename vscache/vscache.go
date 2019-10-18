package vscache

import (
	"log"

	"github.com/andersfylling/disgord"
)

// VoiceStateCache a data structure used to cache the VoiceState
type VoiceStateCache struct {
	Previous *disgord.VoiceState
	Current  *disgord.VoiceState
}

var userVoiceStateCache = make(map[disgord.Snowflake]*VoiceStateCache)

func upsertVoiceState(userID disgord.Snowflake, vs *disgord.VoiceState) {
	vsc, ok := userVoiceStateCache[userID]
	if !ok {
		vsc = &VoiceStateCache{}
		userVoiceStateCache[userID] = vsc
	}
	vsc.Previous = vsc.Current
	vsc.Current = vs
}

// HandleVSU handle VoiceStateUpdate event
func HandleVSU(s disgord.Session, vsu *disgord.VoiceStateUpdate) {
	upsertVoiceState(vsu.UserID, vsu.VoiceState)
}

// PreloadGuilds preload guilds
func PreloadGuilds(s *disgord.Client) {
	guilds, err := s.GetCurrentUserGuilds(&disgord.GetCurrentUserGuildsParams{})
	if err != nil {
		log.Println(err)
		return
	}
	log.Printf("Preloading %d guilds\n", len(guilds))
	for _, partialGuild := range guilds {
		guild, err := s.GetGuild(partialGuild.ID)
		if err != nil {
			log.Println(err)
			continue
		}
		log.Printf("Preloading %d VSs in guild %s\n", len(guild.VoiceStates), guild.Name)
		for _, vs := range guild.VoiceStates {
			if _, isExist := userVoiceStateCache[vs.UserID]; !isExist {
				upsertVoiceState(vs.UserID, vs)
			}
		}
	}
}

// FindUserVoiceState find the voice state of a user
func FindUserVoiceState(userID disgord.Snowflake) (*VoiceStateCache, bool) {
	vs, ok := userVoiceStateCache[userID]
	return vs, ok
}

// GetUsersInVoiceChannel get users in a voice channel
func GetUsersInVoiceChannel(guildID, channelID disgord.Snowflake) []*VoiceStateCache {
	var results []*VoiceStateCache
	for _, vs := range userVoiceStateCache {
		if vs.Current.ChannelID == channelID {
			results = append(results, vs)
		}
	}
	return results
}
