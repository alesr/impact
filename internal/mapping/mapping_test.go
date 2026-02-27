package mapping

import (
	"testing"

	"github.com/alesr/impact/internal/plan"
	"github.com/alesr/impact/internal/scw/catalog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolve(t *testing.T) {
	t.Parallel()

	t.Run("instance maps by zone and type", func(t *testing.T) {
		t.Parallel()

		products := []catalog.Product{{
			SKU:             "/compute/pop2_hc_2c_4g/run_fr-par-2",
			ProductCategory: "instances",
			Locality:        catalog.Locality{Zone: "fr-par-2"},
		}}

		change := plan.ResourceChange{Type: "scaleway_instance_server", After: map[string]any{"zone": "fr-par-2", "type": "POP2-HC-2C-4G"}}

		res, err := Resolve(change, products)
		require.NoError(t, err)
		require.NotNil(t, res.Product)
		assert.Equal(t, "/compute/pop2_hc_2c_4g/run_fr-par-2", res.Product.SKU)
	})

	t.Run("rdb maps only rdb sku families", func(t *testing.T) {
		t.Parallel()

		products := []catalog.Product{
			{SKU: "/storage/local/ssd/storage", ProductCategory: "storage", Locality: catalog.Locality{Region: "fr-par"}},
			{SKU: "/storage/rdb/instance/db-dev-s/fr-par", ProductCategory: "database", Locality: catalog.Locality{Region: "fr-par"}, Product: "RDB DB-DEV-S"},
		}

		change := plan.ResourceChange{Type: "scaleway_rdb_instance", After: map[string]any{"region": "fr-par", "node_type": "DB-DEV-S"}}

		res, err := Resolve(change, products)
		require.NoError(t, err)
		require.NotNil(t, res.Product)
		assert.Equal(t, "/storage/rdb/instance/db-dev-s/fr-par", res.Product.SKU)
	})

	t.Run("redis strict mapping expands main and additional nodes", func(t *testing.T) {
		t.Parallel()

		products := []catalog.Product{
			{SKU: "/storage/local/ssd/storage", ProductCategory: "storage", Locality: catalog.Locality{Zone: "fr-par-1"}},
			{SKU: "/storage/redis/main-node/RED1-micro/fr-par-1", ProductCategory: "instance", Locality: catalog.Locality{Zone: "fr-par-1"}, Product: "Redis RED1-MICRO"},
			{SKU: "/storage/redis/additional-node/RED1-micro/fr-par-1", ProductCategory: "instance", Locality: catalog.Locality{Zone: "fr-par-1"}, Product: "Redis RED1-MICRO"},
		}

		change := plan.ResourceChange{Type: "scaleway_redis_cluster", After: map[string]any{"zone": "fr-par-1", "node_type": "RED1-MICRO", "cluster_size": 2.0}}

		res, err := Resolve(change, products)
		require.NoError(t, err)
		require.NotNil(t, res.Product)
		assert.Equal(t, "/storage/redis/main-node/RED1-micro/fr-par-1", res.Product.SKU)
		assert.Equal(t, 2.0, res.Qty)
		require.Len(t, res.Matches, 2)
		assert.Equal(t, "/storage/redis/main-node/RED1-micro/fr-par-1", res.Matches[0].Product.SKU)
		assert.Equal(t, 1.0, res.Matches[0].Qty)
		assert.Equal(t, "/storage/redis/additional-node/RED1-micro/fr-par-1", res.Matches[1].Product.SKU)
		assert.Equal(t, 1.0, res.Matches[1].Qty)
	})

	t.Run("redis single-node uses main node sku only", func(t *testing.T) {
		t.Parallel()

		products := []catalog.Product{
			{SKU: "/storage/redis/main-node/RED1-micro/fr-par-1", ProductCategory: "instance", Locality: catalog.Locality{Zone: "fr-par-1"}, Product: "Redis RED1-MICRO"},
			{SKU: "/storage/redis/additional-node/RED1-micro/fr-par-1", ProductCategory: "instance", Locality: catalog.Locality{Zone: "fr-par-1"}, Product: "Redis RED1-MICRO"},
		}

		change := plan.ResourceChange{Type: "scaleway_redis_cluster", After: map[string]any{"zone": "fr-par-1", "node_type": "RED1-MICRO", "cluster_size": 1.0}}

		res, err := Resolve(change, products)
		require.NoError(t, err)
		require.NotNil(t, res.Product)
		assert.Equal(t, "/storage/redis/main-node/RED1-micro/fr-par-1", res.Product.SKU)
		require.Len(t, res.Matches, 1)
		assert.Equal(t, 1.0, res.Matches[0].Qty)
	})

	t.Run("kubernetes pool maps to node sku instead of control plane", func(t *testing.T) {
		t.Parallel()

		products := []catalog.Product{
			{SKU: "/kubernetes/kapsule/control-plane/fr-par", ProductCategory: "kubernetes", Locality: catalog.Locality{Region: "fr-par"}, Product: "Kapsule Control Plane"},
			{SKU: "/compute/dev1_m/run_par1", ProductCategory: "instances", Locality: catalog.Locality{Zone: "fr-par-1"}, Product: "DEV1-M"},
		}

		change := plan.ResourceChange{Type: "scaleway_k8s_pool", After: map[string]any{"region": "fr-par", "node_type": "DEV1-M", "size": 3.0}}

		res, err := Resolve(change, products)
		require.NoError(t, err)
		require.NotNil(t, res.Product)
		assert.Equal(t, "/compute/dev1_m/run_par1", res.Product.SKU)
		assert.Equal(t, 3.0, res.Qty)
	})

	t.Run("falls back to plan default zone when zone attribute is nil", func(t *testing.T) {
		t.Parallel()

		products := []catalog.Product{{
			SKU:             "/network/lb/lb-s/fr-par-1",
			ProductCategory: "load_balancer",
			Locality:        catalog.Locality{Zone: "fr-par-1"},
			Product:         "Load Balancer S",
		}}

		change := plan.ResourceChange{
			Type: "scaleway_lb",
			After: map[string]any{
				"type": "LB-S",
				"zone": nil,
			},
			Zone: "fr-par-1",
		}

		res, err := Resolve(change, products)
		require.NoError(t, err)
		require.NotNil(t, res.Product)
		assert.Equal(t, "/network/lb/lb-s/fr-par-1", res.Product.SKU)
	})

	t.Run("returns not implemented for unsupported resource types", func(t *testing.T) {
		t.Parallel()

		change := plan.ResourceChange{Type: "scaleway_object_bucket", After: map[string]any{"name": "logs"}}
		_, err := Resolve(change, []catalog.Product{{ProductCategory: "Instance"}})
		require.Error(t, err)

		var mappingErr *Error
		require.ErrorAs(t, err, &mappingErr)
		assert.Equal(t, ErrorCodeNotImplemented, mappingErr.Code)
	})

	t.Run("returns missing required attribute when required fields are absent", func(t *testing.T) {
		t.Parallel()

		change := plan.ResourceChange{Type: "scaleway_rdb_instance", After: map[string]any{"region": "fr-par"}}
		_, err := Resolve(change, nil)
		require.Error(t, err)

		var mappingErr *Error
		require.ErrorAs(t, err, &mappingErr)
		assert.Equal(t, ErrorCodeMissingRequiredAttribute, mappingErr.Code)
	})
}
