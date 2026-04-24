# Hotline.ua Integration Roadmap

**Status:** v0.1 scaffold shipped. Scrapers are stubbed; real extraction blocked
on fixture capture and endpoint discovery. This doc is the living plan for
everything downstream.

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

1. **Internal JSON / XHR endpoints.** The site itself loads the offers tab,
   "show more" pagination, and filter updates via XHR. These payloads are
   unofficial but usually richer and more stable against cosmetic markup
   changes. Example suspected endpoints (to be confirmed via DevTools):
   - `/svc/frontend-api/shop-prices/...` or similar for offers
   - `/svc/frontend-api/search/...` for search
   - `/ua/<category>/<product-slug>/` → inline `__NEXT_DATA__` / preloaded
     state blob if SSR-hydrated
2. **Server-rendered HTML.** Product cards on search and product pages carry
   enough data to populate summaries. Slowest to maintain because markup
   churns.
3. **Public Hotline APIs.** Only one exists
   ([`/ua/about/api_auctions/`](https://hotline.ua/ua/about/api_auctions/)) and
   it is merchant-side: manage auction bids with an `auth_token`, 300 req/min.
   **Not usable** for product search. Documented here so we don't rediscover
   it.
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

**Rule of thumb:** if a JSON endpoint exists and responds with 200 using the
same cookies as page navigation, prefer it. Fall back to HTML scraping only
when JSON is cumbersome (requires CSRF token refresh, mixed content, etc.).

## 4. HTML scraping strategy

Used when JSON isn't available or is more fragile.

- **Library:** `github.com/PuerkitoBio/goquery`. Selectors live in
  `internal/scrapers/*.go`, one file per page type.
- **Fixture-driven tests.** Every scraper has at least one Go test loading a
  saved HTML file from `test/fixtures/` and asserting against a typed struct.
  This is the contract that catches markup drift.
- **Selector hygiene.** Prefer semantic hooks (`data-*` attributes, microdata,
  JSON-LD `<script type="application/ld+json">`) over positional selectors.
  Hotline product pages historically include JSON-LD — harvest it first, CSS
  selectors second.
- **Encoding.** UA locale pages return UTF-8; no transcoding dance needed.

## 5. Feature roadmap

### v0.2 — "Make v1 tools actually return data" (blocks on fixtures)

- [ ] Capture fixtures: `search.html`, `product.html`, `offers.json` (or
      `offers.html` fallback).
- [ ] Finalise `ParseSearchHTML` → returns real `ProductSummary` values.
- [ ] Finalise `ParseProductHTML` → returns full `Product` (title, price
      range, rating, specs map, JSON-LD if present).
- [ ] Pick offers path: either `ParseOffersJSON` + real endpoint, or
      `ParseOffersHTML` fallback; implement one, mark the other as future work.
- [ ] Fixture-based tests for all three scrapers.
- [ ] Graceful error mapping: distinguish network error, 403/bot-block,
      captcha page, "no results", and expose each as an actionable MCP error.

### v0.3 — Breadth

- [ ] Pagination: `search_products` accepts `page` and returns
      `total_results` + `next_page` hints.
- [ ] Category browsing: new tool `list_category(slug, sort, filters)` for
      browsing without a keyword (e.g. all smartphones under 20k UAH).
- [ ] Sorting/filters for `search_products`: price range, brand, in-stock,
      rating, delivery city.
- [ ] Currency normalisation: always report UAH, no silent unit changes.
- [ ] Expose `image_url` consistently on `ProductSummary` + `Product`.

### v0.4 — Depth

- [ ] `get_reviews(product_url, limit)` — scrape user reviews with rating,
      date, author, pros/cons.
- [ ] `get_price_history(product_url, range)` — if Hotline exposes the
      historical price chart as JSON (the product page has a chart widget
      that hints at one); otherwise defer.
- [ ] Related/competitor products: surface the "similar" rail.
- [ ] Seller detail: `get_shop(shop_slug)` → ratings, contact info,
      shipping options. Useful for "which of these sellers is reputable?"
      queries.

### v1.0 — Stability & polish

- [ ] Retry with jitter on 5xx and transient 403.
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
  `HOTLINE_RATE_LIMIT_RPS`.
- Future: per-path budgets (offers tab is hit more often than product pages),
  exponential backoff on 429/403.

### Caching

- Current: in-memory LRU, 10 min TTL, 256 entries.
- Future: extend with content-aware TTLs (offers tab: 1 min; product page:
  1 hour; category: 15 min).
- Stretch: optional on-disk cache keyed by URL + hash of query params.

### Anti-bot posture

- Rotate UA string per request (done).
- Send a plausible Accept-Language (`uk-UA,uk;q=0.9`) — done.
- Do **not** fake `Referer` unless needed; many anti-bot rules key on
  referer mismatches.
- If Cloudflare JS challenge starts firing, options:
  - `github.com/bogdanfinn/tls-client` for JA3 spoofing
  - headless Chrome (`chromedp`) as a last resort, behind a feature flag
- Keep RPS conservative; spikes are what trip challenges.

### Observability

- Log every outbound request with URL, status, elapsed ms to stderr.
- Surface cache hit/miss in the log line.
- On 403/captcha detection, log the first 200 bytes of the response body so
  we can eyeball what triggered the block.

## 7. Testing strategy

1. **Unit / parser tests.** Load a fixture file, call `Parse*`, assert on
   known fields. Golden JSON output where helpful.
2. **Contract tests.** Tiny harness that occasionally (manually or via nightly
   GitHub Action) fetches live pages and diffs against expected fields. This
   is the early-warning for selector drift.
3. **Integration smoke test.** `go run ./cmd/hotline-ua-mcp` + a minimal
   MCP client script that invokes each tool and prints the result. Lives in
   `scripts/smoke/`.
4. **Fuzz-ish.** Feed `ParseSearchHTML` random HTML slices to ensure it
   doesn't panic on malformed input; returns `ErrNotImplemented` or a
   structured error instead.

## 8. Risks & open questions

| Risk | Likelihood | Mitigation |
|---|---|---|
| Anti-bot tightens, blocks plain HTTP | Medium | Add tls-client; document proxy env var |
| Markup rewrite breaks selectors | High (eventually) | Fixture-driven tests + nightly contract test |
| Offers page has no JSON endpoint | Low | HTML fallback already stubbed |
| ToS / legal escalation | Low (research use) | Keep to public pages, respect rate limits |
| Hotline adds CAPTCHA on product pages | Medium | Detect + return actionable MCP error; advise slowing down |

**Open questions for the next working session:**

1. Is there a JSON-LD / `__NEXT_DATA__` blob on product pages? If yes, most
   of `get_product` becomes trivial.
2. Does the offers tab load via XHR or is it server-rendered in the initial
   HTML? Determines the v0.2 offers path.
3. Is there a stable product ID in the URL (numeric?) we can use as canonical
   identifier, or do we always have to use the slug URL?
4. What's the pagination mechanism — query param, URL segment, or
   cursor-based XHR?

## 9. Release cadence

- **v0.1** — scaffold (done).
- **v0.2** — first release with working scrapers. Ship as soon as all three
  tools return real data for the golden-path queries.
- **v0.x** monthly-ish as features land.
- **v1.0** — when all three v1-scope tools have been stable across at least
  one markup change and CI has been green for a couple of weeks.

Tags are the trigger (`v*`). See README → Releases.
