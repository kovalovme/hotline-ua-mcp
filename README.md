# hotline-ua-mcp

A Claude Code plugin that exposes [hotline.ua](https://hotline.ua) — Ukraine's
largest price-comparison site — as MCP tools. Built in Go.

**Status: v0.1 scaffold.** The project structure, MCP server, plugin manifests,
HTTP client (rate limit + UA rotation + LRU cache), and tool wiring are in
place. Scraper selectors are stubbed and need to be finalized against real
HTML/JSON fixtures (see [Fixtures](#fixtures) below).

Hotline has no official public search API. This plugin parses the public web
UI. UA locale only.

## Tools

| Tool | Input | Output |
|---|---|---|
| `search_products` | `query`, `limit` | list of product summaries (title, URL, price range, offer count, rating) |
| `get_product` | `url` | full product: title, aggregate price range, rating, specs |
| `list_offers` | `product_url`, `sort`, `limit`, `in_stock` | seller offers sorted by price |

## Install

Build the binary, then install the plugin.

```bash
git clone https://github.com/kovalovme/hotline-ua-mcp.git
cd hotline-ua-mcp
go build -o bin/hotline-ua-mcp ./cmd/hotline-ua-mcp
```

In Claude Code:

```
/plugin marketplace add /absolute/path/to/hotline-ua-mcp
/plugin install hotline-ua@hotline-ua-marketplace
```

The plugin registers an MCP server via `.mcp.json` pointing at
`${CLAUDE_PLUGIN_ROOT}/bin/hotline-ua-mcp`.

## Configuration

Environment variables (set on the MCP server entry in `.mcp.json`):

| Var | Default | Description |
|---|---|---|
| `HOTLINE_RATE_LIMIT_RPS` | `1` | Global requests/sec cap |
| `HOTLINE_CACHE_TTL_SEC` | `600` | LRU response cache TTL (seconds) |

## Fixtures

Live endpoint recon was not possible from the scaffolding sandbox. Before
shipping, capture these files into `test/fixtures/`:

1. `search.html` — save-as from `https://hotline.ua/ua/search/?q=iphone+15`
2. `product.html` — save-as from any product detail page
3. `offers.json` — in DevTools → Network, open the "Де купити" / offers tab
   on a product page and copy the XHR response. If no JSON endpoint exists,
   skip this and provide `offers.html` instead.
4. `offers.html` — fallback; save-as from the offers tab.

With fixtures in place, finalize the selectors in
`internal/scrapers/{search,product,offers}.go` (search for `TODO(fixture)`).

## Layout

```
.claude-plugin/
  plugin.json           # plugin metadata
  marketplace.json      # single-plugin marketplace so the repo is installable as-is
.mcp.json               # MCP server registration consumed by Claude Code
cmd/hotline-ua-mcp/     # main binary
internal/
  httpclient/           # undici-style: rate-limited, cached, UA-rotating HTTP
  scrapers/             # HTML/JSON → typed values
  tools/                # MCP tool handlers
  types/                # shared data shapes
test/fixtures/          # saved hotline.ua pages for offline tests
```

## Development

```bash
go build ./...
go vet ./...
go test ./...
```

## License

MIT.
