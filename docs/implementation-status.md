# Implementation Status

**As of 2026-04-24 (updated after v0.2 blocker fixes).** Describes what is
actually built, how it works, and what is broken or missing. Complements the
planning roadmap (`docs/planning/roadmap.md`), which tracks future work.

---

## What is built

### MCP server

`cmd/hotline-ua-mcp/main.go` — stdio MCP server exposing three tools:

| Tool | Handler | Status |
|---|---|---|
| `search_products` | `internal/tools/search_products.go` | Runs; results not keyword-filtered (see §Bugs) |
| `get_product` | `internal/tools/get_product.go` | Runs; blocked by Cloudflare on live requests |
| `list_offers` | `internal/tools/list_offers.go` | Runs; blocked by Cloudflare on live requests |

### HTTP client (`internal/httpclient/client.go`)

- 1 req/sec global token-bucket rate limit (configurable via `HOTLINE_RATE_LIMIT_RPS`)
- In-memory LRU response cache: 256 entries, 10-minute TTL
- UA string rotation per request (Chrome 133 strings)
- Cookie jar (session-level, via `bogdanfinn/tls-client`)
- `Accept-Language: uk-UA,uk;q=0.9`
- **`bogdanfinn/tls-client` Chrome_133 JA3 profile** — mimics real Chrome TLS fingerprint to pass Cloudflare bot checks
- `ErrBotBlock` sentinel: returned on 503/403 responses whose body contains Cloudflare markers (`cloudflare`, `cf-ray`, `just a moment`, `enable javascript`, `challenge`)

### Scrapers (`internal/scrapers/`)

All three parsers are implemented and pass their fixture-driven tests.

#### `ParseProductHTML` (`product.go`)

Two-pass extraction:

1. **schema.org/Product JSON-LD** (`<script type="application/ld+json">` with
   `"@type": "Product"`) via `goquery`:
   - `ld["name"]` → `Title`
   - `ld["url"]` → `URL`
   - `ld["sku"]` → `ID` (numeric Hotline product ID, e.g. `"26737403"`)
   - `ld["image"][0]` → `ImageURL`
   - `ld["description"]` → `Description`
   - `ld["offers"]["lowPrice"]` / `["highPrice"]` → `PriceMin` / `PriceMax`
   - `ld["offers"]["priceCurrency"]` → `Currency`
   - `ld["offers"]["offerCount"]` → `OffersCount`
   - `ld["aggregateRating"]["ratingValue"]` → `Rating`
   - `ld["aggregateRating"]["reviewCount"]` → `ReviewCount`

2. **`window.__NUXT__` state** evaluated with `github.com/dop251/goja`:
   - Path: `state.product.productValues.edges[].node`
   - Skips `isHeader=true` nodes and nodes with `title` in `{"vendor","series"}`
   - Remaining `{title, value}` pairs → `Specs` map

If JSON-LD is absent the function returns an error. If `__NUXT__` is absent
specs are silently omitted (product still returned).

#### `ParseOffersHTML` (`offers.go`)

Single-pass extraction from `window.__NUXT__`:

- Path: `state.product.offers.edges[].node`
- Fields per node:

| `__NUXT__` field | Offer field |
|---|---|
| `firmTitle` | `ShopName` |
| `firmExtraInfo.website` | `ShopURL` (prefixed `https://`) |
| `conversionUrl` | `OfferURL` (prefixed `https://hotline.ua`) |
| `price` | `Price` |
| hardcoded | `Currency = "UAH"` |
| `visible` | `InStock` |
| `condition` | `Condition` |
| `guaranteeTerm` + `guaranteeType` | `Guarantee` (e.g. `"12 міс."`, `"1 міс. от магазина"`) |

Source order is preserved. Sorting by price is the responsibility of the tool
layer (`list_offers`).

`ParseOffersJSON` is a stub that returns `ErrNotImplemented`.

#### `ParseSearchHTML` (`search.go`)

Single-pass extraction from `window.__NUXT__`:

- Path: `state.catalog.products.collection[]`
- Fields per item:

