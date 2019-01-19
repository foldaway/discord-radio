package commands

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/bottleneckco/radio-clerk/util"
	"github.com/bwmarrin/discordgo"
	"github.com/evalphobia/google-tts-go/googletts"
)

func join(s *discordgo.Session, m *discordgo.MessageCreate) {
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
	VoiceConnection = voiceChannel
	s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("%s joined '%s'", m.Author.Mention(), channel.Name))
	url, _ := googletts.GetTTSURL("Hello! I'll be ready in a moment.", "en")
	if os.Getenv("BOT_UPDATE_YTDL") == "true" {
		updateCmd := exec.Command("/usr/bin/curl", "-L", "https://yt-dl.org/downloads/latest/youtube-dl", "-o", "/usr/local/bin/youtube-dl")
		updateCmd.Wait()
	}
	MusicPlayer.Play(url, "0.5")
	SafeCheckPlay()
}
