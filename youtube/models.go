package youtube

type Playlist struct {
	Type    string         `json:"_type"`
	Entries []PlaylistItem `json:"entries"`
}

type PlaylistItem struct {
	Type     string  `json:"_type"`
	IeKey    string  `json:"ie_key"`
	Id       string  `json:"id"`
	Url      string  `json:"url"`
	Title    string  `json:"title"`
	Duration float64 `json:"duration"`
	Uploader string  `json:"uploader"`
}
