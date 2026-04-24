package scrapers_test

import (
	"strings"
	"testing"

	"github.com/kovalovme/hotline-ua-mcp/internal/scrapers"
	"github.com/kovalovme/hotline-ua-mcp/internal/types"
)

func TestParseSearchHTML(t *testing.T) {
	html := loadFixture(t, "search.html")
	results, err := scrapers.ParseSearchHTML(html)
	if err != nil {
		t.Fatalf("ParseSearchHTML error: %v", err)
	}

	if len(results) != 3 {
		t.Fatalf("result count: got %d, want 3", len(results))
	}

	p0 := results[0]
	if p0.ID != "26390745" {
		t.Errorf("results[0].ID: got %q, want %q", p0.ID, "26390745")
	}
	if p0.Title != "Apple iPhone 17 256GB Black (MG6J4)" {
		t.Errorf("results[0].Title: got %q", p0.Title)
	}
	if p0.URL != "https://hotline.ua/ua/mobile-mobilnye-telefony-i-smartfony/apple-iphone-17-256gb-black/" {
		t.Errorf("results[0].URL: got %q", p0.URL)
	}
	if p0.PriceMin != 39860 {
		t.Errorf("results[0].PriceMin: got %v, want 39860", p0.PriceMin)
	}
	if p0.PriceMax != 60990 {
		t.Errorf("results[0].PriceMax: got %v, want 60990", p0.PriceMax)
	}
	if p0.Currency != "UAH" {
		t.Errorf("results[0].Currency: got %q, want UAH", p0.Currency)
	}
	if p0.OffersCount != 81 {
		t.Errorf("results[0].OffersCount: got %d, want 81", p0.OffersCount)
	}
	if p0.ReviewCount != 6 {
		t.Errorf("results[0].ReviewCount: got %d, want 6", p0.ReviewCount)
	}
	if !strings.HasSuffix(p0.ImageURL, "5400153220.jpg") {
		t.Errorf("results[0].ImageURL: got %q, want suffix 5400153220.jpg", p0.ImageURL)
	}
}

func TestParseSearchHTML_Empty(t *testing.T) {
	html := []byte(`<!doctype html><html><body><script>window.__NUXT__={"state":{"catalog":{"products":{"collection":[],"paginationInfo":{}}}}};</script></body></html>`)
	results, err := scrapers.ParseSearchHTML(html)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}
}

func TestFilterByQuery(t *testing.T) {
	products := []types.ProductSummary{
		{Title: "Apple iPhone 17 256GB Black"},
		{Title: "Samsung Galaxy S25 128GB"},
		{Title: "Apple iPhone 15 Pro 512GB"},
	}

	got := scrapers.FilterByQuery(products, "iphone")
	if len(got) != 2 {
		t.Errorf("iphone: want 2, got %d", len(got))
	}

	got = scrapers.FilterByQuery(products, "Apple 256GB")
	if len(got) != 1 || got[0].Title != "Apple iPhone 17 256GB Black" {
		t.Errorf("Apple 256GB: want 1 specific result, got %v", got)
	}

	got = scrapers.FilterByQuery(products, "samsung")
	if len(got) != 1 || got[0].Title != "Samsung Galaxy S25 128GB" {
		t.Errorf("samsung: want 1 result, got %v", got)
	}

	got = scrapers.FilterByQuery(products, "")
	if len(got) != 3 {
		t.Errorf("empty query: want all 3, got %d", len(got))
	}

	got = scrapers.FilterByQuery(products, "pixel")
	if len(got) != 0 {
		t.Errorf("no-match: want 0, got %d", len(got))
	}
}

func TestBuildSearchURL(t *testing.T) {
	u := scrapers.BuildSearchURL("iphone 15")
	if !strings.HasPrefix(u, "https://hotline.ua/ua/") {
		t.Errorf("URL should start with hotline.ua/ua/: got %q", u)
	}
	if !strings.Contains(u, "iphone") && !strings.Contains(u, "15") {
		t.Errorf("URL should encode the query: got %q", u)
	}
}

