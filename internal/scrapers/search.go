// Package scrapers turns hotline.ua HTML/JSON responses into typed values.
//
// UA-locale only. No live endpoint is reachable from the sandbox this code was
// scaffolded in, so selectors are placeholders validated against saved
// fixtures in test/fixtures/. Before shipping, capture real pages and finish
// the TODO(fixture) sections.
package scrapers

import (
	"bytes"
	"errors"
	"fmt"
	"net/url"

	"github.com/PuerkitoBio/goquery"
	"github.com/kovalovme/hotline-ua-mcp/internal/types"
)

var ErrNotImplemented = errors.New("scraper not yet wired to real selectors; drop a fixture in test/fixtures and finish the TODO")

// BuildSearchURL constructs the UA-locale search URL.
func BuildSearchURL(query string) string {
	v := url.Values{}
	v.Set("q", query)
	return fmt.Sprintf("https://hotline.ua/ua/search/?%s", v.Encode())
}

// ParseSearchHTML extracts product summaries from a search results page.
//
// TODO(fixture): with a real saved page, finalize the selectors. Expected
// anchors per product card (subject to change as hotline updates markup):
//
//	.list-item                 // card container
//	.list-item__title a        // title + href
//	.list-item__img img        // image src
//	.list-item__price-value    // price range text
//	.list-item__reviews-count  // review count
func ParseSearchHTML(html []byte) ([]types.ProductSummary, error) {
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(html))
	if err != nil {
		return nil, fmt.Errorf("parse html: %w", err)
	}

	var results []types.ProductSummary
	doc.Find(".list-item").Each(func(_ int, s *goquery.Selection) {
		_ = s // TODO(fixture): populate ProductSummary from s
	})

	if len(results) == 0 {
		return nil, ErrNotImplemented
	}
	return results, nil
}
