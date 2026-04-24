# Hotline.ua Integration Roadmap

**Status:** v1.1 released. Four tools functional, 23 tests passing. Server-side
cross-category search via `search.menu` JSON-RPC + `?q=` SSR filtering confirmed
via live DevTools recon on 2026-04-24.

See `docs/implementation-status.md` for a detailed description of what is
actually built vs. planned.

## 1. Current state (v1)

- MCP stdio server in Go, three tools registered: `search_products`,
  `get_product`, `list_offers`.
- HTTP client: `bogdanfinn/tls-client` Chrome_133 JA3 profile, UA rotation,
  global token-bucket rate limit, LRU response cache, cookie jar.
- `ErrBotBlock` sentinel returned on Cloudflare 503/403 intercepts.
- Claude Code plugin manifests + `.mcp.json` + marketplace entry.
- GoReleaser pipeline triggered on `v*` tags (linux/darwin/windows, amd64/arm64).
- All three scrapers implemented and passing fixture-driven tests (11 total):
  - `ParseProductHTML` — schema.org/Product JSON-LD + `window.__NUXT__` specs.
  - `ParseOffersHTML` — `window.__NUXT__` `state.product.offers.edges`.
  - `ParseSearchHTML` + `FilterByQuery` — `window.__NUXT__` catalog state with
    client-side keyword filtering.
  - `window.__NUXT__` IIFE evaluated in-process via `github.com/dop251/goja`.

## 2. Data sources on hotline.ua

Hotline exposes three surfaces. The plugin prioritises them in this order:

1. **Server-rendered HTML (Nuxt.js SSR).** Live recon (2026-04-24) confirmed
   that hotline.ua runs Nuxt.js with server-side rendering. Product pages,
   search results, and — critically — the full offers list all arrive in the
   initial HTML response. No secondary XHR is required to obtain price data.
   This is the primary (and sufficient) scraping target.
2. **Internal JSON / XHR endpoints.** The site loads some dynamic content
   (filters, lazy sections) via XHR. Suspected endpoints were not confirmed
   during recon because direct HTTP fetches were blocked by Cloudflare (503).
   If discovered via DevTools, JSON endpoints should be preferred where stable.
   Example suspected paths (unconfirmed):
   - `/svc/frontend-api/shop-prices/...` for offers
   - `/svc/frontend-api/search/...` for search
