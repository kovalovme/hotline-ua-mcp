# Hotline.ua Integration Roadmap

**Status:** v0.1 scaffold shipped. Scrapers are stubbed; real extraction blocked
on fixture capture and selector work. This doc is the living plan for everything
downstream. **Section 8 open questions were partially answered via live browser
recon on 2026-04-24 — see updates inline.**

## 1. Current state (v0.1)

- MCP stdio server in Go, three tools registered: `search_products`,
  `get_product`, `list_offers`.
- HTTP client with UA rotation, global token-bucket rate limit, LRU response
  cache, cookie jar, realistic headers.
- Claude Code plugin manifests + `.mcp.json` + marketplace entry.
- GoReleaser pipeline triggered on `v*` tags (linux/darwin/windows, amd64/arm64).
- **Gap:** selector bodies in `internal/scrapers/{search,product,offers}.go`
  are placeholders marked `TODO(fixture)`. Nothing returns real data yet.

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

**Important 2026-04-24 update:** Live recon found no JSON-LD, no `__NUXT__`
inline state blob, and no `<script type="application/json">` data injection on
product pages. The Nuxt SSR output is plain HTML. The full offers list (46
merchants on the test product) was present in the SSR response — no XHR
trigger was needed. The JSON-first path remains worth checking via DevTools,
but the HTML path is confirmed sufficient for all three tools.

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

JSON-LD (`<script type="application/ld+json">`) and schema.org microdata
(`itemtype`, `itemprop`) were **not found** on the pages tested. Do not
assume they are present; parse raw HTML only.

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

- Search page: `https://hotline.ua/ua/search/?q=[url-encoded-query]`
- Pagination parameter: **not yet confirmed** (direct fetch was blocked).
  Candidates: `?page=N`, `?p=N`, or a URL segment. Must verify via DevTools
  on a live browser session before implementing `search_products` pagination.

## 5. Feature roadmap

### v0.2 — "Make v1 tools actually return data" (blocks on fixtures)

- [ ] Capture fixtures: `search.html`, `product.html`, `offers.html`.
      (JSON offers endpoint was not confirmed; treat HTML as primary path.)
- [ ] Finalise `ParseSearchHTML` → returns real `ProductSummary` values.
      Use structural selectors (see §4.2); anchor `href` prefix
      `/ua/mobile-` (or similar) to find product cards.
- [ ] Finalise `ParseProductHTML` → returns full `Product` (title via `h1`,
      price range via `\d+ – \d+ ₴` text pattern, spec rows via `tr/td`).
- [ ] Implement `ParseOffersHTML` (confirmed primary path: offers are SSR,
      anchor `a[href^="/go/price/"]`). Retire `ParseOffersJSON` stub unless
      a real JSON endpoint is discovered.
- [ ] Fixture-based tests for all three scrapers.
- [ ] Graceful error mapping: distinguish network error, 503/Cloudflare block,
      captcha page, "no results", and expose each as an actionable MCP error.

### v0.3 — Breadth

- [ ] Pagination: `search_products` accepts `page` and returns
      `total_results` + `next_page` hints. Verify pagination URL param via
      DevTools before implementing (see §4.4).
- [ ] Category browsing: new tool `list_category(slug, sort, filters)` for
      browsing without a keyword (e.g. all smartphones under 20k UAH).
- [ ] Sorting/filters for `search_products`: price range, brand, in-stock,
      rating, delivery city.
- [ ] Currency normalisation: always report UAH, no silent unit changes.
- [ ] Expose `image_url` consistently on `ProductSummary` + `Product`.
- [ ] Extract `product_id` integer from the legacy URL format (`/ua/.../12345/`)
      to expose as a stable canonical identifier, if discoverable from product
      pages (e.g. via a `<link rel="alternate">` or a redirect chain).

### v0.4 — Depth

- [ ] `get_reviews(product_url, limit)` — scrape user reviews with rating,
      date, author, pros/cons.
- [ ] `get_price_history(product_url, range)` — check DevTools for a chart
      JSON endpoint on product pages (the chart widget strongly implies one);
      defer if not found.
- [ ] Related/competitor products: surface the "similar" rail.
- [ ] Seller detail: `get_shop(shop_slug)` → ratings, contact info,
      shipping options. Useful for "which of these sellers is reputable?"
      queries.

### v1.0 — Stability & polish

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

| Risk | Likelihood | Mitigation |
|---|---|---|
| Cloudflare blocks plain HTTP (TLS fingerprint mismatch) | **Confirmed** | Use `tls-client` for JA3 spoofing; document `HOTLINE_TLS_CLIENT=1` env flag |
| Abbreviated CSS classes change on redeploy | High | Use structural/attribute selectors (see §4.2) instead of class names |
| Markup rewrite breaks structural selectors | Medium | Fixture-driven tests + nightly contract test |
| Offers page has no JSON endpoint | Likely (SSR confirmed) | HTML path is primary; keep `ParseOffersJSON` stub in case DevTools reveals an XHR |
| ToS / legal escalation | Low (research use) | Keep to public pages, respect rate limits |
| Hotline adds CAPTCHA on product pages | Medium | Detect + return actionable MCP error; advise slowing down |

### Open questions — status after 2026-04-24 recon

| Question | Status | Finding |
|---|---|---|
| Is there JSON-LD / `__NEXT_DATA__` on product pages? | **Answered: No** | No JSON-LD, no `__NUXT__` inline state, no schema.org microdata found. Nuxt SSR outputs plain HTML only. |
| Does the offers tab load via XHR or SSR? | **Answered: SSR** | Full offers list (46 merchants) is present in the initial HTML response. No XHR trigger needed. |
| Is there a stable numeric product ID in the URL? | **Partially answered** | `product_id` integer exists in the merchant API and in legacy URLs (`/ua/.../12345/`). Canonical slug URLs do not contain it. Need to confirm whether a canonical product page exposes the ID in any attribute or link. |
| What's the pagination mechanism for search? | **Still open** | Direct HTTP fetch of `?page=2` and `?p=2` returned 404. Must verify via live DevTools session. `?page=N` is the most common Nuxt.js convention. |
| Are XHR endpoints available for search/offers? | **Still open** | Cloudflare blocked direct fetch attempts. Must use DevTools on a live browser session to intercept Network → Fetch/XHR traffic. |

## 9. Release cadence

- **v0.1** — scaffold (done).
- **v0.2** — first release with working scrapers. Ship as soon as all three
  tools return real data for the golden-path queries.
- **v0.x** monthly-ish as features land.
- **v1.0** — when all three v1-scope tools have been stable across at least
  one markup change and CI has been green for a couple of weeks.

Tags are the trigger (`v*`). See README → Releases.
