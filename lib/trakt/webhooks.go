package trakt

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/url"
	"strings"
	"time"

	"github.com/viscerous/goplaxt/lib/store"
	"github.com/xanderstrike/plexhooks"
)

// PlexFullPayload is used to manual extract fields missing in plexhooks.PlexResponse
type PlexFullPayload struct {
	ViewOffset int64 `json:"viewOffset"`
	Metadata   struct {
		Duration   int64       `json:"duration"`
		UserRating float64     `json:"userRating"`
		RawRating  interface{} `json:"rating"` // Could be float or array
	} `json:"Metadata"`
}

// ScrobbleAppMetadata contains version info for Trakt
type ScrobbleAppMetadata struct {
	AppVersion string `json:"app_version"`
	AppDate    string `json:"app_date"`
}

// Handle determines if an item is a show or a movie and routes appropriately
func Handle(ctx context.Context, client Client, pr plexhooks.PlexResponse, body []byte, user store.User) {
	var full PlexFullPayload
	if err := json.Unmarshal(body, &full); err != nil {
		slog.Warn("Error unmarshalling full payload", "error", err)
	}

	slog.Debug("Webhook payload details",
		"event", pr.Event,
		"offset", full.ViewOffset,
		"duration", full.Metadata.Duration,
		"userRating", full.Metadata.UserRating)

	var err error
	switch pr.Event {
	case "media.rate":
		err = handleRate(ctx, client, pr, full, user)
	case "library.new":
		err = handleCollection(ctx, client, pr, body, user)
	case "media.play", "media.pause", "media.resume", "media.stop", "media.scrobble":
		switch pr.Metadata.LibrarySectionType {
		case "show":
			err = handleShow(ctx, client, pr, full, user)
		case "movie":
			err = handleMovie(ctx, client, pr, full, user)
		}
	default:
		slog.Debug("Event not handled", "event", pr.Event)
	}

	if err != nil {
		slog.Error("Failed to handle event", "event", pr.Event, "error", err)
	}
}

// handleShow starts the scrobbling for a show
func handleShow(ctx context.Context, client Client, pr plexhooks.PlexResponse, full PlexFullPayload, user store.User) error {
	finder := func() (interface{}, string, error) {
		ep, err := findEpisode(ctx, client, pr)
		return ep, ep.Title, err
	}
	builder := func(p float64, i interface{}) interface{} {
		return map[string]interface{}{
			"progress":    p,
			"episode":     i,
			"app_version": "1.0.0",
			"app_date":    time.Now().Format("2006-01-02"),
		}
	}
	return handleScrobble(ctx, client, pr, full, user, user.Config.GetEpisodeScrobbleStart(), user.Config.GetEpisodeScrobbleStop(), finder, builder)
}

// handleMovie starts the scrobbling for a movie
func handleMovie(ctx context.Context, client Client, pr plexhooks.PlexResponse, full PlexFullPayload, user store.User) error {
	finder := func() (interface{}, string, error) {
		m, err := findMovie(ctx, client, pr)
		return m, m.Title, err
	}
	builder := func(p float64, i interface{}) interface{} {
		return map[string]interface{}{
			"progress":    p,
			"movie":       i,
			"app_version": "1.0.0",
			"app_date":    time.Now().Format("2006-01-02"),
		}
	}
	return handleScrobble(ctx, client, pr, full, user, user.Config.GetMovieScrobbleStart(), user.Config.GetMovieScrobbleStop(), finder, builder)
}

