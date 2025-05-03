package trakt

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/viscerous/goplaxt/lib/config"
	"github.com/viscerous/goplaxt/lib/store"
	"github.com/xanderstrike/plexhooks"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"
)

type Client interface {
	MakeRequest(url string) []byte
	ScrobbleRequest(action string, body []byte, token string) []byte
}

type RealTraktClient struct{}

func (c *RealTraktClient) MakeRequest(url string) []byte {
	return makeRequest(url)
}

func (c *RealTraktClient) ScrobbleRequest(action string, body []byte, token string) []byte {
	return scrobbleRequest(action, body, token)
}

// AuthRequest authorize the connection with Trakt
func AuthRequest(root, username, code, refreshToken, grantType string) (map[string]interface{}, error) {
	values := map[string]string{
		"code":          code,
		"refresh_token": refreshToken,
		"client_id":     config.TraktClientId,
		"client_secret": config.TraktClientSecret,
		"redirect_uri":  fmt.Sprintf("%s/authorize?username=%s", root, url.PathEscape(username)),
		"grant_type":    grantType,
	}
	jsonValue, err := json.Marshal(values)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request body: %w", err)
	}

	resp, err := http.Post("https://api.trakt.tv/oauth/token", "application/json", bytes.NewBuffer(jsonValue))
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return result, nil
}

// Handle determine if an item is a show or a movie
func Handle(client Client, pr plexhooks.PlexResponse, user store.User) {
	if pr.Metadata.LibrarySectionType == "show" {
		handleShow(client, pr, user.AccessToken)
	} else if pr.Metadata.LibrarySectionType == "movie" {
		handleMovie(client, pr, user.AccessToken)
	}
	log.Print("Event logged")
}

// handleShow start the scrobbling for a show
func handleShow(client Client, pr plexhooks.PlexResponse, accessToken string) {
	event, progress := getAction(pr)

	scrobbleObject := ShowScrobbleBody{
		Progress: progress,
		Episode:  findEpisode(client, pr),
	}

	scrobbleJSON, err := json.Marshal(scrobbleObject)
	handleErr(err)

	client.ScrobbleRequest(event, scrobbleJSON, accessToken)
}

// handleMovie start the scrobbling for a movie
func handleMovie(client Client, pr plexhooks.PlexResponse, accessToken string) {
	event, progress := getAction(pr)

	scrobbleObject := MovieScrobbleBody{
		Progress: progress,
		Movie:    findMovie(client, pr),
	}

	scrobbleJSON, _ := json.Marshal(scrobbleObject)

	client.ScrobbleRequest(event, scrobbleJSON, accessToken)
}

func findEpisode(client Client, pr plexhooks.PlexResponse) Episode {
	guids := pr.Metadata.ExternalGuid

	// Try to find a match with Guid
	for _, guid := range guids {
		index := strings.Index(guid.Id, "://")

		if index == -1 {
			continue
		}

		traktService := guid.Id[:index]
		episodeID := guid.Id[index+3:]

		log.Printf("Finding episode with episode ID %s using %s", episodeID, traktService)

		apiUrl := fmt.Sprintf("https://api.trakt.tv/search/%s/%s?type=episode", traktService, episodeID)

		respBody := client.MakeRequest(apiUrl)

		var showInfo []ShowInfo
		err := json.Unmarshal(respBody, &showInfo)
		handleErr(err)

		if len(showInfo) > 0 {
			episode := showInfo[0].Episode
			log.Printf("Tracking %s - S%02dE%02d - %s using %s", showInfo[0].Show.Title, episode.Season, episode.Number, episode.Title, traktService)

			return episode
		}
	}

	// Fallback with title/year
	log.Printf("Finding episode with title %s (%d)", pr.Metadata.GrandparentTitle, pr.Metadata.Year)
	apiUrl := fmt.Sprintf("https://api.trakt.tv/search/show?query=%s", url.PathEscape(pr.Metadata.GrandparentTitle))

	respBody := client.MakeRequest(apiUrl)

	var results []ShowSearchResult
	err := json.Unmarshal(respBody, &results)
	handleErr(err)

	var show *Show

	for _, result := range results {
		if result.Show.Year == pr.Metadata.Year {
			show = &result.Show
			break
		}
	}

	if show != nil {
		apiUrl = fmt.Sprintf("https://api.trakt.tv/shows/%d/seasons?extended=episodes", show.Ids.Trakt)

		respBody = client.MakeRequest(apiUrl)
		var seasons []Season
		err = json.Unmarshal(respBody, &seasons)
		handleErr(err)

		for _, season := range seasons {
			if season.Number == pr.Metadata.ParentIndex {
				for _, episode := range season.Episodes {
					if episode.Number == pr.Metadata.Index {
						log.Printf("Tracking %s - S%02dE%02d - %s using title search", show.Title, season.Number, episode.Number, episode.Title)

						return episode
					}
				}
			}
		}
	}

	panic("Could not find episode!")
}

