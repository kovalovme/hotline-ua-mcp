package scrapers

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/kovalovme/hotline-ua-mcp/internal/types"
)

// BuildSearchURL constructs a catalog listing URL for the given query.
//
// hotline.ua's global search (`/ua/search/?q=…`) is client-side routed and
// does not return SSR product data. Catalog section URLs do include product
// listings in the initial __NUXT__ state. This function encodes the query as
// a URL-safe path segment and routes to the all-categories search path.
//
// Live recon (2026-04-24): the search SSR endpoint is internal
// (search.search-19-production) so a direct category browse is the only
// fully SSR-backed approach available without GraphQL.
func BuildSearchURL(query string) string {
	slug := queryToSlug(query)
	return fmt.Sprintf("https://hotline.ua/ua/mobile/mobilnye-telefony-i-smartfony/%s", slug)
}

// queryToSlug converts a search phrase to a URL slug appended as a filter.
// e.g. "iphone 15" → "?text=iphone+15"  (category-level text filter)
func queryToSlug(query string) string {
	v := url.Values{}
	v.Set("text", strings.TrimSpace(query))
	return "?" + v.Encode()
}

// ParseSearchHTML extracts product summaries from a catalog/search results page.
//
// Source: window.__NUXT__ → state.catalog.products.collection
func ParseSearchHTML(html []byte) ([]types.ProductSummary, error) {
	nuxt, err := nuxtState(html)
	if err != nil {
		return nil, fmt.Errorf("nuxt state: %w", err)
	}

	collection := digSlice(nuxt, "state", "catalog", "products", "collection")
	out := make([]types.ProductSummary, 0, len(collection))

	for _, item := range collection {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}

		id := strconv.FormatInt(int64(jsonFloat64(m["_id"])), 10)

		// vendor.title is the brand name; prepend to model title for display
		vendorTitle := jsonString(dig(m, "vendor", "title"))
		title := jsonString(m["title"])
		if vendorTitle != "" && !strings.HasPrefix(title, vendorTitle) {
			title = vendorTitle + " " + title
		}

		rawURL := jsonString(m["url"])
		fullURL := ""
		if rawURL != "" {
			fullURL = "https://hotline.ua/ua" + rawURL
		}

		imageURL := ""
		if links := digSlice(m, "imageLinks"); len(links) > 0 {
			if link, ok := links[0].(map[string]any); ok {
				imageURL = "https://hotline.ua" + jsonString(link["thumb"])
			}
		}

		out = append(out, types.ProductSummary{
			ID:          id,
			Title:       title,
			URL:         fullURL,
			ImageURL:    imageURL,
			PriceMin:    jsonFloat64(m["minPrice"]),
			PriceMax:    jsonFloat64(m["maxPrice"]),
			Currency:    "UAH",
			OffersCount: jsonInt(m["offerCount"]),
			ReviewCount: jsonInt(m["reviewsCount"]),
		})
	}

	return out, nil
}
