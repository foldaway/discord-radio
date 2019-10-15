package models

import (
	youtube "google.golang.org/api/youtube/v3"
)

type QueueItem struct {
	Title        string
	ChannelTitle string
	Author       string
	VideoID      string
	Thumbnail    string
}

// ConvertYouTubePlaylistItem convert a YouTube playlist item into a local QueueItem model
func ConvertYouTubePlaylistItem(playlistItem *youtube.PlaylistItem) QueueItem {
	return QueueItem{
		Title:        playlistItem.Snippet.Title,
		ChannelTitle: playlistItem.Snippet.ChannelTitle,
		Author:       "AutoPlaylist",
		VideoID:      playlistItem.ContentDetails.VideoId,
		Thumbnail:    playlistItem.Snippet.Thumbnails.Default.Url,
	}
}
