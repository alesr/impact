package footprint

import (
	"testing"
	"time"

	envfootprint "github.com/scaleway/scaleway-sdk-go/api/environmental_footprint/v1alpha1"
	"github.com/scaleway/scaleway-sdk-go/scw"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewClient(t *testing.T) {
	t.Parallel()

	t.Run("requires access key", func(t *testing.T) {
		t.Parallel()

		_, err := NewClient("", "secret")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "access key is empty")
	})

	t.Run("requires secret key", func(t *testing.T) {
		t.Parallel()

		_, err := NewClient("SCWXXXXXXXXXXXXXXXXX", "")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "secret key is empty")
	})
}

func TestFromSDKImpactDataResponse(t *testing.T) {
	t.Parallel()

	t.Run("maps nested response data", func(t *testing.T) {
		t.Parallel()

		start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
		end := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)

		in := &envfootprint.ImpactDataResponse{
			StartDate: &start,
			EndDate:   &end,
			TotalImpact: &envfootprint.Impact{
				KgCo2Equivalent: 1.2,
				M3WaterUsage:    3.4,
			},
			Projects: []*envfootprint.ProjectImpact{{
				ProjectID: "project-id",
				TotalProjectImpact: &envfootprint.Impact{
					KgCo2Equivalent: 0.5,
					M3WaterUsage:    0.6,
				},
				Regions: []*envfootprint.RegionImpact{{
					Region: scw.RegionFrPar,
					Zones: []*envfootprint.ZoneImpact{{
						Zone: scw.ZoneFrPar1,
						Skus: []*envfootprint.SkuImpact{{
							Sku:             "sku-id",
							ServiceCategory: envfootprint.ServiceCategoryCompute,
							ProductCategory: envfootprint.ProductCategoryInstances,
							TotalSkuImpact: &envfootprint.Impact{
								KgCo2Equivalent: 0.1,
								M3WaterUsage:    0.2,
							},
						}},
					}},
				}},
			}},
		}

		out := fromSDKImpactDataResponse(in)

		assert.Equal(t, start, out.StartDate)
		assert.Equal(t, end, out.EndDate)
		assert.InDelta(t, 1.2, out.TotalImpact.KgCO2Equivalent, 1e-6)
		require.Len(t, out.Projects, 1)
		assert.Equal(t, "project-id", out.Projects[0].ProjectID)
		require.Len(t, out.Projects[0].Regions, 1)
		assert.Equal(t, "fr-par", out.Projects[0].Regions[0].Region)
		require.Len(t, out.Projects[0].Regions[0].Zones, 1)
		assert.Equal(t, "fr-par-1", out.Projects[0].Regions[0].Zones[0].Zone)
		require.Len(t, out.Projects[0].Regions[0].Zones[0].SKUs, 1)
		assert.Equal(t, "sku-id", out.Projects[0].Regions[0].Zones[0].SKUs[0].SKU)
		assert.Equal(t, "compute", out.Projects[0].Regions[0].Zones[0].SKUs[0].ServiceCategory)
		assert.Equal(t, "instances", out.Projects[0].Regions[0].Zones[0].SKUs[0].ProductCategory)
	})

	t.Run("returns empty non-nil response for nil input", func(t *testing.T) {
		t.Parallel()

		out := fromSDKImpactDataResponse(nil)
		assert.NotNil(t, out)
		assert.Empty(t, out.Projects)
	})
}
