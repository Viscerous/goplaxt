package trakt

import "time"

// Time format constants for Trakt API
const (
	// DateFormat is the YYYY-MM-DD format used for app_date in scrobble requests
	DateFormat = "2006-01-02"

	// AppVersion is the version string sent with scrobble requests
	AppVersion = "1.0.0"
)

// Duration constants
const (
	// HTTPTimeout is the default timeout for HTTP requests
	HTTPTimeout = 30 * time.Second

	// MaxRetries is the number of retry attempts for failed requests
	MaxRetries = 3
)

// retryBackoff sleeps for an exponentially increasing duration based on attempt number.
// Attempt should be 0-indexed.
func retryBackoff(attempt int) {
	time.Sleep(time.Duration(attempt+1) * time.Second)
}

// formatAppDate returns the current date in YYYY-MM-DD format for Trakt
func formatAppDate() string {
	return time.Now().Format(DateFormat)
}

// formatCollectedAt converts a Unix timestamp to ISO 8601 UTC format for Trakt collection
func formatCollectedAt(unixTimestamp int64) string {
	if unixTimestamp == 0 {
		return ""
	}
	return time.Unix(unixTimestamp, 0).UTC().Format(time.RFC3339)
}
