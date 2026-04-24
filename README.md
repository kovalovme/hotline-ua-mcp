# hotline-ua-mcp

A Claude Code plugin that exposes [hotline.ua](https://hotline.ua) — Ukraine's
largest price-comparison site — as MCP tools. Built in Go.

Hotline has no official public search API. This plugin parses the public web
UI using server-side rendered HTML and a browser-grade TLS fingerprint to
bypass Cloudflare bot checks. UA locale only.

## Tools

| Tool | Input | Output |
|---|---|---|
| `search_products` | `query`, `limit`, `page`, `price_min`, `price_max` | product summaries with pagination; searches any category |
| `get_product` | `url` | full product: title, aggregate price range, rating, specs |
| `list_offers` | `product_url`, `sort`, `limit`, `in_stock` | seller offers sorted by price |
| `list_category` | `slug`, `page`, `price_min`, `price_max` | browse a category without a keyword |

`search_products` uses the `search.menu` JSON-RPC to find the best category for
your query, then fetches server-side filtered results — works across all
hotline.ua categories, not just smartphones.

## Install

See [docs/installation.md](docs/installation.md) for the full guide. Quick
start:

```bash
git clone https://github.com/kovalovme/hotline-ua-mcp.git
cd hotline-ua-mcp
go build -o bin/hotline-ua-mcp ./cmd/hotline-ua-mcp
```

Then in Claude Code:

```
/plugin marketplace add /absolute/path/to/hotline-ua-mcp
/plugin install hotline-ua@hotline-ua-marketplace
```

## Configuration

Environment variables (set on the MCP server entry in `.mcp.json`):

| Var | Default | Description |
|---|---|---|
| `HOTLINE_RATE_LIMIT_RPS` | `1` | Global requests/sec cap |
| `HOTLINE_CACHE_TTL_SEC` | `600` | LRU response cache TTL (seconds) |

## Layout

```
.claude-plugin/
  plugin.json           # plugin metadata
  marketplace.json      # single-plugin marketplace so the repo is installable as-is
.mcp.json               # MCP server registration consumed by Claude Code
cmd/hotline-ua-mcp/     # main binary
docs/
  installation.md       # full install guide
  implementation-status.md
  planning/roadmap.md
internal/
  httpclient/           # rate-limited, cached, Chrome-fingerprint HTTP client
  scrapers/             # HTML → typed values (JSON-LD + window.__NUXT__)
  tools/                # MCP tool handlers
  types/                # shared data shapes
test/fixtures/          # saved hotline.ua pages for offline tests
```

## Development

```bash
go build ./...
go vet ./...
go test ./...          # 23 tests
```

## Releases

Releases are cut by pushing a `v*` tag. A GitHub Actions workflow
(`.github/workflows/release.yml`) runs [GoReleaser](https://goreleaser.com)
which cross-compiles for:

- linux/amd64, linux/arm64
- darwin/amd64, darwin/arm64
- windows/amd64

Each archive bundles the binary, `README.md`, `LICENSE`, `.mcp.json`, and the
`.claude-plugin/` manifests. A `checksums.txt` is attached to the release.

```bash
git tag v1.0.0
git push origin v1.0.0
```

Validate the config locally:

```bash
goreleaser check
goreleaser release --snapshot --clean --skip=publish
```

## License

MIT.
