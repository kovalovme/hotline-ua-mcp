package scrapers_test

import (
	"os"
	"testing"

	"github.com/kovalovme/hotline-ua-mcp/internal/scrapers"
)

func loadFixture(t *testing.T, name string) []byte {
	t.Helper()
	b, err := os.ReadFile("../../test/fixtures/" + name)
	if err != nil {
		t.Fatalf("load fixture %s: %v", name, err)
	}
	return b
}

func TestParseProductHTML(t *testing.T) {
	html := loadFixture(t, "product.html")
	p, err := scrapers.ParseProductHTML(html)
	if err != nil {
		t.Fatalf("ParseProductHTML error: %v", err)
	}

	if p.ID != "26737403" {
		t.Errorf("ID: got %q, want %q", p.ID, "26737403")
	}
	if p.Title != "Apple iPhone 17e 256GB Black (MHRV4)" {
		t.Errorf("Title: got %q", p.Title)
	}
	if p.URL != "https://hotline.ua/ua/mobile-mobilnye-telefony-i-smartfony/apple-iphone-17e-256gb-black/" {
		t.Errorf("URL: got %q", p.URL)
	}
	if p.PriceMin != 31850 {
		t.Errorf("PriceMin: got %v, want 31850", p.PriceMin)
	}
	if p.PriceMax != 37510 {
		t.Errorf("PriceMax: got %v, want 37510", p.PriceMax)
	}
	if p.Currency != "UAH" {
		t.Errorf("Currency: got %q, want UAH", p.Currency)
	}
	if p.OffersCount != 46 {
		t.Errorf("OffersCount: got %d, want 46", p.OffersCount)
	}
	if p.Rating != 4.8 {
		t.Errorf("Rating: got %v, want 4.8", p.Rating)
	}
	if p.ReviewCount != 3 {
		t.Errorf("ReviewCount: got %d, want 3", p.ReviewCount)
	}
	if p.ImageURL != "https://hotline.ua/img/tx/562/5627713055.jpg" {
		t.Errorf("ImageURL: got %q", p.ImageURL)
	}
	if p.Description == "" {
		t.Error("Description should not be empty")
	}

	// Specs come from __NUXT__ productValues (non-header nodes with non-empty value)
	if len(p.Specs) == 0 {
		t.Error("Specs should not be empty")
	}
	if p.Specs["Попередньо встановлена ОС"] != "iOS 26" {
		t.Errorf("Spec OS: got %q, want %q", p.Specs["Попередньо встановлена ОС"], "iOS 26")
	}
	if p.Specs["Оперативна пам'ять"] != "8 ГБ" {
		t.Errorf("Spec RAM: got %q, want %q", p.Specs["Оперативна пам'ять"], "8 ГБ")
	}
}

func TestParseProductHTML_NoJSONLD(t *testing.T) {
	html := []byte(`<!doctype html><html><head></head><body><script>window.__NUXT__={};</script></body></html>`)
	_, err := scrapers.ParseProductHTML(html)
	if err == nil {
		t.Error("expected error for missing JSON-LD, got nil")
	}
}

func TestParseProductHTML_CurrencyFallbackUAH(t *testing.T) {
	// JSON-LD with no priceCurrency — Currency must default to UAH.
	html := []byte(`<!doctype html><html><head>
<script type="application/ld+json">{"@type":"Product","name":"Test","url":"https://hotline.ua/ua/test/","sku":"123","offers":{"lowPrice":1000,"highPrice":2000,"offerCount":5}}</script>
</head><body><script>window.__NUXT__={};</script></body></html>`)
	p, err := scrapers.ParseProductHTML(html)
	if err != nil {
		t.Fatalf("ParseProductHTML error: %v", err)
	}
	if p.Currency != "UAH" {
		t.Errorf("Currency: got %q, want UAH (should default when priceCurrency absent)", p.Currency)
	}
}
