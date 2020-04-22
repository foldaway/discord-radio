package commands

import (
	"fmt"
	"log"
	"os"
	"os/exec"

	"github.com/andersfylling/disgord"
	"github.com/bottleneckco/discord-radio/ctx"
	"github.com/bottleneckco/discord-radio/vscache"
	"github.com/evalphobia/google-tts-go/googletts"
)

func join(s disgord.Session, m *disgord.MessageCreate) {
	voiceChannelInit(s, m)
	guildSession := safeGetGuildSession(m.Message.GuildID)
	go guildSession.Loop()
}

func voiceChannelInit(s disgord.Session, m *disgord.MessageCreate) {
	guildSession := safeGetGuildSession(m.Message.GuildID)
	voiceStateCache, ok := vscache.FindUserVoiceState(m.Message.Author.ID)
	if !ok {
		log.Println("No voice state cached")
		s.SendMsg(ctx.Ctx, m.Message.ChannelID, fmt.Sprintf("%s you are not in a voice channel", m.Message.Author.Mention()))
		return
	}
	channel, err := s.GetChannel(ctx.Ctx, voiceStateCache.Current.ChannelID)
	if err != nil {
		log.Println(err)
		s.SendMsg(ctx.Ctx, m.Message.ChannelID, fmt.Sprintf("%s error occurred: %s", m.Message.Author.Mention(), err))
		return
	}

	voiceChannel, err := s.VoiceConnect(m.Message.GuildID, voiceStateCache.Current.ChannelID)

	if err != nil {
		log.Println(err)
		s.SendMsg(ctx.Ctx, m.Message.ChannelID, fmt.Sprintf("%s error occurred: %s", m.Message.Author.Mention(), err))
		return
	}
	guildSession.VoiceConnection = voiceChannel
	guildSession.VoiceChannelID = channel.ID
	s.SendMsg(ctx.Ctx, m.Message.ChannelID, fmt.Sprintf("%s joined '%s'", m.Message.Author.Mention(), channel.Name))
	log.Printf(fmt.Sprintf("%s joined '%s' guild '%s'\n", m.Message.Author.Mention(), channel.Name, m.Message.GuildID))
	url, _ := googletts.GetTTSURL("Ready", "en")
	if os.Getenv("BOT_UPDATE_YTDL") == "true" {
		s.SendMsg(ctx.Ctx, m.Message.ChannelID, fmt.Sprintf("%s updating youtube-dl binary, give me some time.", m.Message.Author.Mention()))
		updateCmd := exec.Command("/usr/bin/curl", "-L", "https://yt-dl.org/downloads/latest/youtube-dl", "-o", "/usr/local/bin/youtube-dl")
		updateCmd.Stdout = os.Stdout
		updateCmd.Stderr = os.Stderr
		updateCmd.Run()
		s.SendMsg(ctx.Ctx, m.Message.ChannelID, fmt.Sprintf("%s done!", m.Message.Author.Mention()))
	}
	guildSession.PlayURL(url, 0.5)
}
