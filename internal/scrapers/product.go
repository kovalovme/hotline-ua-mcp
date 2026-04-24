package scrapers

import (
	"fmt"
	"strings"

	"github.com/kovalovme/hotline-ua-mcp/internal/types"
)

// ParseProductHTML extracts a full Product from a product detail page.
//
// Primary sources (in order):
//  1. schema.org/Product JSON-LD  → basic fields (title, url, sku, price range,
//     rating, image, description)
//  2. window.__NUXT__ state.product.productValues → specs map
func ParseProductHTML(html []byte) (*types.Product, error) {
	ld, err := extractProductJSONLD(html)
	if err != nil {
		return nil, fmt.Errorf("product JSON-LD: %w", err)
	}

	p := &types.Product{}

	// --- fields from JSON-LD ---
	p.Title = jsonString(ld["name"])
	p.URL = jsonString(ld["url"])
	p.Description = jsonString(ld["description"])

	if sku := ld["sku"]; sku != nil {
		p.ID = strings.TrimSpace(fmt.Sprintf("%v", int64(jsonFloat64(sku))))
	}

	if imgs, ok := ld["image"].([]any); ok && len(imgs) > 0 {
		p.ImageURL = jsonString(imgs[0])
	}

	if agg, ok := ld["offers"].(map[string]any); ok {
		p.PriceMin = jsonFloat64(agg["lowPrice"])
		p.PriceMax = jsonFloat64(agg["highPrice"])
		p.Currency = jsonString(agg["priceCurrency"])
		p.OffersCount = jsonInt(agg["offerCount"])
	}

	if ar, ok := ld["aggregateRating"].(map[string]any); ok {
		p.Rating = jsonFloat64(ar["ratingValue"])
		p.ReviewCount = jsonInt(ar["reviewCount"])
	}

	// --- specs from __NUXT__ productValues ---
	nuxt, err := nuxtState(html)
	if err == nil {
		edges := digSlice(nuxt, "state", "product", "productValues", "edges")
		specs := make(map[string]string, len(edges))
		for _, e := range edges {
			node, _ := dig(e.(map[string]any), "node").(map[string]any)
			if node == nil {
				continue
			}
			isHeader, _ := node["isHeader"].(bool)
			if isHeader {
				continue
			}
			title := jsonString(node["title"])
			value := jsonString(node["value"])
			if title != "" && value != "" && title != "vendor" && title != "series" {
				specs[title] = value
			}
		}
		if len(specs) > 0 {
			p.Specs = specs
		}
	}

	return p, nil
}