func TestBuildSearchURLFiltered(t *testing.T) {
	f := scrapers.SearchFilters{Page: 3, PriceMin: 5000, PriceMax: 20000}
	u := scrapers.BuildSearchURLFiltered("samsung", f)
	if !strings.Contains(u, "page=3") {
		t.Errorf("should contain page=3: got %q", u)
	}
	if !strings.Contains(u, "priceMin=5000") {
		t.Errorf("should contain priceMin=5000: got %q", u)
	}
	if !strings.Contains(u, "priceMax=20000") {
		t.Errorf("should contain priceMax=20000: got %q", u)
	}
	if !strings.Contains(u, "q=samsung") {
		t.Errorf("should contain q=samsung (server-side filter): got %q", u)
	}
	if strings.Contains(u, "text=") {
		t.Errorf("must not use ?text= (server ignores it): got %q", u)
	}

	// page=1 should be omitted (default)
	u2 := scrapers.BuildSearchURLFiltered("test", scrapers.SearchFilters{Page: 1})
	if strings.Contains(u2, "page=") {
		t.Errorf("page=1 should be omitted: got %q", u2)
	}
}

func TestBuildCategoryURL(t *testing.T) {
	u := scrapers.BuildCategoryURL("mobile/mobilnye-telefony-i-smartfony", scrapers.SearchFilters{})
	if u != "https://hotline.ua/ua/mobile/mobilnye-telefony-i-smartfony/" {
		t.Errorf("bare category URL: got %q", u)
	}

	u2 := scrapers.BuildCategoryURL("mobile/mobilnye-telefony-i-smartfony", scrapers.SearchFilters{Page: 2, PriceMax: 15000})
	if !strings.Contains(u2, "page=2") {
		t.Errorf("should contain page=2: got %q", u2)
	}
	if !strings.Contains(u2, "priceMax=15000") {
		t.Errorf("should contain priceMax=15000: got %q", u2)
	}

	// Leading/trailing slashes in slug should be normalised
	u3 := scrapers.BuildCategoryURL("/mobile/mobilnye-telefony-i-smartfony/", scrapers.SearchFilters{})
	if u3 != "https://hotline.ua/ua/mobile/mobilnye-telefony-i-smartfony/" {
		t.Errorf("slug normalisation: got %q", u3)
	}
}

func TestParseSearchPage_Pagination(t *testing.T) {
	html := loadFixture(t, "search.html")
	products, pagination, err := scrapers.ParseSearchPage(html, 1)
	if err != nil {
		t.Fatalf("ParseSearchPage error: %v", err)
	}
	if len(products) != 3 {
		t.Fatalf("product count: got %d, want 3", len(products))
	}
	if pagination.TotalItems != 3 {
		t.Errorf("TotalItems: got %d, want 3", pagination.TotalItems)
	}
	if pagination.TotalPages != 1 {
		t.Errorf("TotalPages: got %d, want 1", pagination.TotalPages)
	}
	if pagination.CurrentPage != 1 {
		t.Errorf("CurrentPage: got %d, want 1", pagination.CurrentPage)
	}
	if pagination.HasNextPage {
		t.Error("HasNextPage should be false for single-page result")
	}
	if pagination.NextPage != nil {
		t.Errorf("NextPage should be nil, got %v", *pagination.NextPage)
	}
}

func TestParseSearchPage_DefaultCurrentPage(t *testing.T) {
	html := loadFixture(t, "search.html")
	// passing 0 should default to page 1
	_, pagination, err := scrapers.ParseSearchPage(html, 0)
	if err != nil {
		t.Fatalf("ParseSearchPage error: %v", err)
	}
	if pagination.CurrentPage != 1 {
		t.Errorf("CurrentPage: got %d, want 1", pagination.CurrentPage)
	}
}

