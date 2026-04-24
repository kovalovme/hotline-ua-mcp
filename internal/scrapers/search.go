package scrapers

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"unicode"

	"github.com/kovalovme/hotline-ua-mcp/internal/types"
)

// SearchFilters holds optional parameters for search and category URLs.
type SearchFilters struct {
	Page     int
	PriceMin float64
	PriceMax float64
}

// BuildSearchURL constructs a catalog listing URL for the given query.
//
// hotline.ua's global search (`/ua/search/?q=…`) is client-side routed and
// does not return SSR product data. Catalog section URLs do include product
// listings in the initial __NUXT__ state. This function encodes the query as
// a URL-safe query string and routes to the all-categories search path.
//
// Live recon (2026-04-24): the search SSR endpoint is internal
// (search.search-19-production) so a direct category browse is the only
// fully SSR-backed approach available without GraphQL.
func BuildSearchURL(query string) string {
	return BuildSearchURLFiltered(query, SearchFilters{})
}

// BuildSearchURLFiltered builds a search URL with optional pagination and price filters.
//
// Pagination uses ?page=N (standard Nuxt.js convention — confirmed as most
// likely candidate; verify via DevTools if live results differ).
// Price filter params are ?priceMin=N&priceMax=M.
func BuildSearchURLFiltered(query string, f SearchFilters) string {
	v := url.Values{}
	v.Set("text", strings.TrimSpace(query))
	if f.Page > 1 {
		v.Set("page", strconv.Itoa(f.Page))
	}
	if f.PriceMin > 0 {
		v.Set("priceMin", strconv.FormatFloat(f.PriceMin, 'f', 0, 64))
	}
	if f.PriceMax > 0 {
		v.Set("priceMax", strconv.FormatFloat(f.PriceMax, 'f', 0, 64))
	}
	return "https://hotline.ua/ua/mobile/mobilnye-telefony-i-smartfony/?" + v.Encode()
}

// BuildCategoryURL constructs a category browse URL with optional pagination
// and price filters. slug should be the path segment(s) after /ua/, e.g.
// "mobile/mobilnye-telefony-i-smartfony".
func BuildCategoryURL(slug string, f SearchFilters) string {
	base := "https://hotline.ua/ua/" + strings.Trim(slug, "/") + "/"
	v := url.Values{}
	if f.Page > 1 {
		v.Set("page", strconv.Itoa(f.Page))
	}
	if f.PriceMin > 0 {
		v.Set("priceMin", strconv.FormatFloat(f.PriceMin, 'f', 0, 64))
	}
	if f.PriceMax > 0 {
		v.Set("priceMax", strconv.FormatFloat(f.PriceMax, 'f', 0, 64))
	}
	if len(v) == 0 {
		return base
	}
	return base + "?" + v.Encode()
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
	products, _, err := parseSearchNUXT(nuxt, 0)
	return products, err
}

// ParseSearchPage extracts product summaries and pagination info from a
// catalog/search results page. currentPage should be the page number that was
// requested (1-based); pass 0 to default to page 1.
func ParseSearchPage(html []byte, currentPage int) ([]types.ProductSummary, types.PaginationInfo, error) {
	nuxt, err := nuxtState(html)
	if err != nil {
		return nil, types.PaginationInfo{}, fmt.Errorf("nuxt state: %w", err)
	}
	return parseSearchNUXT(nuxt, currentPage)
}

// parseSearchNUXT is the shared implementation used by both ParseSearchHTML
// and ParseSearchPage. currentPage is the 1-based page number of the request.
func parseSearchNUXT(nuxt map[string]any, currentPage int) ([]types.ProductSummary, types.PaginationInfo, error) {
	if currentPage <= 0 {
		currentPage = 1
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

	// Extract paginationInfo: {"lastPage": N, "totalCount": N, "itemsPerPage": N}
	pi := types.PaginationInfo{CurrentPage: currentPage}
	if info, ok := dig(nuxt, "state", "catalog", "products", "paginationInfo").(map[string]any); ok {
		pi.TotalItems = jsonInt(info["totalCount"])
		pi.TotalPages = jsonInt(info["lastPage"])
	}
	if pi.TotalPages == 0 {
		pi.TotalPages = 1
	}
	pi.HasNextPage = currentPage < pi.TotalPages
	if pi.HasNextPage {
		next := currentPage + 1
		pi.NextPage = &next
	}

	return out, pi, nil
}
