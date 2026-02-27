package tui

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTruncate(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "short", truncate("short", 10))
	assert.Equal(t, "very-lo...", truncate("very-long-resource-address", 10))
}
