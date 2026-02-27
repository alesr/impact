package catalog

import (
	"context"
	"fmt"
	"net/http"

	productcatalog "github.com/scaleway/scaleway-sdk-go/api/product_catalog/v2alpha1"
	"github.com/scaleway/scaleway-sdk-go/scw"
)

type Client struct {
	api *productcatalog.PublicCatalogAPI
}

func NewClient(opts ...Option) (*Client, error) {
	var cfg options
	for _, opt := range opts {
		if opt == nil {
			continue
		}
		opt(&cfg)
	}

	sdkOpts := []scw.ClientOption{scw.WithoutAuth()}
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
		return nil, fmt.Errorf("could not create catalog client: %w", err)
	}

	return &Client{api: productcatalog.NewPublicCatalogAPI(client)}, nil
}

func (c *Client) ListProducts(ctx context.Context, req ListProductsRequest) (*ListProductsResponse, error) {
	page := int32(req.Page)
	pageSize := uint32(req.PageSize)

	sdkReq := &productcatalog.PublicCatalogAPIListPublicCatalogProductsRequest{}
	if req.Page > 0 {
		sdkReq.Page = &page
	}
	if req.PageSize > 0 {
		sdkReq.PageSize = &pageSize
	}

	resp, err := c.api.ListPublicCatalogProducts(sdkReq, scw.WithContext(ctx))
	if err != nil {
		return nil, fmt.Errorf("could not list products: %w", err)
	}

	products := make([]Product, 0, len(resp.Products))
	for _, p := range resp.Products {
		if p == nil {
			continue
		}
		products = append(products, fromSDKProduct(p))
	}

	return &ListProductsResponse{Products: products, TotalCount: int(resp.TotalCount)}, nil
}

func (c *Client) ListAllProducts(ctx context.Context) ([]Product, error) {
	page := int32(1)
	pageSize := uint32(100)
	resp, err := c.api.ListPublicCatalogProducts(
		&productcatalog.PublicCatalogAPIListPublicCatalogProductsRequest{Page: &page, PageSize: &pageSize},
		scw.WithContext(ctx),
		scw.WithAllPages(),
	)
	if err != nil {
		return nil, fmt.Errorf("could not list products: %w", err)
	}

	products := make([]Product, 0, len(resp.Products))
	for _, p := range resp.Products {
		if p == nil {
			continue
		}
		products = append(products, fromSDKProduct(p))
	}

	return products, nil
}

func fromSDKProduct(p *productcatalog.PublicCatalogProduct) Product {
	product := Product{
		SKU:             p.Sku,
		ServiceCategory: p.ServiceCategory,
		ProductCategory: p.ProductCategory,
		Product:         p.Product,
		Variant:         p.Variant,
		Description:     p.Description,
		Status:          p.Status.String(),
		EndOfLifeAt:     p.EndOfLifeAt,
	}

	if p.UnitOfMeasure != nil {
		product.UnitOfMeasure = UnitOfMeasure{Unit: p.UnitOfMeasure.Unit.String(), Size: p.UnitOfMeasure.Size}
	}

	if p.Locality != nil {
		if p.Locality.Global != nil {
			product.Locality.Global = p.Locality.Global
		}
		if p.Locality.Region != nil {
			product.Locality.Region = p.Locality.Region.String()
		}
		if p.Locality.Zone != nil {
			product.Locality.Zone = p.Locality.Zone.String()
		}
	}

	if p.EnvironmentalImpactEstimation != nil {
		env := &EnvironmentalEstimation{}
		if p.EnvironmentalImpactEstimation.KgCo2Equivalent != nil {
			v := float64(*p.EnvironmentalImpactEstimation.KgCo2Equivalent)
			env.KgCO2Equivalent = &v
		}
		if p.EnvironmentalImpactEstimation.M3WaterUsage != nil {
			v := float64(*p.EnvironmentalImpactEstimation.M3WaterUsage)
			env.M3WaterUsage = &v
		}
		product.EnvironmentalImpactEstimation = env
	}

	if len(p.Badges) > 0 {
		product.Badges = make([]string, 0, len(p.Badges))
		for _, badge := range p.Badges {
			product.Badges = append(product.Badges, badge.String())
		}
	}

	return product
}
