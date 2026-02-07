package trakt

import (
	"encoding/json"
	"log/slog"
	"strings"
)

type PlexMediaContainer struct {
	Metadata struct {
		Media []struct {
			VideoResolution string `json:"videoResolution"`
			AudioCodec      string `json:"audioCodec"`
			AudioChannels   int    `json:"audioChannels"`
			Container       string `json:"container"`
			Part            []struct {
				Stream []struct {
					StreamType           int    `json:"streamType"`
					Codec                string `json:"codec"`
					DisplayTitle         string `json:"displayTitle"`
					ExtendedDisplayTitle string `json:"extendedDisplayTitle"`
					ColorTrc             string `json:"colorTrc"` // for HDR detection (smpte2084, arib-std-b67)
				} `json:"Stream"`
			} `json:"Part"`
		} `json:"Media"`
	} `json:"Metadata"`
}

func extractMediaInfo(body []byte) MediaMetadata {
	var container PlexMediaContainer
	if err := json.Unmarshal(body, &container); err != nil {
		slog.Warn("Error parsing media info", "error", err)
		return MediaMetadata{}
	}

	if len(container.Metadata.Media) == 0 {
		return MediaMetadata{}
	}

	media := container.Metadata.Media[0]
	meta := MediaMetadata{}

	// Resolution
	switch media.VideoResolution {
	case "4k", "2160":
		meta.Resolution = "uhd_4k"
	case "1080":
		meta.Resolution = "hd_1080p"
	case "720":
		meta.Resolution = "hd_720p"
	case "480":
		meta.Resolution = "sd_480p"
	case "576":
		meta.Resolution = "sd_576p"
	}

	// Audio
	// Plex often gives generic codec names, map best effort
	switch strings.ToLower(media.AudioCodec) {
	case "aac":
		meta.Audio = "aac"
	case "ac3":
		meta.Audio = "dolby_digital"
	case "eac3":
		meta.Audio = "dolby_digital_plus"
	case "dts":
		meta.Audio = "dts"
	case "dca":
		meta.Audio = "dts"
	case "truehd":
		meta.Audio = "dolby_truehd"
	case "flac":
		meta.Audio = "flac"
	case "mp3":
		meta.Audio = "mp3"
	case "pcm":
		meta.Audio = "lpcm"
	case "opus":
		meta.Audio = "ogg_opus"
	case "vorbis":
		meta.Audio = "ogg"
	}

	// Audio Channels
	switch media.AudioChannels {
	case 1:
		meta.AudioChannels = "1.0"
	case 2:
		meta.AudioChannels = "2.0"
	case 6: // 5.1
		meta.AudioChannels = "5.1"
	case 8: // 7.1
		meta.AudioChannels = "7.1"
	}

	// Check for Atmos or DTS:X via extended titles loop if needed
	// And HDR
	for _, part := range media.Part {
		for _, stream := range part.Stream {
			if stream.StreamType == 1 { // Video
				switch stream.ColorTrc {
				case "smpte2084":
					meta.HDR = "hdr10" // Default assumption
					// Could check further for Dolby Vision (dovi)
				case "arib-std-b67":
					meta.HDR = "hlg"
				}
			}
		}
	}

	// Media Type (Physical vs Digital)
	// Rough guess based on container/bitrate/filename?
	// For now default to digital unless we find signs of rip?
	meta.MediaType = "digital"

	return meta
}
