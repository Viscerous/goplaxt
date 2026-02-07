package trakt

import (
	"encoding/json"
	"log/slog"
)

// PlexMediaContainer is used to extract addedAt timestamp from Plex payload
type PlexMediaContainer struct {
	Metadata struct {
		AddedAt int64 `json:"addedAt"`
	} `json:"Metadata"`
}

// extractCollectedAt extracts and formats the addedAt timestamp from Plex payload as ISO 8601 for Trakt
func extractCollectedAt(body []byte) string {
	var container PlexMediaContainer
	if err := json.Unmarshal(body, &container); err != nil {
		slog.Warn("Error parsing addedAt time", "error", err)
		return ""
	}
	return formatCollectedAt(container.Metadata.AddedAt)
}
