package main

import (
	"bytes"
	"flag"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/textproto"
)

func main() {
	event := flag.String("event", "media.play", "Event type: media.play, media.pause, media.rate, library.new")
	media := flag.String("media", "movie", "Media type: movie, show, episode")
	rating := flag.Int("rating", 0, "Rating (1-10) for media.rate event")
	user := flag.String("user", "test-user-id", "User ID query param")
	url := flag.String("url", "http://localhost:8000/api", "Target URL")
	title := flag.String("title", "Test Title", "Media title")
	year := flag.Int("year", 2024, "Media year")
	guid := flag.String("guid", "plex://movie/12345", "PLEX GUID")
	username := flag.String("username", "testuser", "Plex username associated with the account")
	offset := flag.Int64("offset", 0, "View offset in ms")
	duration := flag.Int64("duration", 8700000, "Duration in ms")
	addedAt := flag.Int64("addedAt", 1707494400, "Added at timestamp for library.new events")

	flag.Parse()

	// Minified JSON payload to ensure regex matching works on server
	payload := fmt.Sprintf(`{"event":"%s","viewOffset":%d,"Account":{"title":"%s"},"Metadata":{"librarySectionType":"%s","type":"%s","title":"%s","grandparentTitle":"%s","year":%d,"guid":"%s","rating":%d,"duration":%d,"userRating":%d,"addedAt":%d,"ExternalGuid":[{"id":"imdb://tt1234567"},{"id":"tmdb://12345"}]}}`,
		*event, *offset, *username, *media, *media, *title, *title, *year, *guid, *rating, *duration, *rating, *addedAt)

	if *media == "episode" {
		payload = fmt.Sprintf(`{"event":"%s","viewOffset":%d,"Account":{"title":"%s"},"Metadata":{"librarySectionType":"show","type":"episode","title":"Episode Title","grandparentTitle":"%s","parentIndex":1,"index":1,"year":%d,"guid":"%s","rating":%d,"duration":%d,"userRating":%d,"addedAt":%d,"ExternalGuid":[{"id":"imdb://tt1234567"},{"id":"tmdb://12345"}]}}`,
			*event, *offset, *username, *title, *year, *guid, *rating, *duration, *rating, *addedAt)
	}

	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)

	// Create a form part with a custom Content-Type for JSON
	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition", `form-data; name="payload"`)
	h.Set("Content-Type", "application/json")
	part, err := writer.CreatePart(h)
	if err != nil {
		panic(err)
	}
	part.Write([]byte(payload))
	writer.Close()

	fullURL := fmt.Sprintf("%s?id=%s", *url, *user)
	req, err := http.NewRequest("POST", fullURL, body)
	if err != nil {
		panic(err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	fmt.Printf("Sent %s event to %s. Status: %s\n", *event, fullURL, resp.Status)
}
