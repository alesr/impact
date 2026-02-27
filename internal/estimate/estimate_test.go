package estimate

import (
	"testing"

	"github.com/alesr/impact/internal/plan"
	"github.com/alesr/impact/internal/scw/catalog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func float64ptr(v float64) *float64 { return &v }

func TestBuild(t *testing.T) {
	t.Parallel()

	t.Run("create computes totals when footprint data exists", func(t *testing.T) {
		t.Parallel()

		products := []catalog.Product{{
			SKU:             "/compute/dev1_m/test",
			ProductCategory: "instances",
			Locality:        catalog.Locality{Zone: "fr-par-2"},
			UnitOfMeasure:   catalog.UnitOfMeasure{Unit: "hour", Size: 1},
			EnvironmentalImpactEstimation: &catalog.EnvironmentalEstimation{
				KgCO2Equivalent: float64ptr(0.001),
				M3WaterUsage:    float64ptr(0.000001),
			},
		}}

		changes := []plan.ResourceChange{{
			Address: "scaleway_instance_server.web",
			Type:    "scaleway_instance_server",
			Actions: []string{"create"},
			After:   map[string]any{"zone": "fr-par-2", "type": "DEV1-M"},
		}}

		report := Build(changes, products)
		require.Len(t, report.Rows, 1)
		assert.Greater(t, report.Totals.KgCO2eMonth, 0.0)
		assert.True(t, report.Rows[0].KgCO2eKnown)
		assert.True(t, report.Rows[0].M3WaterKnown)
		assert.True(t, report.Totals.KgCO2eKnown)
		assert.True(t, report.Totals.M3WaterKnown)
		assert.Equal(t, 0, report.Totals.UnknownRows)
		assert.Empty(t, report.Unsupported)
	})

	t.Run("missing footprint marks unknown flags and partial totals", func(t *testing.T) {
		t.Parallel()

		products := []catalog.Product{{
			SKU:             "/compute/dev1_m/test-no-footprint",
			ProductCategory: "instances",
			Locality:        catalog.Locality{Zone: "fr-par-2"},
			UnitOfMeasure:   catalog.UnitOfMeasure{Unit: "hour", Size: 1},
		}}

		changes := []plan.ResourceChange{{
			Address: "scaleway_instance_server.web",
			Type:    "scaleway_instance_server",
			Actions: []string{"create"},
			After:   map[string]any{"zone": "fr-par-2", "type": "DEV1-M"},
		}}

		report := Build(changes, products)
		require.Len(t, report.Rows, 1)
		assert.False(t, report.Rows[0].KgCO2eKnown)
		assert.False(t, report.Rows[0].M3WaterKnown)
		assert.False(t, report.Totals.KgCO2eKnown)
		assert.False(t, report.Totals.M3WaterKnown)
		assert.Equal(t, 1, report.Totals.UnknownRows)
		assert.Equal(t, 0.0, report.Totals.KgCO2eMonth)
		assert.Equal(t, 0.0, report.Totals.M3WaterMonth)
	})

	t.Run("unsupported resources include reason metadata", func(t *testing.T) {
		t.Parallel()

		changes := []plan.ResourceChange{{
			Address: "scaleway_vpc_private_network.main",
			Type:    "scaleway_vpc_private_network",
			Actions: []string{"create"},
			After:   map[string]any{"region": "fr-par"},
		}}

		report := Build(changes, nil)
		require.Len(t, report.Unsupported, 1)
		assert.Equal(t, "scaleway_vpc_private_network.main", report.Unsupported[0].Address)
		assert.Equal(t, "not_implemented", report.Unsupported[0].Code)
		assert.NotEmpty(t, report.Unsupported[0].Reason)
	})

	t.Run("redis cluster expands to main and additional node rows", func(t *testing.T) {
		t.Parallel()

		products := []catalog.Product{
			{
				SKU:             "/storage/redis/main-node/RED1-micro/fr-par-1",
				ProductCategory: "instance",
				Locality:        catalog.Locality{Zone: "fr-par-1"},
				UnitOfMeasure:   catalog.UnitOfMeasure{Unit: "hour", Size: 1},
				EnvironmentalImpactEstimation: &catalog.EnvironmentalEstimation{
					KgCO2Equivalent: float64ptr(0.001),
				},
			},
			{
				SKU:             "/storage/redis/additional-node/RED1-micro/fr-par-1",
				ProductCategory: "instance",
				Locality:        catalog.Locality{Zone: "fr-par-1"},
				UnitOfMeasure:   catalog.UnitOfMeasure{Unit: "hour", Size: 1},
				EnvironmentalImpactEstimation: &catalog.EnvironmentalEstimation{
					KgCO2Equivalent: float64ptr(0.0005),
				},
			},
		}

		changes := []plan.ResourceChange{{
			Address: "scaleway_redis_cluster.cache",
			Type:    "scaleway_redis_cluster",
			Actions: []string{"create"},
			After:   map[string]any{"zone": "fr-par-1", "node_type": "RED1-MICRO", "cluster_size": 3.0},
		}}

		report := Build(changes, products)
		require.Len(t, report.Rows, 2)
		assert.Equal(t, "/storage/redis/main-node/RED1-micro/fr-par-1", report.Rows[0].SKU)
		assert.Equal(t, "/storage/redis/additional-node/RED1-micro/fr-par-1", report.Rows[1].SKU)
		assert.Greater(t, report.Rows[0].KgCO2eMonth, 0.0)
		assert.Greater(t, report.Rows[1].KgCO2eMonth, 0.0)
	})

	t.Run("replace with equivalent before and after yields net zero totals", func(t *testing.T) {
		t.Parallel()

		products := []catalog.Product{{
			SKU:                           "/compute/dev1_m/test",
			ProductCategory:               "instances",
			Locality:                      catalog.Locality{Zone: "fr-par-2"},
			UnitOfMeasure:                 catalog.UnitOfMeasure{Unit: "hour", Size: 1},
			EnvironmentalImpactEstimation: &catalog.EnvironmentalEstimation{KgCO2Equivalent: float64ptr(0.001)},
		}}

		changes := []plan.ResourceChange{{
			Address: "scaleway_instance_server.web",
			Type:    "scaleway_instance_server",
			Actions: []string{"delete", "create"},
			Before:  map[string]any{"zone": "fr-par-2", "type": "DEV1-M"},
			After:   map[string]any{"zone": "fr-par-2", "type": "DEV1-M"},
		}}

		report := Build(changes, products)
		require.Len(t, report.Rows, 2)
		assert.Equal(t, 0.0, report.Totals.KgCO2eMonth)
	})

	t.Run("update contributes before-after delta", func(t *testing.T) {
		t.Parallel()

		products := []catalog.Product{
			{
				SKU:                           "/compute/dev1_s/test",
				ProductCategory:               "instances",
				Locality:                      catalog.Locality{Zone: "fr-par-2"},
				UnitOfMeasure:                 catalog.UnitOfMeasure{Unit: "hour", Size: 1},
				EnvironmentalImpactEstimation: &catalog.EnvironmentalEstimation{KgCO2Equivalent: float64ptr(0.001)},
			},
			{
				SKU:                           "/compute/dev1_m/test",
				ProductCategory:               "instances",
				Locality:                      catalog.Locality{Zone: "fr-par-2"},
				UnitOfMeasure:                 catalog.UnitOfMeasure{Unit: "hour", Size: 1},
				EnvironmentalImpactEstimation: &catalog.EnvironmentalEstimation{KgCO2Equivalent: float64ptr(0.002)},
			},
		}

		changes := []plan.ResourceChange{{
			Address: "scaleway_instance_server.web",
			Type:    "scaleway_instance_server",
			Actions: []string{"update"},
			Before:  map[string]any{"zone": "fr-par-2", "type": "DEV1-S"},
			After:   map[string]any{"zone": "fr-par-2", "type": "DEV1-M"},
		}}

		report := Build(changes, products)
		require.Len(t, report.Rows, 2)
		assert.Equal(t, "update", report.Rows[0].Action)
		assert.Equal(t, "update", report.Rows[1].Action)
		assert.InDelta(t, 0.73, report.Totals.KgCO2eMonth, 1e-9)
	})

	t.Run("unit of measure size scales monthly footprint", func(t *testing.T) {
		t.Parallel()

		products := []catalog.Product{{
			SKU:                           "/storage/block/snapshot/fr-par-1",
			ProductCategory:               "block_storage",
			Locality:                      catalog.Locality{Zone: "fr-par-1"},
			UnitOfMeasure:                 catalog.UnitOfMeasure{Unit: "month", Size: 100},
			EnvironmentalImpactEstimation: &catalog.EnvironmentalEstimation{KgCO2Equivalent: float64ptr(2)},
		}}

		changes := []plan.ResourceChange{{
			Address: "scaleway_block_volume.data",
			Type:    "scaleway_block_volume",
			Actions: []string{"create"},
			After:   map[string]any{"zone": "fr-par-1", "size_in_gb": 250.0},
		}}

		report := Build(changes, products)
		require.Len(t, report.Rows, 1)
		assert.Equal(t, 5.0, report.Rows[0].KgCO2eMonth)
	})

	t.Run("partial footprint tracks known flags independently", func(t *testing.T) {
		t.Parallel()

		products := []catalog.Product{{
			SKU:             "/compute/dev1_m/test",
			ProductCategory: "instances",
			Locality:        catalog.Locality{Zone: "fr-par-2"},
			UnitOfMeasure:   catalog.UnitOfMeasure{Unit: "hour", Size: 1},
			EnvironmentalImpactEstimation: &catalog.EnvironmentalEstimation{
				KgCO2Equivalent: float64ptr(0.001),
			},
		}}

		changes := []plan.ResourceChange{{
			Address: "scaleway_instance_server.web",
			Type:    "scaleway_instance_server",
			Actions: []string{"create"},
			After:   map[string]any{"zone": "fr-par-2", "type": "DEV1-M"},
		}}

		report := Build(changes, products)
		require.Len(t, report.Rows, 1)
		assert.True(t, report.Rows[0].KgCO2eKnown)
		assert.False(t, report.Rows[0].M3WaterKnown)
		assert.True(t, report.Totals.KgCO2eKnown)
		assert.False(t, report.Totals.M3WaterKnown)
	})
}
