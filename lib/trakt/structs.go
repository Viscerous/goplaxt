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
	Ids   Ids
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
}

// Season represents a season
type Season struct {
	Number   int
	Episodes []Episode
}

// Movie represents a movie
type Movie struct {
	Title string `json:"title"`
	Year  int    `json:"year"`
	Ids   Ids    `json:"ids"`
}

// MovieSearchResult represents a search result for a movie
type MovieSearchResult struct {
	Movie Movie
}

// ShowScrobbleBody represents the scrobbling status for a show
type ShowScrobbleBody struct {
	Episode  Episode `json:"episode"`
	Progress int     `json:"progress"`
}

// MovieScrobbleBody represents the scrobbling status for a movie
type MovieScrobbleBody struct {
	Movie    Movie `json:"movie"`
	Progress int   `json:"progress"`
}