func handleScrobble(ctx context.Context, client Client, pr plexhooks.PlexResponse, full PlexFullPayload, user store.User,
	startEnabled, stopEnabled bool,
	findItem func() (interface{}, string, error),
	buildBody func(float64, interface{}) interface{}) error {

	event, progress := getAction(pr, full)
	if event == "" {
		return nil
	}

	// Trakt 422 Error Prevention
	if (event == "pause" || event == "stop") && progress < 1.0 {
		slog.Info("Progress too low for scrobble, clearing status instead", "event", event, "progress", progress)
		_ = client.DeleteCheckin(ctx, user.AccessToken)
		return nil
	}

	if event == "start" && !startEnabled {
		slog.Debug("Start Scrobble disabled by user", "type", pr.Metadata.LibrarySectionType)
		return nil
	}
	if event == "stop" && !stopEnabled {
		slog.Debug("Stop Scrobble disabled by user", "type", pr.Metadata.LibrarySectionType)
		return nil
	}

	item, title, err := findItem()
	if err != nil {
		return fmt.Errorf("failed to find item: %w", err)
	}

	body := buildBody(progress, item)
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("failed to marshal scrobble object: %w", err)
	}

	_, err = client.ScrobbleRequest(ctx, event, jsonBody, user.AccessToken)
	if err == nil {
		slog.Info("Scrobble successful", "action", event, "item", title, "progress", progress)
	}
	return err
}

func handleRate(ctx context.Context, client Client, pr plexhooks.PlexResponse, full PlexFullPayload, user store.User) error {
	rating := full.Metadata.UserRating
	if rating == 0 {
		// Fallback to 'rating' field if userRating is 0 (handles both float and array formats)
		switch val := full.Metadata.RawRating.(type) {
		case float64:
			rating = val
		case []interface{}:
			if len(val) > 0 {
				if r, ok := val[0].(float64); ok {
					rating = r
				}
			}
		}
	}

	intRating := int(rating)
	if intRating < 1 && rating > 0 {
		intRating = 1
	}

	slog.Debug("Handling rate event", "rating", intRating, "user_rating", full.Metadata.UserRating)

	if intRating == 0 {
		return handleRatingRemove(ctx, client, pr, user)
	}

	rateBody := RateBody{}
	switch pr.Metadata.LibrarySectionType {
	case "movie":
		if !user.Config.GetMovieRate() {
			slog.Debug("Movie Rating Sync disabled by user")
			return nil
		}

		movie, err := findMovie(ctx, client, pr)
		if err != nil {
			return fmt.Errorf("failed to find movie: %w", err)
		}
		rateBody.Movies = []MovieRating{{
			Rating: intRating,
			Title:  movie.Title,
			Year:   movie.Year,
			Ids:    movie.Ids,
		}}
	case "show":
		switch pr.Metadata.Type {
		case "episode":
			if !user.Config.GetEpisodeRate() {
				slog.Debug("Episode Rating Sync disabled by user")
				return nil
			}
			episode, err := findEpisode(ctx, client, pr)
			if err != nil {
				return fmt.Errorf("failed to find episode: %w", err)
			}
			rateBody.Episodes = []EpisodeRating{{
				Rating:  intRating,
				Episode: episode,
			}}
		case "show":
			if !user.Config.GetShowRate() {
				slog.Debug("Show Rating Sync disabled by user")
				return nil
			}
			slog.Debug("Rating shows directly not fully supported yet")
			return nil
		}
	}

	jsonBody, err := json.Marshal(rateBody)
	if err != nil {
		return fmt.Errorf("failed to marshal rate body: %w", err)
	}

	_, err = client.SyncRequest(ctx, "ratings", jsonBody, user.AccessToken)
	if err == nil {
		slog.Info("Rating synced successfully", "rating", intRating)
	}
	return err
}

