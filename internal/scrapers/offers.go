package scrapers

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/PuerkitoBio/goquery"
	"github.com/kovalovme/hotline-ua-mcp/internal/types"
)

// ParseOffersJSON tries to decode the offers list from an internal JSON
// endpoint (preferred path: faster, richer data). The exact schema is
// unknown without live access; this is a typed guess that will need
// adjustment once a real response is captured.
//
// TODO(fixture): capture the real XHR (DevTools → Network tab when opening
// the "Где купить" tab on a product page), save JSON to
// test/fixtures/offers.json, then finalize this mapping.
func ParseOffersJSON(body []byte) ([]types.Offer, error) {
	var raw struct {
		Offers []struct {
			ShopName  string  `json:"shop_name"`
			ShopURL   string  `json:"shop_url"`
			URL       string  `json:"url"`
			Price     float64 `json:"price"`
			Currency  string  `json:"currency"`
			InStock   bool    `json:"in_stock"`
			Condition string  `json:"condition"`
			Guarantee string  `json:"guarantee"`
		} `json:"offers"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("decode offers json: %w", err)
	}

	out := make([]types.Offer, 0, len(raw.Offers))
	for _, o := range raw.Offers {
		out = append(out, types.Offer{
			ShopName:  o.ShopName,
			ShopURL:   o.ShopURL,
			OfferURL:  o.URL,
			Price:     o.Price,
			Currency:  o.Currency,
			InStock:   o.InStock,
			Condition: o.Condition,
			Guarantee: o.Guarantee,
		})
	}
	if len(out) == 0 {
		return nil, ErrNotImplemented
	}
	return out, nil
}

// ParseOffersHTML is the fallback path: scrape the offers tab HTML.
//
// TODO(fixture): save test/fixtures/offers.html and finalize selectors.
// Expected hooks:
//
//	.price-item                  // offer row
//	.price-item__shop-name       // shop name
//	.price-item__price           // price
//	.price-item__go a            // outbound offer link
func ParseOffersHTML(html []byte) ([]types.Offer, error) {
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(html))
	if err != nil {
		return nil, fmt.Errorf("parse html: %w", err)
	}
	_ = doc
	return nil, ErrNotImplemented
}
