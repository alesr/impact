package footprint

import "time"

type ServiceCategory string

const (
	ServiceCategoryBaremetal ServiceCategory = "baremetal"
	ServiceCategoryCompute   ServiceCategory = "compute"
	ServiceCategoryStorage   ServiceCategory = "storage"
)

type ProductCategory string

const (
	ProductCategoryAppleSilicon  ProductCategory = "apple_silicon"
	ProductCategoryBlockStorage  ProductCategory = "block_storage"
	ProductCategoryDedibox       ProductCategory = "dedibox"
	ProductCategoryElasticMetal  ProductCategory = "elastic_metal"
	ProductCategoryInstances     ProductCategory = "instances"
	ProductCategoryObjectStorage ProductCategory = "object_storage"
)

type QueryImpactDataRequest struct {
	OrganizationID    string
	StartDate         *time.Time
	EndDate           *time.Time
	ProjectIDs        []string
	Regions           []string
	Zones             []string
	ServiceCategories []ServiceCategory
	ProductCategories []ProductCategory
}

type QueryImpactDataResponse struct {
	StartDate   time.Time       `json:"start_date"`
	EndDate     time.Time       `json:"end_date"`
	TotalImpact TotalImpact     `json:"total_impact"`
	Projects    []ProjectImpact `json:"projects"`
}

type TotalImpact struct {
	KgCO2Equivalent float64 `json:"kg_co2_equivalent"`
	M3WaterUsage    float64 `json:"m3_water_usage"`
}

type ProjectImpact struct {
	ProjectID          string         `json:"project_id"`
	TotalProjectImpact TotalImpact    `json:"total_project_impact"`
	Regions            []RegionImpact `json:"regions"`
}

type RegionImpact struct {
	Region            string       `json:"region"`
	TotalRegionImpact TotalImpact  `json:"total_region_impact"`
	Zones             []ZoneImpact `json:"zones"`
}

type ZoneImpact struct {
	Zone            string      `json:"zone"`
	TotalZoneImpact TotalImpact `json:"total_zone_impact"`
	SKUs            []SKUImpact `json:"skus"`
}

type SKUImpact struct {
	SKU             string      `json:"sku"`
	TotalSKUImpact  TotalImpact `json:"total_sku_impact"`
	ServiceCategory string      `json:"service_category"`
	ProductCategory string      `json:"product_category"`
}
