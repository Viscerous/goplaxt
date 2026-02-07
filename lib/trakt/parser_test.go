package trakt

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExtractCollectedAt(t *testing.T) {
	tests := []struct {
		name     string
		payload  string
		expected string
	}{
		{
			name: "Valid timestamp",
			payload: `{
				"Metadata": {
					"addedAt": 1707494400
				}
			}`,
			expected: "2024-02-09T16:00:00Z",
		},
		{
			name: "Zero timestamp",
			payload: `{
				"Metadata": {
					"addedAt": 0
				}
			}`,
			expected: "",
		},
		{
			name: "Missing addedAt field",
			payload: `{
				"Metadata": {}
			}`,
			expected: "",
		},
		{
			name:     "Invalid JSON",
			payload:  `{invalid}`,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractCollectedAt([]byte(tt.payload))
			assert.Equal(t, tt.expected, result)
		})
	}
}