3. **Public Hotline APIs.** Only one exists
   ([`/ua/about/api_auctions/`](https://hotline.ua/ua/about/api_auctions/)) and
   it is merchant-side: manage auction bids with an `auth_token`, 5 req/sec /
   300 req/min. **Not usable** for product search. Response schema is documented
   (see §6 for the `product_id` and `firm_offer_id` fields it exposes).
4. **Merchant XML feeds.** Outbound from merchants into Hotline, not a
   consumer-facing read surface. Ignore.

## 3. JSON-first discovery playbook

Before writing any HTML scraper, always check if a stable JSON endpoint exists.

1. Open the target page in Chrome with DevTools → Network → Fetch/XHR.
2. Repro the user action (load product, open offers tab, change filter, click
   page 2).
3. Look for JSON responses whose schema matches the rendered UI.
4. Note: request method, required cookies/headers (often `Referer`,
   `X-Requested-With`), URL template, response shape.
5. Save a representative response to `test/fixtures/<name>.json` and wire a
   parser test.

**Update 2026-04-24 (HTML curl recon, supersedes WebFetch findings):** Product
pages contain:

1. **schema.org/Product JSON-LD** (`<script type="application/ld+json">`) — fully
   populated: name, SKU (numeric product ID), URL, price range, offer count,
   rating, review count, image URLs, description. This is the primary source
   for `ParseProductHTML`.
2. **`window.__NUXT__` packed IIFE** (144 KB on test product) — contains
   `state.product.offers.edges` (all 46 offers with prices, shop names, guarantee
   terms, shipping) and `state.product.productValues.edges` (specs). Primary
   source for `ParseOffersHTML` and specs in `ParseProductHTML`. Also contains
   `state.catalog.products.collection` on catalog/search pages (used by
   `ParseSearchHTML`).

The `__NUXT__` IIFE is evaluated with `github.com/dop251/goja` at parse time.
Individual offer prices are **not** in the rendered HTML — they only exist in
the `__NUXT__` state.

**Search limitation:** the global search URL (`/ua/search/?q=…`) is client-side
routed and redirects to 404 under direct HTTP. The SSR search endpoint is
internal (`search.search-19-production/api/json-rpc`). `search_products` is
implemented against catalog listing pages (`/ua/mobile/mobilnye-telefony-i-smartfony/?text=…`)
which do include product listings in SSR `__NUXT__` state.

**Rule of thumb:** if a JSON endpoint exists and responds with 200 using the
same cookies as page navigation, prefer it. Fall back to HTML scraping only
when JSON is cumbersome (requires CSRF token refresh, mixed content, etc.).

## 4. HTML scraping strategy

Used when JSON isn't available or is more fragile.

### 4.1 Library and test harness

- **Library:** `github.com/PuerkitoBio/goquery`. Selectors live in
  `internal/scrapers/*.go`, one file per page type.
- **Fixture-driven tests.** Every scraper has at least one Go test loading a
  saved HTML file from `test/fixtures/` and asserting against a typed struct.
  This is the contract that catches markup drift.
- **Encoding.** UA locale pages return UTF-8; no transcoding dance needed.

### 4.2 CSS class situation (important — read before writing selectors)

Live recon confirmed that hotline.ua does **not** use BEM, semantic class names,
or stable utility classes. The CSS classes observed are abbreviated tokens like
`s265` and `tx/562` — likely a proprietary or auto-generated utility system
that may change on each deploy. **Do not rely on class-name selectors.**

Instead, use these stable structural hooks:

| Target | Recommended selector |
|---|---|
| Product title | `h1` |
| Product page offers | `a[href^="/go/price/"]` |
| Offer shop name | first `span` inside `a[href^="/go/price/"]` |
| Offer price | `span` containing `₴` inside offer anchor |
| Offer guarantee | `span` with text prefix "Гарантія" |
| Spec table rows | `tr` where first `td` ends with `:` |
| Breadcrumb | ordered sequence of `a` elements in the breadcrumb container |
| Price range text | element containing pattern `\d+ – \d+ ₴` |
| Result count | element containing pattern `\d+ з \d+ товарів` |

**Update 2026-04-24 (curl recon, supersedes earlier WebFetch finding):**
`<script type="application/ld+json">` with `"@type": "Product"` **is present**
on product pages and is the primary source for `ParseProductHTML`. Earlier
WebFetch-based recon was incorrect because WebFetch strips script tag content
when converting HTML to markdown. Always use raw `curl` output when inspecting
inline `<script>` blocks.

### 4.3 URL and ID patterns (confirmed via recon)

- **Canonical product URL:** `/ua/[category-subcategory-slug]/[product-slug]/`
  - Example: `/ua/mobile-mobilnye-telefony-i-smartfony/apple-iphone-17e-256gb-black/`
- **Legacy URL format (still resolves):** `/ua/[category]/[subcategory]/[product-id]/`
  - Example: `/ua/mobile/mobilnye-telefony-i-smartfony/21562714/`
  - The numeric `product-id` here matches the integer `product_id` field in
    the merchant bid API — useful as a canonical internal identifier.
- **Offer redirect URL:** `/go/price/[firm_offer_id]/`
  - The numeric `firm_offer_id` matches the merchant API's `firm_offer_id`.
  - The raw offer URL (before redirect) is what Hotline shows users; it is not
    the direct merchant URL.
- **No numeric ID in canonical slug.** The product slug encodes brand, model,
  storage, colour, and sometimes the SKU (e.g., `apple-iphone-17e-256gb-black`),
  but does **not** embed the Hotline internal `product_id`. If a stable
  identifier is needed, extract it from the legacy URL format or from the
  merchant API.

### 4.4 Search URL format

**Update 2026-04-24 (DevTools recon, confirmed):**

- Global search navigates to `https://hotline.ua/ua/sr/?q=[query]` — this page
  shows a category breakdown (counts per category), not product listings.
- Category pages with `?q=[query]` (e.g. `/ua/mobile/mobilnye-telefony-i-smartfony/?q=iphone+17`)
  deliver SSR-filtered `__NUXT__` catalog state. `?text=` was the old param and
  is **ignored** by the server.
- The best category for a query is discovered via
  `POST /svc/search/api/json-rpc` method `search.menu` — no auth token needed,
  just session cookies. Returns sections with catalogs and result counts.
- Pagination parameter: `?page=N` — confirmed via `__NUXT__` `paginationInfo`
  (`lastPage`, `totalCount`, `itemsPerPage`).

## 5. Feature roadmap

### v1 — Initial release (done)

- [x] Capture fixtures: `test/fixtures/product.html`, `test/fixtures/search.html`.
- [x] Implement `ParseProductHTML` — JSON-LD primary, `__NUXT__` specs secondary.
- [x] Implement `ParseOffersHTML` — `__NUXT__` `state.product.offers.edges`.
- [x] Implement `ParseSearchHTML` — `__NUXT__` `state.catalog.products.collection`.
- [x] Fixture-based tests for all three scrapers (11 tests, all passing).
- [x] Fix panic in `ParseOffersHTML` on missing `firmExtraInfo`.
- [x] Fix `search_products` keyword filtering via `FilterByQuery` client-side filter.
- [x] Fix Cloudflare TLS blocking — `bogdanfinn/tls-client` Chrome_133 JA3 profile.
- [x] Graceful error mapping: `ErrBotBlock` sentinel on 503/403 + Cloudflare markers.
- [x] Installation guide (`docs/installation.md`).

### v1.1 — Breadth (done)

- [x] Pagination: `search_products` accepts `page` and returns `PaginationInfo`
      (`total_items`, `total_pages`, `current_page`, `has_next_page`, `next_page`).
      Confirmed `?page=N` via `__NUXT__` `paginationInfo.lastPage`.
- [x] Category browsing: new tool `list_category(slug, page, price_min,
      price_max)` — browse any category without a keyword.
- [x] Price filters for `search_products` and `list_category`: `price_min`,
      `price_max` passed as `?priceMin=N&priceMax=M`.
- [x] Server-side keyword search: replaced `FilterByQuery` client-side hack with
      `POST /svc/search/api/json-rpc` (`search.menu`) to discover the best
      category, then `?q=<query>` on the category page — confirmed via DevTools
      that `?q=` filters SSR `__NUXT__` catalog state; `?text=` was ignored.
      Works across all hotline.ua categories, not just smartphones.
- [x] Currency normalisation: `Currency` defaults to `"UAH"` when
      `priceCurrency` is absent from JSON-LD.
- [x] `image_url` consistently exposed on `ProductSummary` and `Product`
      (was already populated by both scrapers in v1; verified with test assertions).
- [x] `product_id` exposed as `id` field on `ProductSummary` (from `__NUXT__`
      `_id`) and `Product` (from JSON-LD `sku`).
- [ ] GraphQL search endpoint (`/svc/frontend-api/graphql`) — investigated via
      DevTools: GraphQL only serves auxiliary data (ads, profiles, sections),
      **not** product listings. Not applicable; superseded by `search.menu`.

### v1.2 — Depth

- [ ] `get_reviews(product_url, limit)` — scrape user reviews with rating,
      date, author, pros/cons.
- [ ] `get_price_history(product_url, range)` — check DevTools for a chart
      JSON endpoint on product pages (the chart widget strongly implies one);
      defer if not found.
- [ ] Related/competitor products: surface the "similar" rail.
- [ ] Seller detail: `get_shop(shop_slug)` → ratings, contact info,
      shipping options. Useful for "which of these sellers is reputable?"
      queries.

### v2.0 — Stability & polish

- [ ] Retry with jitter on 5xx and transient Cloudflare 503.
- [ ] Circuit breaker: if error rate spikes, fail fast for 60s instead of
      hammering.
- [ ] Optional persistent cache (sqlite-on-disk) behind an env flag for
      users who want it across MCP restarts.
- [ ] Structured logging to stderr with log level env var.
- [ ] CI: `go test ./...` + `go vet ./...` + `golangci-lint` on every PR.
- [ ] Dependabot for Go modules + GitHub Actions.
- [ ] README screenshots / demo transcript.

### Post-v1 / stretch

- [ ] Reverse-engineer the mobile app's API (it is almost certainly a JSON
      API on top of the same backend, and usually more stable than the web
      XHRs). Capture via mitmproxy on an emulator.
- [ ] MCP *resource* endpoints (read-only URIs) instead of tools, so Claude
      can reference a product page as a citeable resource.
- [ ] MCP *prompts* that bundle common user intents ("cheapest in-stock
      offer", "compare two products side-by-side").

## 6. Non-functional concerns

### Rate limiting

- Current: global 1 req/sec token bucket, configurable via
  `HOTLINE_RATE_LIMIT_RPS`. This is below the observed Cloudflare trigger
  threshold for casual browsing.
- Merchant API (reference): 5 req/sec / 300 req/min.
- Future: per-path budgets (offers tab is hit more often than product pages),
  exponential backoff on 429/503.

### Caching

- Current: in-memory LRU, 10 min TTL, 256 entries.
- Future: extend with content-aware TTLs (offers: 1 min; product page:
  1 hour; category: 15 min).
- Stretch: optional on-disk cache keyed by URL + hash of query params.

### Anti-bot posture

- Rotate UA string per request (done).
- Send a plausible `Accept-Language: uk-UA,uk;q=0.9` — done.
- Do **not** fake `Referer` unless needed; many anti-bot rules key on
  referer mismatches.
- Cloudflare protection is active on hotline.ua (confirmed: direct HTTP
  fetches return 503 without proper TLS fingerprint). Options in escalating
  order:
  1. `github.com/bogdanfinn/tls-client` for JA3/TLS fingerprint spoofing
     (mimics a real browser TLS handshake) — try this first.
  2. Headless Chrome (`chromedp`) as a last resort, behind a feature flag.
- Keep RPS conservative (≤ 1 req/sec); traffic spikes are what trip
  Cloudflare challenges.
- On 503/Cloudflare block, return a structured `ErrBotBlock` error from the
  scraper so the tool layer can surface an actionable message to the user.

### Observability

- Log every outbound request with URL, status, elapsed ms to stderr.
- Surface cache hit/miss in the log line.
- On 503/captcha detection, log the first 200 bytes of the response body so
  we can eyeball what triggered the block.

### Hotline.ua data model reference

From the merchant bid API, the following identifiers exist in Hotline's backend:

| Field | Type | Description |
|---|---|---|
| `product_id` | integer | Hotline's canonical product identifier. Appears in legacy URLs (`/ua/.../[product_id]/`). Not in canonical slug URLs. |
| `firm_offer_id` | string | Unique identifier for a specific shop's offer. Appears in `/go/price/[firm_offer_id]/` redirect URLs on product pages. |

## 7. Testing strategy

1. **Unit / parser tests.** Load a fixture file, call `Parse*`, assert on
   known fields. Golden JSON output where helpful.
2. **Contract tests.** Tiny harness that occasionally (manually or via nightly
   GitHub Action) fetches live pages and diffs against expected fields. This
   is the early-warning for selector drift. Because hotline.ua uses abbreviated
   CSS classes, structural selectors are less likely to drift than class-based
   ones — but layout restructuring is still possible.
3. **Integration smoke test.** `go run ./cmd/hotline-ua-mcp` + a minimal
   MCP client script that invokes each tool and prints the result. Lives in
   `scripts/smoke/`.
4. **Fuzz-ish.** Feed `ParseSearchHTML` random HTML slices to ensure it
   doesn't panic on malformed input; returns `ErrNotImplemented` or a
   structured error instead.

## 8. Risks & open questions

### v1 release blockers — all resolved

| Blocker | Fix applied |
|---|---|
| **Panic on missing `firmExtraInfo`** | `offers.go` — unsafe assertion replaced with `dig(node, "firmExtraInfo", "website")`; guarded by `TestParseOffersHTML_MissingFirmExtraInfo` |
| **`search_products` returns unfiltered results** | `scrapers.FilterByQuery` added; all query words matched case-insensitively against title after parse; guarded by `TestFilterByQuery`. GraphQL endpoint still unknown — deferred to v1.1 |
| **Cloudflare TLS blocking** | `client.go` rewritten to use `bogdanfinn/tls-client` Chrome_133 profile; `ErrBotBlock` sentinel added with 503/403 + body-marker detection; guarded by `TestIsBotBlock` |

### Risk table

| Risk | Likelihood | Mitigation |
|---|---|---|
| Cloudflare blocks plain HTTP (TLS fingerprint mismatch) | **Mitigated** | `bogdanfinn/tls-client` Chrome_133 profile in use; if Cloudflare adapts, update the profile or try a newer Chrome version |
| Abbreviated CSS classes change on redeploy | High | Use structural/attribute selectors (see §4.2) instead of class names |
| Markup rewrite breaks structural selectors | Medium | Fixture-driven tests + nightly contract test |
| Offers page has no JSON endpoint | Likely (SSR confirmed) | HTML path is primary; keep `ParseOffersJSON` stub in case DevTools reveals an XHR |
| ToS / legal escalation | Low (research use) | Keep to public pages, respect rate limits |
| Hotline adds CAPTCHA on product pages | Medium | Detect + return actionable MCP error; advise slowing down |

### Open questions — status after 2026-04-24 recon

| Question | Status | Finding |
|---|---|---|
| Is there JSON-LD / `__NUXT__` on product pages? | **Answered: Both present** | schema.org/Product JSON-LD + 144 KB `window.__NUXT__` IIFE both present in SSR. JSON-LD has price range / rating; `__NUXT__` has individual offer prices and specs. |
| Does the offers tab load via XHR or SSR? | **Answered: SSR** | All 46 offer nodes (with prices) are in the initial `__NUXT__` state. No XHR trigger needed. |
| Is there a stable numeric product ID in the URL? | **Answered** | `product_id` is the `sku` field in JSON-LD (exposed as `id` on `Product`) and the `_id` field in `__NUXT__` catalog state (exposed as `id` on `ProductSummary`). Both populated since v1. |
| What's the pagination mechanism for search? | **Answered** | `?page=N` confirmed via `__NUXT__` `paginationInfo` fields (`lastPage`, `totalCount`, `itemsPerPage`). Implemented in v1.1. |
| Are XHR endpoints available for search/offers? | **Answered (v1.1)** | `POST /svc/search/api/json-rpc` (`search.menu`) returns category breakdown for any query — no auth token needed. GraphQL endpoint (`/svc/frontend-api/graphql`) only handles auxiliary data (ads, profiles), not products. Category pages with `?q=` provide SSR-filtered results without any XHR. |

## 9. Release cadence

- **v1** — initial release. All three tools functional, 11 tests passing,
  Cloudflare bypass via Chrome_133 JA3, installation guide shipped (done).
- **v1.1** — breadth: pagination, category browsing, GraphQL search, image
  URLs. Monthly-ish cadence as features land.
- **v1.2** — depth: reviews, price history, seller detail.
- **v2.0** — stability: retry/circuit-breaker, persistent cache, structured
  logging, CI pipeline, Dependabot.

Tags are the trigger (`v*`). See README → Releases.
