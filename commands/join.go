package commands

import (
	"fmt"
	"log"
	"os"
	"os/exec"

	"github.com/bottleneckco/discord-radio/util"
	"github.com/bwmarrin/discordgo"
	"github.com/evalphobia/google-tts-go/googletts"
)

func join(s *discordgo.Session, m *discordgo.MessageCreate) {
	guildSession := safeGetGuildSession(m.GuildID)
	voiceState, err := util.FindUserVoiceState(s, m.Author.ID)
	if err != nil {
		s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("%s you are not in a voice channel", m.Author.Mention()))
		return
	}
	channel, err := s.Channel(voiceState.ChannelID)
	if err != nil {
		s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("%s error occurred: %s", m.Author.Mention(), err))
		return
	}
	voiceChannel, err := s.ChannelVoiceJoin(voiceState.GuildID, voiceState.ChannelID, false, true)
	if err != nil {
		s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("%s error occurred: %s", m.Author.Mention(), err))
		return
	}
	guildSession.VoiceConnection = voiceChannel
	s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("%s joined '%s'", m.Author.Mention(), channel.Name))
	log.Printf(fmt.Sprintf("%s joined '%s' guild '%s'\n", m.Author.Mention(), channel.Name, m.GuildID))
	url, _ := googletts.GetTTSURL("Ready", "en")
	if os.Getenv("BOT_UPDATE_YTDL") == "true" {
		updateCmd := exec.Command("/usr/bin/curl", "-L", "https://yt-dl.org/downloads/latest/youtube-dl", "-o", "/usr/local/bin/youtube-dl")
		updateCmd.Stdout = os.Stdout
		updateCmd.Stderr = os.Stderr
		updateCmd.Run()
	}
	guildSession.PlayURL(url, 0.5)
	SafeCheckPlay(guildSession)
}
