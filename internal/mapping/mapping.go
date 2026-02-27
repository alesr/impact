package mapping

import (
	"fmt"
	"strings"

	"github.com/alesr/impact/internal/plan"
	"github.com/alesr/impact/internal/scw/catalog"
)

type Result struct {
	Product *catalog.Product
	Qty     float64
	Matches []Match
}

type Match struct {
	Product catalog.Product
	Qty     float64
}

type ErrorCode string

const (
	ErrorCodeNotImplemented           ErrorCode = "not_implemented"
	ErrorCodeMissingRequiredAttribute ErrorCode = "missing_required_attribute"
	ErrorCodeNoCatalogMatch           ErrorCode = "no_catalog_match"
)

type Error struct {
	Code   ErrorCode
	Reason string
}

func (e *Error) Error() string {
	if e == nil {
		return ""
	}

	if e.Reason == "" {
		return string(e.Code)
	}

	return fmt.Sprintf("%s: %s", e.Code, e.Reason)
}

func Resolve(change plan.ResourceChange, products []catalog.Product) (Result, error) {
	attrs := change.After
	if len(attrs) == 0 {
		attrs = change.Before
	}

	zone := getString(attrs, "zone")
	region := getString(attrs, "region")
	if zone == "" {
		zone = change.Zone
	}
	if region == "" {
		region = change.Region
	}
	rawResourceType := strings.TrimSpace(getString(attrs, "type"))
	rawNodeType := strings.TrimSpace(getString(attrs, "node_type"))
	resourceTypeToken := normalizeToken(rawResourceType)
	nodeTypeToken := normalizeToken(rawNodeType)

	switch change.Type {
	case "scaleway_instance_server":
		if resourceTypeToken == "" {
			return Result{}, &Error{Code: ErrorCodeMissingRequiredAttribute, Reason: "missing required attribute: type"}
		}

		product := findBestProduct(products, isInstanceProduct, zone, region, resourceTypeToken, resourceTypeToken != "")
		if product != nil {
			return Result{Product: product, Qty: 1}, nil
		}

		return Result{}, noCatalogMatchError(zone, region, "type", rawResourceType)
	case "scaleway_baremetal_server":
		product := findBestProduct(products, isBaremetalProduct, zone, region, resourceTypeToken, resourceTypeToken != "")
		if product != nil {
			return Result{Product: product, Qty: 1}, nil
		}

		return Result{}, noCatalogMatchError(zone, region, "type", rawResourceType)
	case "scaleway_k8s_pool":
		if nodeTypeToken == "" {
			return Result{}, &Error{Code: ErrorCodeMissingRequiredAttribute, Reason: "missing required attribute: node_type"}
		}

		size := getFloat(attrs, "size", 1)
		product := findBestProduct(products, isInstanceProduct, zone, region, nodeTypeToken, nodeTypeToken != "")
		if product != nil {
			return Result{Product: product, Qty: size}, nil
		}

		return Result{}, noCatalogMatchError(zone, region, "node_type", rawNodeType)
	case "scaleway_lb":
		if resourceTypeToken == "" {
			return Result{}, &Error{Code: ErrorCodeMissingRequiredAttribute, Reason: "missing required attribute: type"}
		}

		lbTypeToken := normalizeLoadBalancerType(rawResourceType)
		product := findBestProduct(products, isLoadBalancerProduct, zone, region, lbTypeToken, lbTypeToken != "")
		if product != nil {
			return Result{Product: product, Qty: 1}, nil
		}

		return Result{}, noCatalogMatchError(zone, region, "type", rawResourceType)
	case "scaleway_block_volume":
		size := getFloat(attrs, "size_in_gb", 1)
		product := findBestProduct(products, isBlockStorageProduct, zone, region, "", false)
		if product != nil {
			return Result{Product: product, Qty: size}, nil
		}

		return Result{}, noCatalogMatchError(zone, region, "", "")
	case "scaleway_rdb_instance":
		if nodeTypeToken == "" {
			return Result{}, &Error{Code: ErrorCodeMissingRequiredAttribute, Reason: "missing required attribute: node_type"}
		}

		product := findBestProduct(products, isRDBProduct, zone, region, nodeTypeToken, nodeTypeToken != "")
		if product != nil {
			return Result{Product: product, Qty: 1}, nil
		}

		return Result{}, noCatalogMatchError(zone, region, "node_type", rawNodeType)
	case "scaleway_redis_cluster":
		if nodeTypeToken == "" {
			return Result{}, &Error{Code: ErrorCodeMissingRequiredAttribute, Reason: "missing required attribute: node_type"}
		}

		clusterSize := normalizeCount(getFloat(attrs, "cluster_size", 1))

		mainNode := findBestRedisRoleProduct(products, "main-node", zone, region, nodeTypeToken)
		if mainNode == nil {
			return Result{}, noCatalogMatchError(zone, region, "node_type", rawNodeType)
		}

		matches := []Match{{Product: *mainNode, Qty: 1}}
		if clusterSize > 1 {
			additionalNode := findBestRedisRoleProduct(products, "additional-node", zone, region, nodeTypeToken)
			if additionalNode == nil {
				return Result{}, noCatalogMatchError(zone, region, "node_type", rawNodeType)
			}

			matches = append(matches, Match{Product: *additionalNode, Qty: float64(clusterSize - 1)})
		}

		return Result{Product: mainNode, Qty: float64(clusterSize), Matches: matches}, nil
	default:
		return Result{}, &Error{Code: ErrorCodeNotImplemented, Reason: "not implemented"}
	}
}

