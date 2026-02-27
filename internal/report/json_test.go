package report

import (
	"io"
	"os"
	"testing"

	"github.com/alesr/impact/internal/estimate"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPrintJSON(t *testing.T) {
	rep := estimate.Report{Rows: []estimate.Row{{Address: "a", KgCO2eMonth: 0.12}}, Totals: estimate.Totals{KgCO2eMonth: 0.12}}

	output := captureStdout(t, func() {
		require.NoError(t, PrintJSON(rep))
	})

	assert.Contains(t, output, "\"rows\"")
	assert.Contains(t, output, "\"address\": \"a\"")
	assert.Contains(t, output, "\"kgco2e_month\": 0.12")
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()

	original := os.Stdout
	r, w, err := os.Pipe()
	require.NoError(t, err)
	os.Stdout = w

	fn()

	require.NoError(t, w.Close())
	os.Stdout = original

	b, err := io.ReadAll(r)
	require.NoError(t, err)
	require.NoError(t, r.Close())

	return string(b)
}
