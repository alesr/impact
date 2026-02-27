package catalog

import (
	"testing"
	"time"

	productcatalog "github.com/scaleway/scaleway-sdk-go/api/product_catalog/v2alpha1"
	"github.com/scaleway/scaleway-sdk-go/scw"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFromSDKProduct(t *testing.T) {
	t.Parallel()

	t.Run("maps fields and optional environmental values", func(t *testing.T) {
		t.Parallel()

		global := true
		region := scw.RegionFrPar
		zone := scw.ZoneFrPar1
		kg := float32(1.25)
		endOfLife := time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC)

		in := &productcatalog.PublicCatalogProduct{
			Sku:             "sku-id",
			ServiceCategory: "compute",
			ProductCategory: "instances",
			Product:         "DEV1-M",
			Variant:         "run",
			Description:     "desc",
			Locality:        &productcatalog.PublicCatalogProductLocality{Global: &global, Region: &region, Zone: &zone},
			UnitOfMeasure:   &productcatalog.PublicCatalogProductUnitOfMeasure{Unit: productcatalog.PublicCatalogProductUnitOfMeasureCountableUnitHour, Size: 2},
			EnvironmentalImpactEstimation: &productcatalog.PublicCatalogProductEnvironmentalImpactEstimation{
				KgCo2Equivalent: &kg,
			},
			Status:      productcatalog.PublicCatalogProductStatusGeneralAvailability,
			EndOfLifeAt: &endOfLife,
			Badges:      []productcatalog.PublicCatalogProductProductBadge{productcatalog.PublicCatalogProductProductBadgeNewProduct},
		}

		out := fromSDKProduct(in)

		assert.Equal(t, "sku-id", out.SKU)
		assert.Equal(t, "compute", out.ServiceCategory)
		assert.Equal(t, "instances", out.ProductCategory)
		assert.Equal(t, "fr-par", out.Locality.Region)
		assert.Equal(t, "fr-par-1", out.Locality.Zone)
		assert.NotNil(t, out.Locality.Global)
		assert.True(t, *out.Locality.Global)
		assert.Equal(t, uint64(2), out.UnitOfMeasure.Size)
		assert.NotNil(t, out.EnvironmentalImpactEstimation)
		require.NotNil(t, out.EnvironmentalImpactEstimation.KgCO2Equivalent)
		assert.Equal(t, 1.25, *out.EnvironmentalImpactEstimation.KgCO2Equivalent)
		assert.Nil(t, out.EnvironmentalImpactEstimation.M3WaterUsage)
		assert.Equal(t, "general_availability", out.Status)
		assert.Equal(t, []string{"new_product"}, out.Badges)
	})

	t.Run("handles nil pointers", func(t *testing.T) {
		t.Parallel()

		out := fromSDKProduct(&productcatalog.PublicCatalogProduct{Sku: "sku-id"})
		assert.Equal(t, "sku-id", out.SKU)
		assert.Nil(t, out.EnvironmentalImpactEstimation)
		assert.Equal(t, uint64(0), out.UnitOfMeasure.Size)
	})
}
