package scrapers_test

import (
	"testing"

	"github.com/kovalovme/hotline-ua-mcp/internal/scrapers"
	"github.com/kovalovme/hotline-ua-mcp/internal/types"
)

func TestParseOffersHTML(t *testing.T) {
	html := loadFixture(t, "product.html")
	offers, err := scrapers.ParseOffersHTML(html)
	if err != nil {
		t.Fatalf("ParseOffersHTML error: %v", err)
	}

	if len(offers) != 3 {
		t.Fatalf("offer count: got %d, want 3", len(offers))
	}

	// Find each expected offer by shop name (scraper preserves source order;
	// sorting is the responsibility of the tool layer).
	byShop := make(map[string]interface{})
	for _, o := range offers {
		byShop[o.ShopName] = o
	}
	for _, name := range []string{"Техно Їжак", "Touch", "Just Buy"} {
		if _, ok := byShop[name]; !ok {
			t.Errorf("expected offer from shop %q not found", name)
		}
	}

	// Spot-check Техно Їжак
	techno := offers[0]
	if techno.Price != 34999 {
		t.Errorf("Техно Їжак price: got %v, want 34999", techno.Price)
	}
	if techno.Currency != "UAH" {
		t.Errorf("Техно Їжак currency: got %q, want UAH", techno.Currency)
	}
	if techno.OfferURL != "https://hotline.ua/go/price/14077373589/" {
		t.Errorf("Техно Їжак OfferURL: got %q", techno.OfferURL)
	}
	if !techno.InStock {
		t.Error("Техно Їжак InStock should be true (visible=true)")
	}
	if techno.Condition != "новый" {
		t.Errorf("Техно Їжак Condition: got %q", techno.Condition)
	}
	if techno.Guarantee != "12 міс." {
		t.Errorf("Техно Їжак Guarantee: got %q, want %q", techno.Guarantee, "12 міс.")
	}

	// Spot-check Just Buy (guaranteeType "от магазина", 1 month)
	var justBuy *types.Offer
	for i := range offers {
		if offers[i].ShopName == "Just Buy" {
			justBuy = &offers[i]
			break
		}
	}
	if justBuy == nil {
		t.Fatal("Just Buy offer not found")
	}
	if justBuy.Price != 32939 {
		t.Errorf("Just Buy price: got %v, want 32939", justBuy.Price)
	}
	if justBuy.Guarantee != "1 міс. от магазина" {
		t.Errorf("Just Buy Guarantee: got %q, want %q", justBuy.Guarantee, "1 міс. от магазина")
	}
}

func TestParseOffersHTML_NoOffers(t *testing.T) {
	html := []byte(`<!doctype html><html><body><script>window.__NUXT__={"state":{"product":{"offers":{"edges":[]}}}};</script></body></html>`)
	offers, err := scrapers.ParseOffersHTML(html)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(offers) != 0 {
		t.Errorf("expected 0 offers, got %d", len(offers))
	}
}