func normalizeCount(value float64) int {
	if value < 1 {
		return 1
	}

	return int(value)
}

func findBestRedisRoleProduct(products []catalog.Product, role, zone, region, nodeTypeToken string) *catalog.Product {
	rolePath := "/storage/redis/" + strings.ToLower(strings.TrimSpace(role)) + "/"
	filtered := make([]catalog.Product, 0, len(products))

	for _, product := range products {
		sku := strings.ToLower(product.SKU)
		if !strings.Contains(sku, rolePath) {
			continue
		}

		filtered = append(filtered, product)
	}

	return findBestProduct(filtered, isRedisProduct, zone, region, nodeTypeToken, nodeTypeToken != "")
}

func noCatalogMatchError(zone, region, typeKey, typeValue string) error {
	parts := make([]string, 0, 4)

	if typeKey != "" {
		parts = append(parts, fmt.Sprintf("%s=%s", typeKey, typeValue))
	}
	if zone != "" {
		parts = append(parts, fmt.Sprintf("zone=%s", zone))
	}
	if region != "" {
		parts = append(parts, fmt.Sprintf("region=%s", region))
	}

	reason := "no catalog match"
	if len(parts) > 0 {
		reason += " (" + strings.Join(parts, ", ") + ")"
	}

	return &Error{Code: ErrorCodeNoCatalogMatch, Reason: reason}
}

func findBestProduct(products []catalog.Product, matchResource func(catalog.Product) bool, zone, region, typeToken string, requireType bool) *catalog.Product {
	bestIndex := -1
	bestScore := -1

	for i := range products {
		product := products[i]
		if !matchResource(product) {
			continue
		}

		localityScore, ok := scoreLocality(product, zone, region)
		if !ok {
			continue
		}

		score := localityScore
		if typeToken != "" {
			if !matchesToken(product, typeToken) {
				if requireType {
					continue
				}
			} else {
				score += 100
			}
		}

		if bestIndex < 0 || score > bestScore || (score == bestScore && strings.Compare(product.SKU, products[bestIndex].SKU) < 0) {
			bestIndex = i
			bestScore = score
		}
	}

	if bestIndex < 0 {
		return nil
	}

	return &products[bestIndex]
}

func getString(attrs map[string]any, key string) string {
	v, ok := attrs[key]
	if !ok || v == nil {
		return ""
	}
	s, ok := v.(string)
	if !ok {
		return ""
	}
	return s
}

