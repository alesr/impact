package report

import (
	"testing"

	"github.com/alesr/impact/internal/estimate"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPrintTable(t *testing.T) {
	t.Parallel()

	rep := estimate.Report{
		Rows: []estimate.Row{{
			Address:      "scaleway_instance_server.web",
			Action:       "create",
			KgCO2eMonth:  0.1,
			KgCO2eKnown:  true,
			M3WaterMonth: 0.01,
			M3WaterKnown: true,
			SKU:          "sku-id",
		}},
		Unsupported: []estimate.UnsupportedResource{{Address: "scaleway_x.y", Reason: "not implemented"}},
		Totals:      estimate.Totals{KgCO2eMonth: 0.1, KgCO2eKnown: true, M3WaterMonth: 0.01, M3WaterKnown: true},
	}

	output := captureStdout(t, func() {
		require.NoError(t, PrintTable(rep))
	})

	assert.Contains(t, output, "Totals")
	assert.Contains(t, output, "ADDRESS")
	assert.Contains(t, output, "scaleway_instance_server.web")
	assert.Contains(t, output, "Unsupported resources (1)")
}
