package scrapers

import (
	"bytes"
	"fmt"

	"github.com/PuerkitoBio/goquery"
	"github.com/kovalovme/hotline-ua-mcp/internal/types"
)

// ParseProductHTML extracts a full Product from a product detail page.
//
// TODO(fixture): finalize selectors once a real page is saved at
// test/fixtures/product.html. Expected hooks:
//
//	h1.title                       // title
//	.product-info__price-value     // aggregate price range
//	.product-info__reviews         // rating + review count
//	.specifications__row           // spec key/value rows
func ParseProductHTML(html []byte) (*types.Product, error) {
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(html))
	if err != nil {
		return nil, fmt.Errorf("parse html: %w", err)
	}
	_ = doc
	return nil, ErrNotImplemented
}
