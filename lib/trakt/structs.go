package trakt

// Ids represents the IDs representing a media item across the metadata providers
type Ids struct {
	Trakt  int    `json:"trakt"`
	Tvdb   int    `json:"tvdb"`
	Imdb   string `json:"imdb"`
	Tmdb   int    `json:"tmdb"`
	Tvrage int    `json:"tvrage"`
}

// Show represents a show's IDs
type Show struct {
	Title string `json:"title"`
	Year  int    `json:"year"`
	Ids   Ids    `json:"ids"`
}

// ShowSearchResult represents a search result for a show
type ShowSearchResult struct {
	Show Show
}

// ShowInfo represents a show
type ShowInfo struct {
	Show    Show
	Episode Episode
}

// Episode represents an episode
type Episode struct {
	Season int    `json:"season"`
	Number int    `json:"number"`
	Title  string `json:"title"`
	Ids    Ids    `json:"ids"`
	MediaMetadata
}

// Season represents a season
type Season struct {
	Number   int
	Episodes []Episode
}

// MediaMetadata representing audio/video technical details
type MediaMetadata struct {
	MediaType     string `json:"media_type,omitempty"`
	Resolution    string `json:"resolution,omitempty"`
	Audio         string `json:"audio,omitempty"`
	AudioChannels string `json:"audio_channels,omitempty"`
	HDR           string `json:"hdr,omitempty"`
	ThreeD        bool   `json:"3d,omitempty"`
}

// Movie represents a movie
type Movie struct {
	Title string `json:"title"`
	Year  int    `json:"year"`
	Ids   Ids    `json:"ids"`
	MediaMetadata
}

// MovieSearchResult represents a search result for a movie
type MovieSearchResult struct {
	Movie Movie
}

// ShowScrobbleBody represents the scrobbling status for a show
type ShowScrobbleBody struct {
	Episode  Episode `json:"episode"`
	Progress float64 `json:"progress"`
}

// MovieScrobbleBody represents the scrobbling status for a movie
type MovieScrobbleBody struct {
	Movie    Movie   `json:"movie"`
	Progress float64 `json:"progress"`
}

// RateBody represents the rating payload to Trakt
type RateBody struct {
	Movies   []MovieRating   `json:"movies,omitempty"`
	Shows    []ShowRating    `json:"shows,omitempty"`
	Episodes []EpisodeRating `json:"episodes,omitempty"`
}

type MovieRating struct {
	Rating int    `json:"rating"`
	Title  string `json:"title"`
	Year   int    `json:"year"`
	Ids    Ids    `json:"ids"`
}

type ShowRating struct {
	Rating int    `json:"rating"`
	Title  string `json:"title"`
	Year   int    `json:"year"`
	Ids    Ids    `json:"ids"`
}

type EpisodeRating struct {
	Rating  int     `json:"rating"`
	Episode Episode `json:"episode"`
}

// CollectionBody represents the collection payload to Trakt
type CollectionBody struct {
	Movies   []Movie   `json:"movies,omitempty"`
	Shows    []Show    `json:"shows,omitempty"`
	Episodes []Episode `json:"episodes,omitempty"`
}
