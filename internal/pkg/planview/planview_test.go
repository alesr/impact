package planview

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFormatKg(t *testing.T) {
	t.Parallel()

	t.Run("returns formatted value when known", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, "0.123456", FormatKg(0.123456, true))
	})

	t.Run("returns N/A when unknown", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, "N/A", FormatKg(0, false))
	})
}

func TestFormatWater(t *testing.T) {
	t.Parallel()

	t.Run("returns formatted value when known", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, "0.654321", FormatWater(0.654321, true))
	})

	t.Run("returns N/A when unknown", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, "N/A", FormatWater(0, false))
	})
}

func TestUnknownImpactNote(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "", UnknownImpactNote(0))
	assert.Equal(t, "partial totals: 2 row(s) have unknown footprint data", UnknownImpactNote(2))
}
