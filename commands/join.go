package commands

import (
	"context"
	"fmt"
	"github.com/andersfylling/disgord"
	"log"
	"os"
	"os/exec"

	"github.com/bottleneckco/discord-radio/util"
)

func join(s disgord.Session, m *disgord.MessageCreate) {
	voiceChannelInit(s, m)
	findOrCreateGuildSession(s, m.Message.GuildID)
}

func voiceChannelInit(s disgord.Session, m *disgord.MessageCreate) {
	var guildSession = findOrCreateGuildSession(s, m.Message.GuildID)
	var userVoiceState *disgord.VoiceState
	var err error

	userVoiceState, ok := util.GlobalVoiceStateCache.VoiceStates[m.Message.Author.ID]
	if !ok {
		m.Message.Reply(
			context.Background(),
			s,
			fmt.Sprintf("%s you are not in a voice channel.", m.Message.Author.Mention()),
		)
		return
	}
	if userVoiceState == nil {
		return
	}

	channel, err := s.Channel(userVoiceState.ChannelID).Get()
	if err != nil {
		log.Println(err)
		m.Message.Reply(
			context.Background(),
			s,
			fmt.Sprintf("%s error occurred: %s", m.Message.Author.Mention(), err),
		)
		return
	}

	voiceConnection, err := s.
		Guild(m.Message.GuildID).
		VoiceChannel(userVoiceState.ChannelID).
		Connect(false, true)

	if err != nil {
		log.Println(err)
		m.Message.Reply(
			context.Background(),
			s,
			fmt.Sprintf("%s error occurred: %s", m.Message.Author.Mention(), err),
		)
		return
	}

	// Update guildSession
	guildSession.VoiceConnection = &voiceConnection
	guildSession.VoiceChannelID = channel.ID
	guildSession.MusicPlayer.PlaybackChannel = make(chan []byte)

	m.Message.Reply(
		context.Background(),
		s,
		fmt.Sprintf("%s joined '%s'", m.Message.Author.Mention(), channel.Name),
	)
	log.Printf(fmt.Sprintf("%s joined '%s' guild '%s'\n", m.Message.Author.Mention(), channel.Name, m.Message.GuildID))

	//url, _ := googletts.GetTTSURL("Ready", "en")
	if os.Getenv("BOT_UPDATE_YTDL") == "true" {
		m.Message.Reply(
			context.Background(),
			s,
			fmt.Sprintf("%s updating youtube-dl binary, give me some time.", m.Message.Author.Mention()),
		)

		// Update youtube-dl
		ytdlCmd := exec.Command("/usr/bin/curl", "-L", "https://yt-dl.org/downloads/latest/youtube-dl", "-o", "/usr/local/bin/youtube-dl")
		ytdlCmd.Stdout = os.Stdout
		ytdlCmd.Stderr = os.Stderr
		ytdlCmd.Run()

		m.Message.Reply(
			context.Background(),
			s,
			fmt.Sprintf("%s done!", m.Message.Author.Mention()),
		)
	}

	// Clear youtube-dl cache
	ytdlCmd := exec.Command("youtube-dl", "--rm-cache-dir")
	ytdlCmd.Stdout = os.Stdout
	ytdlCmd.Stderr = os.Stderr
	ytdlCmd.Run()

	//guildSession.MusicPlayer.PlayURL(url)
}
