package footprint

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestQueryImpactDataResponseJSONUsesSnakeCaseKeys(t *testing.T) {
	t.Parallel()

	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 1, 31, 0, 0, 0, 0, time.UTC)

	rep := QueryImpactDataResponse{
		StartDate: start,
		EndDate:   end,
		TotalImpact: TotalImpact{
			KgCO2Equivalent: 1.23,
			M3WaterUsage:    4.56,
		},
		Projects: []ProjectImpact{{
			ProjectID: "project-id",
			TotalProjectImpact: TotalImpact{
				KgCO2Equivalent: 0.12,
				M3WaterUsage:    0.34,
			},
			Regions: []RegionImpact{{
				Region: "fr-par",
				TotalRegionImpact: TotalImpact{
					KgCO2Equivalent: 0.11,
					M3WaterUsage:    0.22,
				},
				Zones: []ZoneImpact{{
					Zone: "fr-par-1",
					TotalZoneImpact: TotalImpact{
						KgCO2Equivalent: 0.1,
						M3WaterUsage:    0.2,
					},
					SKUs: []SKUImpact{{
						SKU: "sku-id",
						TotalSKUImpact: TotalImpact{
							KgCO2Equivalent: 0.01,
							M3WaterUsage:    0.02,
						},
						ServiceCategory: "compute",
						ProductCategory: "instances",
					}},
				}},
			}},
		}},
	}

	data, err := json.Marshal(rep)
	require.NoError(t, err)

	var payload map[string]any
	require.NoError(t, json.Unmarshal(data, &payload))

	assert.Contains(t, payload, "start_date")
	assert.Contains(t, payload, "end_date")
	assert.Contains(t, payload, "total_impact")
	assert.Contains(t, payload, "projects")

	projects, ok := payload["projects"].([]any)
	require.True(t, ok)
	require.Len(t, projects, 1)

	project, ok := projects[0].(map[string]any)
	require.True(t, ok)
	assert.Contains(t, project, "project_id")
	assert.Contains(t, project, "total_project_impact")
	assert.Contains(t, project, "regions")
}
