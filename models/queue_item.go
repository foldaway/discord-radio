package models

import (
	"github.com/bottleneckco/discord-radio/util"
)

// QueueItem represents an item in the music queue
type QueueItem struct {
	Title        string
	ChannelTitle string
	Author       string
	VideoID      string
	Thumbnail    string
}

// ConvertYouTubePlaylistItem convert a YouTube playlist item into a local QueueItem model
func ConvertYouTubePlaylistItem(playlistItem util.PlaylistItem) QueueItem {
	return QueueItem{
		Title:        playlistItem.Title,
		ChannelTitle: playlistItem.Uploader,
		Author:       "AutoPlaylist",
		VideoID:      playlistItem.Id,
		Thumbnail:    "",
	}
}