| `__NUXT__` field | ProductSummary field |
|---|---|
| `_id` | `ID` |
| `vendor.title` + `title` | `Title` (vendor prepended if not already present) |
| `url` | `URL` (prefixed `https://hotline.ua/ua`) |
| `imageLinks[0].thumb` | `ImageURL` (prefixed `https://hotline.ua`) |
| `minPrice` / `maxPrice` | `PriceMin` / `PriceMax` |
| hardcoded | `Currency = "UAH"` |
| `offerCount` | `OffersCount` |
| `reviewsCount` | `ReviewCount` |

`BuildSearchURL` generates the URL used by `search_products`:
`https://hotline.ua/ua/mobile/mobilnye-telefony-i-smartfony/?text=<query>`

### Core extraction helpers (`internal/scrapers/extract.go`)

| Symbol | Purpose |
|---|---|
| `ErrNotImplemented` | Sentinel for unimplemented stub parsers |
| `extractProductJSONLD` | goquery → first `script[type="application/ld+json"]` with `@type=Product` |
| `nuxtState` | regex captures `window.__NUXT__=…`, wraps in IIFE, evaluates with goja, returns `map[string]any` |
| `dig` / `digSlice` | Safe nested `map[string]any` traversal |
| `jsonFloat64` / `jsonInt` / `jsonString` | Safe type coercions from `any` |

### Test fixtures (`test/fixtures/`)

| File | Contents |
|---|---|
| `product.html` | Minimal HTML with real schema.org/Product JSON-LD + compact `window.__NUXT__` containing 3 offers and 5 spec nodes |
| `search.html` | Minimal HTML with compact `window.__NUXT__` containing a catalog state with 3 products |

The fixtures are handcrafted to match the shape of real hotline.ua pages but
are minimal: only the fields the parsers actually read are present. They are
not captures of live pages.

### Keyword filtering (`scrapers.FilterByQuery`)

Called by `search_products` after `ParseSearchHTML`. All words in the query
must appear case-insensitively in the product title. This is a client-side
workaround for the SSR `?text=` limitation; real server-side search via the
GraphQL endpoint is deferred to v0.3.

### Tests

All 11 tests pass (`go test ./...`):

| Test | File |
|---|---|
| `TestParseProductHTML` | `product_test.go` |
| `TestParseProductHTML_NoJSONLD` | `product_test.go` |
| `TestParseOffersHTML` | `offers_test.go` |
| `TestParseOffersHTML_MissingFirmExtraInfo` | `offers_test.go` |
| `TestParseOffersHTML_NoOffers` | `offers_test.go` |
| `TestParseSearchHTML` | `search_test.go` |
| `TestParseSearchHTML_Empty` | `search_test.go` |
| `TestFilterByQuery` | `search_test.go` |
| `TestBuildSearchURL` | `search_test.go` |
| `TestIsBotBlock` | `httpclient/client_test.go` |
| `TestErrBotBlockSentinel` | `httpclient/client_test.go` |

---

## Resolved bugs (fixed for v0.2)

### 1. Panic in `ParseOffersHTML` on missing `firmExtraInfo` — FIXED

`offers.go:48` — replaced unsafe type assertion with `dig(node, "firmExtraInfo", "website")`.
`TestParseOffersHTML_MissingFirmExtraInfo` verifies the fix.

### 2. `search_products` keyword filtering — FIXED (client-side)

`scrapers.FilterByQuery` added: all words in the query must appear
case-insensitively in the product title. Called by the tool after
`ParseSearchHTML`. The SSR `?text=` limitation remains; real server-side
filtering via GraphQL is deferred to v0.3. `TestFilterByQuery` covers the
behaviour.

### 3. Cloudflare TLS blocking — FIXED

`client.go` rewritten to use `bogdanfinn/tls-client` Chrome_133 profile.
`ErrBotBlock` sentinel added; detection covers 503/403 + body markers.
`TestIsBotBlock` and `TestErrBotBlockSentinel` cover the logic.

---

## Minor issues

| Issue | Location | Impact |
|---|---|---|
| `go 1.25.0` in `go.mod` declares a future Go version | `go.mod:3` | Build fails on any Go toolchain older than 1.25 (which does not exist yet); harmless until Go 1.25 ships but confusing |
| `search_products` is hardcoded to the smartphones category | `search.go:24` | Cannot search other product categories |
| No live fixtures | `test/fixtures/` | Tests pass on handcrafted fixtures; markup drift will not be caught until live fixtures are captured and wired |
