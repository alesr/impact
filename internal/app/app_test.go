package app

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/alesr/impact/internal/plan"
	"github.com/alesr/impact/internal/scw/catalog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRun(t *testing.T) {
	t.Parallel()

	t.Run("returns usage error when no args are provided", func(t *testing.T) {
		t.Parallel()

		err := Run(nil)
		require.Error(t, err)
		assert.ErrorIs(t, err, errUsage)
	})
}

func TestRunPlan(t *testing.T) {
	t.Parallel()

	t.Run("rejects conflicting source flags", func(t *testing.T) {
		t.Parallel()

		err := runPlan(planOptions{planFile: "x.json", fromTerraform: true})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "either --file or --from-terraform")
	})
}

func TestBuildEstimateReport(t *testing.T) {
	t.Parallel()

	t.Run("builds estimate report from fetched products", func(t *testing.T) {
		t.Parallel()

		changes := []plan.ResourceChange{{
			Address: "scaleway_instance_server.web",
			Type:    "scaleway_instance_server",
			Actions: []string{"create"},
			After:   map[string]any{"zone": "fr-par-2", "type": "POP2-HC-2C-4G"},
		}}

		called := false
		lister := &mockCatalogProductLister{
			listAllProductsFunc: func(context.Context) ([]catalog.Product, error) {
				called = true
				return []catalog.Product{{
					SKU:             "/compute/pop2_hc_2c_4g/run_fr-par-2",
					ProductCategory: "instances",
					Locality:        catalog.Locality{Zone: "fr-par-2"},
					UnitOfMeasure:   catalog.UnitOfMeasure{Unit: "hour", Size: 1},
				}}, nil
			},
		}

		rep, err := buildEstimateReport(context.Background(), changes, lister)
		require.NoError(t, err)
		assert.True(t, called)
		assert.Len(t, rep.Rows, 1)
	})

	t.Run("returns wrapped error when catalog fetch fails", func(t *testing.T) {
		t.Parallel()

		changes := []plan.ResourceChange{{Address: "x", Type: "scaleway_instance_server", Actions: []string{"create"}}}
		lister := &mockCatalogProductLister{
			listAllProductsFunc: func(context.Context) ([]catalog.Product, error) {
				return nil, errors.New("boom")
			},
		}

		_, err := buildEstimateReport(context.Background(), changes, lister)
		assert.Error(t, err)
	})
}

func TestParseDate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{name: "rfc3339", input: "2025-02-01T12:00:00Z", want: "2025-02-01T12:00:00Z"},
		{name: "yyyy-mm-dd", input: "2025-02-01", want: "2025-02-01T00:00:00Z"},
		{name: "invalid", input: "not-a-date", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := parseDate(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.want, got.Format(time.RFC3339))
		})
	}
}

func TestDoctorQueryWindow(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 2, 25, 15, 4, 5, 987654321, time.UTC)
	start, end := doctorQueryWindow(now)

	assert.Equal(t, "2026-02-25T15:04:05Z", end.Format(time.RFC3339))
	assert.Equal(t, "2026-01-26T15:04:05Z", start.Format(time.RFC3339))
}

func TestParseServiceCategories(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		wantLen int
		wantErr bool
	}{
		{name: "valid", input: "compute,storage", wantLen: 2},
		{name: "valid aliases", input: "bare-metal, compute", wantLen: 2},
		{name: "empty", input: "", wantLen: 0},
		{name: "unsupported network", input: "network", wantErr: true},
		{name: "unsupported containers", input: "containers", wantErr: true},
		{name: "invalid", input: "wrong", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := parseServiceCategories(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Len(t, got, tt.wantLen)
		})
	}
}

func TestParseProductCategories(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		wantLen int
		wantErr bool
	}{
		{name: "valid", input: "instances,objectstorage", wantLen: 2},
		{name: "valid aliases", input: "apple-silicon, block_storage", wantLen: 2},
		{name: "empty", input: "", wantLen: 0},
		{name: "unsupported load balancer", input: "load-balancer", wantErr: true},
		{name: "unsupported kubernetes", input: "kubernetes", wantErr: true},
		{name: "invalid", input: "wrong", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := parseProductCategories(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Len(t, got, tt.wantLen)
		})
	}
}