func getFloat(attrs map[string]any, key string, fallback float64) float64 {
	v, ok := attrs[key]
	if !ok || v == nil {
		return fallback
	}
	switch n := v.(type) {
	case float64:
		return n
	case int:
		return float64(n)
	default:
		return fallback
	}
}

func normalizeToken(v string) string {
	v = strings.ToLower(strings.TrimSpace(v))
	v = strings.ReplaceAll(v, "-", "")
	v = strings.ReplaceAll(v, "_", "")
	v = strings.ReplaceAll(v, " ", "")
	v = strings.ReplaceAll(v, "/", "")
	return v
}

func isInstanceProduct(product catalog.Product) bool {
	category := normalizeToken(product.ProductCategory)
	sku := strings.ToLower(product.SKU)
	return category == "instance" || category == "instances" || strings.Contains(sku, "/compute/")
}

func isBaremetalProduct(product catalog.Product) bool {
	category := normalizeToken(product.ProductCategory)
	sku := strings.ToLower(product.SKU)
	return category == "elasticmetal" || category == "baremetal" || strings.Contains(sku, "/elastic-metal/") || strings.Contains(sku, "/apple-silicon/")
}

func isLoadBalancerProduct(product catalog.Product) bool {
	category := normalizeToken(product.ProductCategory)
	sku := strings.ToLower(product.SKU)
	return category == "loadbalancer" || strings.Contains(sku, "/network/lb/") || strings.Contains(sku, "/loadbalancer/")
}

func isBlockStorageProduct(product catalog.Product) bool {
	category := normalizeToken(product.ProductCategory)
	sku := strings.ToLower(product.SKU)
	return category == "blockstorage" || strings.Contains(sku, "/storage/block/")
}

func isRDBProduct(product catalog.Product) bool {
	sku := strings.ToLower(product.SKU)
	return strings.Contains(sku, "/storage/rdb/")
}

func isRedisProduct(product catalog.Product) bool {
	sku := strings.ToLower(product.SKU)
	return strings.Contains(sku, "/storage/redis/")
}

func scoreLocality(product catalog.Product, zone, region string) (int, bool) {
	if zone != "" {
		if strings.EqualFold(product.Locality.Zone, zone) {
			return 50, true
		}

		zoneRegion := regionFromZone(zone)
		if zoneRegion != "" && strings.EqualFold(product.Locality.Region, zoneRegion) {
			return 35, true
		}

		if strings.EqualFold(product.Locality.Region, region) && region != "" {
			return 35, true
		}

		if strings.EqualFold(regionFromZone(product.Locality.Zone), zoneRegion) && zoneRegion != "" {
			return 30, true
		}

		if product.Locality.Global != nil && *product.Locality.Global {
			return 5, true
		}

		return 0, false
	}

	if region != "" {
		if strings.EqualFold(product.Locality.Region, region) {
			return 40, true
		}

		if strings.EqualFold(regionFromZone(product.Locality.Zone), region) {
			return 35, true
		}

		if product.Locality.Global != nil && *product.Locality.Global {
			return 5, true
		}

		return 0, false
	}
	return 1, true
}

func regionFromZone(zone string) string {
	parts := strings.Split(strings.ToLower(strings.TrimSpace(zone)), "-")
	if len(parts) < 2 {
		return ""
	}
	return parts[0] + "-" + parts[1]
}

func matchesToken(product catalog.Product, token string) bool {
	haystack := normalizeToken(product.SKU + " " + product.Product + " " + product.Variant + " " + product.Description)
	return strings.Contains(haystack, token)
}

func normalizeLoadBalancerType(raw string) string {
	value := strings.ToLower(strings.TrimSpace(raw))
	if value == "" {
		return ""
	}

	value = strings.TrimPrefix(value, "lb-")
	value = strings.ReplaceAll(value, "_", "-")
	value = strings.ReplaceAll(value, " ", "")

	return normalizeToken("loadbalancer-" + value)
}
