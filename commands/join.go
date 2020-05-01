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
	voiceChannelInit(s, m)
	guildSession := safeGetGuildSession(m.Message.GuildID)
	go guildSession.Loop()
}

func voiceChannelInit(s *discordgo.Session, m *discordgo.MessageCreate) {
	var guildSession = safeGetGuildSession(m.Message.GuildID)
	var userVoiceState *discordgo.VoiceState
	var err error

	userVoiceState, err = util.FindUserVoiceState(s, m.Message.Author.ID)

	channel, err := s.Channel(userVoiceState.ChannelID)
	if err != nil {
		log.Println(err)
		s.ChannelMessageSend(m.Message.ChannelID, fmt.Sprintf("%s error occurred: %s", m.Message.Author.Mention(), err))
		return
	}

	voiceChannel, err := s.ChannelVoiceJoin(m.Message.GuildID, userVoiceState.ChannelID, false, true)
	if err != nil {
		log.Println(err)
		s.ChannelMessageSend(m.Message.ChannelID, fmt.Sprintf("%s error occurred: %s", m.Message.Author.Mention(), err))
		return
	}

	// Update guildSession
	guildSession.VoiceConnection = voiceChannel
	guildSession.VoiceChannelID = channel.ID

	s.ChannelMessageSend(m.Message.ChannelID, fmt.Sprintf("%s joined '%s'", m.Message.Author.Mention(), channel.Name))
	log.Printf(fmt.Sprintf("%s joined '%s' guild '%s'\n", m.Message.Author.Mention(), channel.Name, m.Message.GuildID))

	url, _ := googletts.GetTTSURL("Ready", "en")
	if os.Getenv("BOT_UPDATE_YTDL") == "true" {
		s.ChannelMessageSend(m.Message.ChannelID, fmt.Sprintf("%s updating youtube-dl binary, give me some time.", m.Message.Author.Mention()))

		// Update youtube-dl
		ytdlCmd := exec.Command("/usr/bin/curl", "-L", "https://yt-dl.org/downloads/latest/youtube-dl", "-o", "/usr/local/bin/youtube-dl")
		ytdlCmd.Stdout = os.Stdout
		ytdlCmd.Stderr = os.Stderr
		ytdlCmd.Run()

		s.ChannelMessageSend(m.Message.ChannelID, fmt.Sprintf("%s done!", m.Message.Author.Mention()))
	}

	// Clear youtube-dl cache
	ytdlCmd := exec.Command("youtube-dl", "--rm-cache-dir")
	ytdlCmd.Stdout = os.Stdout
	ytdlCmd.Stderr = os.Stderr
	ytdlCmd.Run()

	guildSession.PlayURL(url, 0.5)
}