func findMovie(client Client, pr plexhooks.PlexResponse) Movie {
	guids := pr.Metadata.ExternalGuid

	// Try to find a match with Guid
	for _, guid := range guids {
		index := strings.Index(guid.Id, "://")

		if index == -1 {
			continue
		}

		traktService := guid.Id[:index]
		movieId := guid.Id[index+3:]

		log.Printf("Finding movie ID %s using %s", movieId, traktService)

		apiUrl := fmt.Sprintf("https://api.trakt.tv/search/%s/%s?type=movie", traktService, movieId)

		respBody := client.MakeRequest(apiUrl)

		var movies []MovieSearchResult
		err := json.Unmarshal(respBody, &movies)
		handleErr(err)

		if len(movies) > 0 {
			log.Printf("Tracking %s - using %s", movies[0].Movie.Title, traktService)

			return movies[0].Movie
		}
	}

	// Fallback with title/year
	log.Printf("Finding movie with title %s (%d)", pr.Metadata.Title, pr.Metadata.Year)
	apiUrl := fmt.Sprintf("https://api.trakt.tv/search/movie?query=%s", url.PathEscape(pr.Metadata.Title))

	respBody := client.MakeRequest(apiUrl)

	var results []MovieSearchResult
	err := json.Unmarshal(respBody, &results)
	handleErr(err)

	for _, result := range results {
		if result.Movie.Year == pr.Metadata.Year {
			log.Printf("Tracking %s - using title search", result.Movie.Title)
			return result.Movie
		}
	}

	panic("Could not find movie!")
}

func makeRequest(url string) []byte {
	client := &http.Client{}

	req, err := http.NewRequest("GET", url, nil)
	handleErr(err)

	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("trakt-api-version", "2")
	req.Header.Add("trakt-api-key", config.TraktClientId)

	resp, err := client.Do(req)
	handleErr(err)
	defer resp.Body.Close()

	respBody, _ := ioutil.ReadAll(resp.Body)

	return respBody
}

func scrobbleRequest(action string, body []byte, accessToken string) []byte {
	client := &http.Client{}

	apiUrl := fmt.Sprintf("https://api.trakt.tv/scrobble/%s", action)

	req, err := http.NewRequest("POST", apiUrl, bytes.NewBuffer(body))
	handleErr(err)

	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", accessToken))
	req.Header.Add("trakt-api-version", "2")
	req.Header.Add("trakt-api-key", config.TraktClientId)

	resp, err := client.Do(req)
	if err != nil {
		handleErr(err)
	}
	defer resp.Body.Close()

	respBody, _ := ioutil.ReadAll(resp.Body)

	return respBody
}

func getAction(pr plexhooks.PlexResponse) (string, int) {
	switch pr.Event {
	case "media.play":
		return "start", 0
	case "media.pause":
		return "stop", 0
	case "media.resume":
		return "start", 0
	case "media.stop":
		return "stop", 0
	case "media.scrobble":
		return "stop", 90
	}
	return "", 0
}

func handleErr(err error) {
	if err != nil {
		panic(err)
	}
}
