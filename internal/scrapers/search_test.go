package scrapers_test

import (
	"strings"
	"testing"

	"github.com/kovalovme/hotline-ua-mcp/internal/scrapers"
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

func TestBuildSearchURL(t *testing.T) {
	u := scrapers.BuildSearchURL("iphone 15")
	if !strings.HasPrefix(u, "https://hotline.ua/ua/") {
		t.Errorf("URL should start with hotline.ua/ua/: got %q", u)
	}
	if !strings.Contains(u, "iphone") && !strings.Contains(u, "15") {
		t.Errorf("URL should encode the query: got %q", u)
	}
}
