package app

import (
	"context"

	"github.com/alesr/impact/internal/scw/catalog"
)

var _ catalogProductLister = (*mockCatalogProductLister)(nil)

type mockCatalogProductLister struct {
	listAllProductsFunc func(context.Context) ([]catalog.Product, error)
}

func (m *mockCatalogProductLister) ListAllProducts(ctx context.Context) ([]catalog.Product, error) {
	return m.listAllProductsFunc(ctx)
}
