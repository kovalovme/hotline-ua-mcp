package scrapers

import (
	"fmt"

	"github.com/kovalovme/hotline-ua-mcp/internal/types"
)

// ParseOffersHTML extracts the seller offer list from a product page.
//
// Source: window.__NUXT__ → state.product.offers.edges
//
// Each offer node contains price, shop name, condition, guarantee, and
// visibility (in-stock indicator).
func ParseOffersHTML(html []byte) ([]types.Offer, error) {
	nuxt, err := nuxtState(html)
	if err != nil {
		return nil, fmt.Errorf("nuxt state: %w", err)
	}

	edges := digSlice(nuxt, "state", "product", "offers", "edges")
	out := make([]types.Offer, 0, len(edges))

	for _, e := range edges {
		em, ok := e.(map[string]any)
		if !ok {
			continue
		}
		node, ok := em["node"].(map[string]any)
		if !ok {
			continue
		}

		visible, _ := node["visible"].(bool)

		convURL := jsonString(node["conversionUrl"])
		if convURL != "" {
			convURL = "https://hotline.ua" + convURL
		}

		guarantee := guaranteeText(
			jsonInt(node["guaranteeTerm"]),
			jsonString(node["guaranteeType"]),
		)

		out = append(out, types.Offer{
			ShopName:  jsonString(node["firmTitle"]),
			ShopURL:   shopURL(jsonString(dig(node, "firmExtraInfo", "website"))),
			OfferURL:  convURL,
			Price:     jsonFloat64(node["price"]),
			Currency:  "UAH",
			InStock:   visible,
			Condition: jsonString(node["condition"]),
			Guarantee: guarantee,
		})
	}

	return out, nil
}

func guaranteeText(months int, gtype string) string {
	if months == 0 {
		return ""
	}
	base := fmt.Sprintf("%d міс.", months)
	if gtype != "" {
		return base + " " + gtype
	}
	return base
}

func shopURL(website string) string {
	if website == "" {
		return ""
	}
	return "https://" + website
}

// ParseOffersJSON is retained as a stub. The JSON XHR path was not confirmed
// during recon; the __NUXT__ HTML path is the confirmed primary route.
func ParseOffersJSON(body []byte) ([]types.Offer, error) {
	return nil, ErrNotImplemented
}
