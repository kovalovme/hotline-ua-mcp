package scrapers

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"

	"github.com/PuerkitoBio/goquery"
	"github.com/dop251/goja"
)

// ErrNotImplemented is returned by parser stubs that have not yet been wired
// to real selectors or endpoints.
var ErrNotImplemented = errors.New("scraper not yet wired to real selectors; drop a fixture in test/fixtures and finish the TODO")

var reNUXTScript = regexp.MustCompile(`(?s)window\.__NUXT__\s*=\s*(.+?)\s*;?\s*</script>`)

// extractProductJSONLD finds the first schema.org/Product JSON-LD block in html
// and returns its decoded fields as a plain map.
func extractProductJSONLD(html []byte) (map[string]any, error) {
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(html))
	if err != nil {
		return nil, fmt.Errorf("parse html: %w", err)
	}

	var result map[string]any
	var parseErr error

	doc.Find(`script[type="application/ld+json"]`).EachWithBreak(func(_ int, s *goquery.Selection) bool {
		var m map[string]any
		if err := json.Unmarshal([]byte(s.Text()), &m); err != nil {
			return true // skip malformed blocks
		}
		if m["@type"] == "Product" {
			result = m
			return false // stop iteration
		}
		return true
	})

	if result == nil {
		parseErr = errors.New("no schema.org Product JSON-LD found in page")
	}
	return result, parseErr
}

// nuxtState evaluates the window.__NUXT__ assignment found in html using the
// goja JS engine and returns the decoded state object.
func nuxtState(html []byte) (map[string]any, error) {
	m := reNUXTScript.FindSubmatch(html)
	if m == nil {
		return nil, errors.New("window.__NUXT__ not found in page")
	}
	script := string(m[1])

	vm := goja.New()
	// Provide a minimal window object so the script can write window.__NUXT__
	// but also handle the plain-assignment form used in fixtures.
	win := vm.NewObject()
	if err := vm.Set("window", win); err != nil {
		return nil, fmt.Errorf("goja setup: %w", err)
	}

	// Wrap in a try/catch so Date constructor and other globals don't crash us.
	wrapped := fmt.Sprintf(`(function(){window.__NUXT__=%s;})()`, script)
	if _, err := vm.RunString(wrapped); err != nil {
		return nil, fmt.Errorf("eval __NUXT__: %w", err)
	}

	nuxt := win.Get("__NUXT__")
	if nuxt == nil || goja.IsUndefined(nuxt) {
		return nil, errors.New("window.__NUXT__ evaluated to undefined")
	}

	// Marshal through JSON to get a plain Go map.
	b, err := json.Marshal(nuxt.Export())
	if err != nil {
		return nil, fmt.Errorf("marshal nuxt: %w", err)
	}
	var out map[string]any
	if err := json.Unmarshal(b, &out); err != nil {
		return nil, fmt.Errorf("unmarshal nuxt: %w", err)
	}
	return out, nil
}

// jsonFloat64 safely reads a float64 from a map value (handles float64 and
// JSON numbers stored as json.Number).
func jsonFloat64(v any) float64 {
	switch n := v.(type) {
	case float64:
		return n
	case json.Number:
		f, _ := n.Float64()
		return f
	}
	return 0
}

// jsonInt safely reads an int from a map value.
func jsonInt(v any) int {
	return int(jsonFloat64(v))
}

// jsonString safely reads a string from a map value.
func jsonString(v any) string {
	s, _ := v.(string)
	return s
}

// dig traverses a nested map[string]any by successive keys, returning the
// final value or nil if any step is missing.
func dig(m map[string]any, keys ...string) any {
	var cur any = m
	for _, k := range keys {
		mm, ok := cur.(map[string]any)
		if !ok {
			return nil
		}
		cur = mm[k]
	}
	return cur
}

// digSlice is like dig but asserts the final value is a []any.
func digSlice(m map[string]any, keys ...string) []any {
	v := dig(m, keys...)
	s, _ := v.([]any)
	return s
}
