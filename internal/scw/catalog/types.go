package catalog

import "time"

type ListProductsRequest struct {
	Page     int
	PageSize int
}

type ListProductsResponse struct {
	Products   []Product
	TotalCount int
}

type Product struct {
	SKU                           string
	ServiceCategory               string
	ProductCategory               string
	Product                       string
	Variant                       string
	Description                   string
	Locality                      Locality
	UnitOfMeasure                 UnitOfMeasure
	EnvironmentalImpactEstimation *EnvironmentalEstimation
	Status                        string
	EndOfLifeAt                   *time.Time
	Badges                        []string
}

type Locality struct {
	Global *bool
	Region string
	Zone   string
}

type UnitOfMeasure struct {
	Unit string
	Size uint64
}

type EnvironmentalEstimation struct {
	KgCO2Equivalent *float64
	M3WaterUsage    *float64
}
