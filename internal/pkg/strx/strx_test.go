package strx

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseCSV(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input []string
		want  []string
	}{
		{
			name:  "trims and drops empty entries",
			input: []string{"fr-par, nl-ams, , pl-waw"},
			want:  []string{"fr-par", "nl-ams", "pl-waw"},
		},
		{
			name:  "returns nil when empty",
			input: []string{" "},
			want:  nil,
		},
		{
			name:  "merges multiple values",
			input: []string{"fr-par", "nl-ams,pl-waw"},
			want:  []string{"fr-par", "nl-ams", "pl-waw"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := ParseCSV(tt.input...)
			assert.Equal(t, tt.want, got)
		})
	}
}
