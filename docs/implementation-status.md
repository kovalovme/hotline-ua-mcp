# Implementation Status

**As of 2026-04-24 (v1.1).** Describes what is actually built, how it works,
and what is broken or missing. Complements the planning roadmap
(`docs/planning/roadmap.md`), which tracks future work.

---

## What is built

### MCP server

`cmd/hotline-ua-mcp/main.go` — stdio MCP server exposing four tools:

| Tool | Handler | Status |
|---|---|---|
| `search_products` | `internal/tools/search_products.go` | Functional; cross-category server-side search via `search.menu` + `?q=`; pagination and price filters |
| `get_product` | `internal/tools/get_product.go` | Functional; Chrome_133 JA3 profile bypasses Cloudflare |
| `list_offers` | `internal/tools/list_offers.go` | Functional; Chrome_133 JA3 profile bypasses Cloudflare |
| `list_category` | `internal/tools/list_category.go` | Functional; browse any category by slug with pagination and price filters |

### HTTP client (`internal/httpclient/client.go`)

- 1 req/sec global token-bucket rate limit (configurable via `HOTLINE_RATE_LIMIT_RPS`)
- In-memory LRU response cache: 256 entries, 10-minute TTL (GET only)
- UA string rotation per request (Chrome 133 strings)
- Cookie jar (session-level, via `bogdanfinn/tls-client`)
- `Accept-Language: uk-UA,uk;q=0.9`
- **`bogdanfinn/tls-client` Chrome_133 JA3 profile** — mimics real Chrome TLS fingerprint to pass Cloudflare bot checks
- `ErrBotBlock` sentinel: returned on 503/403 responses whose body contains Cloudflare markers
- **`PostJSON`** — rate-limited POST for JSON-RPC API calls (no cache)

### Scrapers (`internal/scrapers/`)

All parsers are implemented and pass their fixture-driven tests (23 total).

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
   - `ld["offers"]["priceCurrency"]` → `Currency` (falls back to `"UAH"` if absent)
   - `ld["offers"]["offerCount"]` → `OffersCount`
   - `ld["aggregateRating"]["ratingValue"]` → `Rating`
   - `ld["aggregateRating"]["reviewCount"]` → `ReviewCount`

2. **`window.__NUXT__` state** evaluated with `github.com/dop251/goja`:
   - Path: `state.product.productValues.edges[].node`
   - Skips `isHeader=true` nodes and nodes with `title` in `{"vendor","series"}`
   - Remaining `{title, value}` pairs → `Specs` map

#### `ParseOffersHTML` (`offers.go`)

Single-pass extraction from `window.__NUXT__`:

- Path: `state.product.offers.edges[].node`
- Fields: `ShopName`, `ShopURL`, `OfferURL`, `Price`, `Currency="UAH"`, `InStock`, `Condition`, `Guarantee`

#### `ParseSearchHTML` / `ParseSearchPage` (`search.go`)

Single-pass extraction from `window.__NUXT__`:

- Path: `state.catalog.products.collection[]`
- Fields per item: `ID` (from `_id`), `Title` (vendor + model), `URL`, `ImageURL`, `PriceMin`, `PriceMax`, `Currency="UAH"`, `OffersCount`, `ReviewCount`
- `ParseSearchPage` additionally extracts `PaginationInfo` from `state.catalog.products.paginationInfo` (`lastPage`, `totalCount`, `itemsPerPage`)

#### `ParseSearchMenuResponse` (`search.go`)

Parses `POST /svc/search/api/json-rpc` (`search.menu`) JSON response:

- Returns the `url` path of the first catalog in the first section
- Used by `search_products` to discover which category best matches a query

### URL builders (`internal/scrapers/search.go`)

| Function | Purpose |
|---|---|
| `BuildCategorySearchURL(path, query, filters)` | Category page with `?q=query` for server-side SSR filtering + optional `?page=N&priceMin=N&priceMax=M` |
| `BuildCategoryURL(slug, filters)` | Category browse without keyword (used by `list_category`) |
| `BuildSearchURL(query)` | Convenience wrapper → smartphones category with `?q=query` |
| `BuildSearchURLFiltered(query, filters)` | Same with pagination/price filters |

