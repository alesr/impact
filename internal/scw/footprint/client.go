package footprint

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	envfootprint "github.com/scaleway/scaleway-sdk-go/api/environmental_footprint/v1alpha1"
	"github.com/scaleway/scaleway-sdk-go/scw"
)

type Client struct {
	api *envfootprint.UserAPI
}

func NewClient(accessKey, secretKey string, opts ...Option) (*Client, error) {
	if accessKey == "" {
		return nil, errors.New("could not create footprint client: access key is empty")
	}
	if secretKey == "" {
		return nil, errors.New("could not create footprint client: secret key is empty")
	}

	var cfg options
	for _, opt := range opts {
		if opt == nil {
			continue
		}
		opt(&cfg)
	}

	sdkOpts := []scw.ClientOption{scw.WithAuth(accessKey, secretKey)}
	if cfg.baseURL != "" {
		sdkOpts = append(sdkOpts, scw.WithAPIURL(cfg.baseURL))
	}
	if cfg.userAgent != "" {
		sdkOpts = append(sdkOpts, scw.WithUserAgent(cfg.userAgent))
	}
	if cfg.httpClient != nil {
		sdkOpts = append(sdkOpts, scw.WithHTTPClient(cfg.httpClient))
	} else if cfg.timeout > 0 {
		sdkOpts = append(sdkOpts, scw.WithHTTPClient(&http.Client{Timeout: cfg.timeout}))
	}

	client, err := scw.NewClient(sdkOpts...)
	if err != nil {
		return nil, fmt.Errorf("could not create footprint client: %w", err)
	}

	return &Client{api: envfootprint.NewUserAPI(client)}, nil
}

func (c *Client) QueryImpactData(ctx context.Context, req QueryImpactDataRequest) (*QueryImpactDataResponse, error) {
	sdkReq := &envfootprint.UserAPIGetImpactDataRequest{
		OrganizationID:    req.OrganizationID,
		StartDate:         req.StartDate,
		EndDate:           req.EndDate,
		ProjectIDs:        req.ProjectIDs,
		Regions:           req.Regions,
		Zones:             req.Zones,
		ServiceCategories: make([]envfootprint.ServiceCategory, 0, len(req.ServiceCategories)),
		ProductCategories: make([]envfootprint.ProductCategory, 0, len(req.ProductCategories)),
	}

	for _, svc := range req.ServiceCategories {
		sdkReq.ServiceCategories = append(sdkReq.ServiceCategories, envfootprint.ServiceCategory(svc))
	}
	for _, cat := range req.ProductCategories {
		sdkReq.ProductCategories = append(sdkReq.ProductCategories, envfootprint.ProductCategory(cat))
	}

	resp, err := c.api.GetImpactData(sdkReq, scw.WithContext(ctx))
	if err != nil {
		return nil, fmt.Errorf("could not query impact data: %w", err)
	}

	return fromSDKImpactDataResponse(resp), nil
}

func fromSDKImpactDataResponse(resp *envfootprint.ImpactDataResponse) *QueryImpactDataResponse {
	if resp == nil {
		return &QueryImpactDataResponse{}
	}

	out := &QueryImpactDataResponse{}
	if resp.StartDate != nil {
		out.StartDate = *resp.StartDate
	}
	if resp.EndDate != nil {
		out.EndDate = *resp.EndDate
	}
	if resp.TotalImpact != nil {
		out.TotalImpact = fromSDKImpact(resp.TotalImpact)
	}

	out.Projects = make([]ProjectImpact, 0, len(resp.Projects))
	for _, project := range resp.Projects {
		if project == nil {
			continue
		}
		projectImpact := ProjectImpact{ProjectID: project.ProjectID}
		if project.TotalProjectImpact != nil {
			projectImpact.TotalProjectImpact = fromSDKImpact(project.TotalProjectImpact)
		}

		projectImpact.Regions = make([]RegionImpact, 0, len(project.Regions))
		for _, region := range project.Regions {
			if region == nil {
				continue
			}
			regionImpact := RegionImpact{Region: region.Region.String()}
			if region.TotalRegionImpact != nil {
				regionImpact.TotalRegionImpact = fromSDKImpact(region.TotalRegionImpact)
			}

			regionImpact.Zones = make([]ZoneImpact, 0, len(region.Zones))
			for _, zone := range region.Zones {
				if zone == nil {
					continue
				}
				zoneImpact := ZoneImpact{Zone: zone.Zone.String()}
				if zone.TotalZoneImpact != nil {
					zoneImpact.TotalZoneImpact = fromSDKImpact(zone.TotalZoneImpact)
				}

				zoneImpact.SKUs = make([]SKUImpact, 0, len(zone.Skus))
				for _, sku := range zone.Skus {
					if sku == nil {
						continue
					}
					skuImpact := SKUImpact{SKU: sku.Sku, ServiceCategory: sku.ServiceCategory.String(), ProductCategory: sku.ProductCategory.String()}
					if sku.TotalSkuImpact != nil {
						skuImpact.TotalSKUImpact = fromSDKImpact(sku.TotalSkuImpact)
					}
					zoneImpact.SKUs = append(zoneImpact.SKUs, skuImpact)
				}

				regionImpact.Zones = append(regionImpact.Zones, zoneImpact)
			}

			projectImpact.Regions = append(projectImpact.Regions, regionImpact)
		}

		out.Projects = append(out.Projects, projectImpact)
	}

	return out
}

func fromSDKImpact(in *envfootprint.Impact) TotalImpact {
	if in == nil {
		return TotalImpact{}
	}

	return TotalImpact{
		KgCO2Equivalent: float64(in.KgCo2Equivalent),
		M3WaterUsage:    float64(in.M3WaterUsage),
	}
}
