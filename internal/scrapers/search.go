package scrapers

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"unicode"

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

// FilterByQuery performs client-side keyword filtering on a product list.
//
// The hotline.ua SSR path ignores the ?text= query parameter — the catalog
// page renders all products in the category regardless of query. This function
// filters the already-parsed slice by checking that every word in query appears
// (case-insensitively) in the product title. It is the tool layer's
// responsibility to call this after ParseSearchHTML.
func FilterByQuery(results []types.ProductSummary, query string) []types.ProductSummary {
	words := strings.FieldsFunc(strings.ToLower(query), func(r rune) bool {
		return unicode.IsSpace(r) || unicode.IsPunct(r)
	})
	if len(words) == 0 {
		return results
	}
	out := results[:0:0]
	for _, r := range results {
		title := strings.ToLower(r.Title)
		match := true
		for _, w := range words {
			if !strings.Contains(title, w) {
				match = false
				break
			}
		}
		if match {
			out = append(out, r)
		}
	}
	return out
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