### `search_products` tool flow (v1.1)

1. `POST /svc/search/api/json-rpc` `search.menu` → discover best category path for query
2. `ParseSearchMenuResponse` → category path (e.g. `/mobile/mobilnye-telefony-i-smartfony/`)
3. `BuildCategorySearchURL(path, query, filters)` → URL with `?q=query`
4. `GET` that URL → SSR HTML with server-filtered `__NUXT__` catalog state
5. `ParseSearchPage` → products + pagination

**`FilterByQuery` is no longer called** — the server filters results via `?q=`.

### Test fixtures (`test/fixtures/`)

| File | Contents |
|---|---|
| `product.html` | Minimal HTML with real schema.org/Product JSON-LD + compact `window.__NUXT__` containing 3 offers and 5 spec nodes |
| `search.html` | Minimal HTML with compact `window.__NUXT__` containing a catalog state with 3 products and `paginationInfo` |

### Tests

All 23 tests pass (`go test ./...`):

| Test | File |
|---|---|
| `TestParseProductHTML` | `product_test.go` |
| `TestParseProductHTML_NoJSONLD` | `product_test.go` |
| `TestParseProductHTML_CurrencyFallbackUAH` | `product_test.go` |
| `TestParseOffersHTML` | `offers_test.go` |
| `TestParseOffersHTML_MissingFirmExtraInfo` | `offers_test.go` |
| `TestParseOffersHTML_NoOffers` | `offers_test.go` |
| `TestParseSearchHTML` | `search_test.go` |
| `TestParseSearchHTML_Empty` | `search_test.go` |
| `TestFilterByQuery` | `search_test.go` |
| `TestBuildSearchURL` | `search_test.go` |
| `TestBuildSearchURLFiltered` | `search_test.go` |
| `TestBuildSearchURLFiltered_UsesQParam` | `search_test.go` |
| `TestBuildCategoryURL` | `search_test.go` |
| `TestBuildCategorySearchURL` | `search_test.go` |
| `TestParseSearchPage_Pagination` | `search_test.go` |
| `TestParseSearchPage_DefaultCurrentPage` | `search_test.go` |
| `TestParseSearchPage_MultiPage` | `search_test.go` |
| `TestParseSearchMenuResponse` | `search_test.go` |
| `TestParseSearchMenuResponse_Empty` | `search_test.go` |
| `TestParseSearchMenuResponse_NoCatalogs` | `search_test.go` |
| `TestIsBotBlock` | `httpclient/client_test.go` |
| `TestErrBotBlockSentinel` | `httpclient/client_test.go` |

---

## Resolved bugs

### 1. Panic in `ParseOffersHTML` on missing `firmExtraInfo` — FIXED (v0.2)

`offers.go` — replaced unsafe type assertion with `dig(node, "firmExtraInfo", "website")`.

### 2. `search_products` keyword filtering — FIXED (v1.1)

Replaced client-side `FilterByQuery` workaround with real server-side filtering:
`POST /svc/search/api/json-rpc` (`search.menu`) discovers the best category,
then `?q=<query>` on the category page filters SSR `__NUXT__` catalog state.

### 3. Cloudflare TLS blocking — FIXED (v0.2)

`client.go` rewritten to use `bogdanfinn/tls-client` Chrome_133 profile.

---

## Minor issues

| Issue | Location | Impact |
|---|---|---|
| `go 1.25.0` in `go.mod` declares a future Go version | `go.mod:3` | Harmless until Go 1.25 ships |
| `search_products` always picks the first catalog in the first `search.menu` section | `tools/search_products.go` | For "iphone 17", returns phones (491 results) not cases (1271). Works correctly but could be smarter. |
| No live fixtures | `test/fixtures/` | Tests pass on handcrafted fixtures; markup drift not caught automatically |