func handleRatingRemove(ctx context.Context, client Client, pr plexhooks.PlexResponse, user store.User) error {
	slog.Debug("Handling rating removal event")

	removeBody := CollectionBody{}
	switch pr.Metadata.LibrarySectionType {
	case "movie":
		if !user.Config.GetMovieRate() {
			return nil
		}
		movie, err := findMovie(ctx, client, pr)
		if err != nil {
			return fmt.Errorf("failed to find movie: %w", err)
		}
		removeBody.Movies = []Movie{movie}
	case "show":
		if pr.Metadata.Type == "episode" {
			if !user.Config.GetEpisodeRate() {
				return nil
			}
			episode, err := findEpisode(ctx, client, pr)
			if err != nil {
				return fmt.Errorf("failed to find episode: %w", err)
			}
			removeBody.Episodes = []Episode{episode}
		} else {
			slog.Debug("Rating removal for show directly not supported yet")
			return nil
		}
	}

	jsonBody, err := json.Marshal(removeBody)
	if err != nil {
		return fmt.Errorf("failed to marshal remove body: %w", err)
	}

	_, err = client.SyncRequest(ctx, "ratings/remove", jsonBody, user.AccessToken)
	if err == nil {
		slog.Info("Rating removal synced successfully")
	}
	return err
}

func handleCollection(ctx context.Context, client Client, pr plexhooks.PlexResponse, body []byte, user store.User) error {
	slog.Debug("Handling collection add event")

	mediaMeta := extractMediaInfo(body)

	collectionBody := CollectionBody{}
	switch pr.Metadata.LibrarySectionType {
	case "movie":
		if !user.Config.GetMovieCollection() {
			slog.Debug("Movie Collection Sync disabled by user")
			return nil
		}
		movie, err := findMovie(ctx, client, pr)
		if err != nil {
			return fmt.Errorf("failed to find movie: %w", err)
		}
		movie.MediaMetadata = mediaMeta
		collectionBody.Movies = []Movie{movie}
	case "show":
		if pr.Metadata.Type == "episode" {
			if !user.Config.GetEpisodeCollection() {
				slog.Debug("Episode Collection Sync disabled by user")
				return nil
			}
			episode, err := findEpisode(ctx, client, pr)
			if err != nil {
				return fmt.Errorf("failed to find episode: %w", err)
			}
			episode.MediaMetadata = mediaMeta
			collectionBody.Episodes = []Episode{episode}
		} else {
			slog.Debug("Collection add for show directly not supported yet")
			return nil
		}
	}

	jsonBody, err := json.Marshal(collectionBody)
	if err != nil {
		return fmt.Errorf("failed to marshal collection body: %w", err)
	}

	_, err = client.SyncRequest(ctx, "collection", jsonBody, user.AccessToken)
	if err == nil {
		slog.Info("Collection added successfully")
	}
	return err
}

func findEpisode(ctx context.Context, client Client, pr plexhooks.PlexResponse) (Episode, error) {
	// Try GUID search
	var episode Episode
	found, err := searchByGuids(ctx, client, pr.Metadata.ExternalGuid, "episode", func(body []byte) bool {
		var showInfo []ShowInfo
		if err := json.Unmarshal(body, &showInfo); err != nil {
			return false
		}
		if len(showInfo) > 0 {
			episode = showInfo[0].Episode
			slog.Info("Tracking episode", "show", showInfo[0].Show.Title, "season", episode.Season, "number", episode.Number)
			return true
		}
		return false
	})
	if err != nil {
		slog.Debug("GUID search error", "error", err)
	}
	if found {
		return episode, nil
	}

	// Fallback with title/year
	slog.Debug("Finding episode by title", "title", pr.Metadata.GrandparentTitle, "year", pr.Metadata.Year)
	apiUrl := fmt.Sprintf("%s/search/show?query=%s", BaseURL, url.PathEscape(pr.Metadata.GrandparentTitle))

	respBody, err := client.MakeRequest(ctx, apiUrl)
	if err != nil {
		return Episode{}, fmt.Errorf("failed to search show: %w", err)
	}

	var results []ShowSearchResult
	if err := json.Unmarshal(respBody, &results); err != nil {
		return Episode{}, fmt.Errorf("failed to unmarshal show search results: %w", err)
	}

	var show *Show

	for _, result := range results {
		if pr.Metadata.Year == 0 || result.Show.Year == pr.Metadata.Year {
			show = &result.Show
			break
		}
	}

	if show != nil {
		apiUrl = fmt.Sprintf("%s/shows/%d/seasons?extended=episodes", BaseURL, show.Ids.Trakt)

		respBody, err = client.MakeRequest(ctx, apiUrl)
		if err != nil {
			return Episode{}, fmt.Errorf("failed to get seasons: %w", err)
		}
		var seasons []Season
		if err := json.Unmarshal(respBody, &seasons); err != nil {
			return Episode{}, fmt.Errorf("failed to unmarshal seasons: %w", err)
		}

		for _, season := range seasons {
			if season.Number == pr.Metadata.ParentIndex {
				for _, episode := range season.Episodes {
					if episode.Number == pr.Metadata.Index {
						slog.Info("Tracking episode via title search", "show", show.Title, "season", season.Number, "number", episode.Number)

						return episode, nil
					}
				}
			}
		}
	}

	return Episode{}, fmt.Errorf("could not find episode")
}