func TestParseSearchPage_MultiPage(t *testing.T) {
	// Synthetic fixture: 100 total items, 24 per page, last page = 5
	html := []byte(`<!doctype html><html><body><script>window.__NUXT__={"state":{"catalog":{"products":{"collection":[],"paginationInfo":{"lastPage":5,"totalCount":100,"itemsPerPage":24}}}}};</script></body></html>`)
	_, pagination, err := scrapers.ParseSearchPage(html, 2)
	if err != nil {
		t.Fatalf("ParseSearchPage error: %v", err)
	}
	if pagination.TotalItems != 100 {
		t.Errorf("TotalItems: got %d, want 100", pagination.TotalItems)
	}
	if pagination.TotalPages != 5 {
		t.Errorf("TotalPages: got %d, want 5", pagination.TotalPages)
	}
	if pagination.CurrentPage != 2 {
		t.Errorf("CurrentPage: got %d, want 2", pagination.CurrentPage)
	}
	if !pagination.HasNextPage {
		t.Error("HasNextPage should be true")
	}
	if pagination.NextPage == nil || *pagination.NextPage != 3 {
		t.Errorf("NextPage: want 3, got %v", pagination.NextPage)
	}
}

func TestBuildSearchURLFiltered_UsesQParam(t *testing.T) {
	f := scrapers.SearchFilters{Page: 2, PriceMin: 5000, PriceMax: 20000}
	u := scrapers.BuildSearchURLFiltered("samsung", f)
	if strings.Contains(u, "text=") {
		t.Errorf("URL should not use ?text= param (server-side ignored): got %q", u)
	}
	if !strings.Contains(u, "q=samsung") {
		t.Errorf("URL should use ?q= for server-side filtering: got %q", u)
	}
}

func TestBuildCategorySearchURL(t *testing.T) {
	u := scrapers.BuildCategorySearchURL("/mobile/mobilnye-telefony-i-smartfony/", "iphone 17", scrapers.SearchFilters{})
	if u != "https://hotline.ua/ua/mobile/mobilnye-telefony-i-smartfony/?q=iphone+17" {
		t.Errorf("basic URL: got %q", u)
	}

	u2 := scrapers.BuildCategorySearchURL("/mobile/mobilnye-telefony-i-smartfony/", "samsung", scrapers.SearchFilters{Page: 2, PriceMax: 15000})
	if !strings.Contains(u2, "q=samsung") {
		t.Errorf("should contain q=samsung: got %q", u2)
	}
	if !strings.Contains(u2, "page=2") {
		t.Errorf("should contain page=2: got %q", u2)
	}
	if !strings.Contains(u2, "priceMax=15000") {
		t.Errorf("should contain priceMax=15000: got %q", u2)
	}
}

func TestParseSearchMenuResponse(t *testing.T) {
	body := []byte(`{"jsonrpc":"2.0","id":1,"result":{"products":{"sections":[{"sectionTitle":"Smartphones","catalogs":[{"id":"11","catalogTitle":"Phones","url":"/mobile/mobilnye-telefony-i-smartfony/","total":491},{"id":"599","catalogTitle":"Cases","url":"/mobile/chehly/","total":1271}]}]}}}`)
	path, err := scrapers.ParseSearchMenuResponse(body)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if path != "/mobile/mobilnye-telefony-i-smartfony/" {
		t.Errorf("category path: got %q, want /mobile/mobilnye-telefony-i-smartfony/", path)
	}
}

func TestParseSearchMenuResponse_Empty(t *testing.T) {
	body := []byte(`{"jsonrpc":"2.0","id":1,"result":{"products":{"sections":[]}}}`)
	_, err := scrapers.ParseSearchMenuResponse(body)
	if err == nil {
		t.Error("expected error for empty sections, got nil")
	}
}

func TestParseSearchMenuResponse_NoCatalogs(t *testing.T) {
	body := []byte(`{"jsonrpc":"2.0","id":1,"result":{"products":{"sections":[{"sectionTitle":"X","catalogs":[]}]}}}`)
	_, err := scrapers.ParseSearchMenuResponse(body)
	if err == nil {
		t.Error("expected error for section with no catalogs, got nil")
	}
}