func findMovie(ctx context.Context, client Client, pr plexhooks.PlexResponse) (Movie, error) {
	// Try GUID search
	var movie Movie
	found, err := searchByGuids(ctx, client, pr.Metadata.ExternalGuid, "movie", func(body []byte) bool {
		var movies []MovieSearchResult
		if err := json.Unmarshal(body, &movies); err != nil {
			return false
		}
		if len(movies) > 0 {
			movie = movies[0].Movie
			slog.Info("Tracking movie", "title", movie.Title)
			return true
		}
		return false
	})
	if err != nil {
		slog.Debug("GUID search error", "error", err)
	}
	if found {
		return movie, nil
	}

	// Fallback with title/year
	slog.Debug("Finding movie by title", "title", pr.Metadata.Title, "year", pr.Metadata.Year)
	apiUrl := fmt.Sprintf("%s/search/movie?query=%s", BaseURL, url.PathEscape(pr.Metadata.Title))

	respBody, err := client.MakeRequest(ctx, apiUrl)
	if err != nil {
		return Movie{}, fmt.Errorf("failed to search movie: %w", err)
	}

	var results []MovieSearchResult
	if err := json.Unmarshal(respBody, &results); err != nil {
		return Movie{}, fmt.Errorf("failed to unmarshal movie search results: %w", err)
	}

	for _, result := range results {
		if pr.Metadata.Year == 0 || result.Movie.Year == pr.Metadata.Year {
			slog.Info("Tracking movie via title search", "title", result.Movie.Title)
			return result.Movie, nil
		}
	}

	return Movie{}, fmt.Errorf("could not find movie")
}

func searchByGuids(ctx context.Context, client Client, guids []plexhooks.ExternalGuid, typeStr string, parser func([]byte) bool) (bool, error) {
	for _, guid := range guids {
		index := strings.Index(guid.Id, "://")
		if index == -1 {
			continue
		}
		traktService := guid.Id[:index]
		id := guid.Id[index+3:]

		slog.Debug("Finding item by guid", "id", id, "service", traktService, "type", typeStr)
		apiUrl := fmt.Sprintf("%s/search/%s/%s?type=%s", BaseURL, traktService, id, typeStr)

		respBody, err := client.MakeRequest(ctx, apiUrl)
		if err != nil {
			slog.Debug("Error searching by guid", "error", err)
			continue
		}

		if parser(respBody) {
			return true, nil
		}
	}
	return false, nil
}

func getAction(pr plexhooks.PlexResponse, full PlexFullPayload) (string, float64) {
	progress := 0.0
	if full.Metadata.Duration > 0 {
		progress = (float64(full.ViewOffset) / float64(full.Metadata.Duration)) * 100.0
	}

	if progress > 100.0 {
		progress = 100.0
	}

	switch pr.Event {
	case "media.play", "media.resume":
		return "start", progress
	case "media.pause":
		return "pause", progress
	case "media.stop":
		if progress >= 90.0 {
			return "stop", progress
		}
		return "pause", progress // Trakt 422 if we 'stop' too early? Pause is safer.
	case "media.scrobble":
		return "stop", 90.0
	}
	return "", 0.0
}
