# Installation Guide

This guide covers every way to run hotline-ua-mcp and connect it to Claude Code.

## Prerequisites

- **Claude Code** — desktop app, VS Code extension, or JetBrains extension.
- **Go 1.22+** — only required if you are building from source.

---

## Option A — Pre-built binary (recommended)

1. Go to the [Releases page](https://github.com/kovalovme/hotline-ua-mcp/releases) and download the archive for your platform:

   | OS | Arch | File |
   |---|---|---|
   | macOS | Apple Silicon | `hotline-ua-mcp_*_darwin_arm64.tar.gz` |
   | macOS | Intel | `hotline-ua-mcp_*_darwin_amd64.tar.gz` |
   | Linux | x86-64 | `hotline-ua-mcp_*_linux_amd64.tar.gz` |
   | Linux | ARM64 | `hotline-ua-mcp_*_linux_arm64.tar.gz` |
   | Windows | x86-64 | `hotline-ua-mcp_*_windows_amd64.zip` |

2. Extract the archive to a permanent directory, for example `~/.local/share/hotline-ua-mcp`:

   ```bash
   mkdir -p ~/.local/share/hotline-ua-mcp
   tar -xzf hotline-ua-mcp_*_linux_amd64.tar.gz -C ~/.local/share/hotline-ua-mcp
   ```

3. Register it as a Claude Code plugin (see [Plugin registration](#plugin-registration) below).

---

## Option B — Build from source

```bash
git clone https://github.com/kovalovme/hotline-ua-mcp.git
cd hotline-ua-mcp
go build -o bin/hotline-ua-mcp ./cmd/hotline-ua-mcp
```

The binary lands at `bin/hotline-ua-mcp` inside the cloned directory. Use that directory path in the registration step below.

To verify:

```bash
./bin/hotline-ua-mcp --help   # prints usage
go test ./...                 # 11 tests, all should pass
```

---

## Plugin registration

The plugin ships a local marketplace manifest (`.claude-plugin/marketplace.json`) so it can be installed with two Claude Code slash commands.

Run these inside Claude Code (chat input or terminal):

```
/plugin marketplace add /absolute/path/to/hotline-ua-mcp
/plugin install hotline-ua@hotline-ua-marketplace
```

Replace `/absolute/path/to/hotline-ua-mcp` with:
- **Pre-built:** the directory you extracted the archive into (e.g. `~/.local/share/hotline-ua-mcp`).
- **Source build:** the root of the cloned repo.

After installation Claude Code reads `.mcp.json` and starts the MCP server automatically. The three tools (`search_products`, `get_product`, `list_offers`) become available in every session.

---

## Manual MCP configuration (alternative)

If you prefer to wire the server directly without using the plugin system, add an entry to your project's `.mcp.json` (or your global Claude Code MCP config):

```json
{
  "mcpServers": {
    "hotline-ua": {
      "command": "/absolute/path/to/bin/hotline-ua-mcp",
      "env": {
        "HOTLINE_RATE_LIMIT_RPS": "1",
        "HOTLINE_CACHE_TTL_SEC": "600"
      }
    }
  }
}
```

---

## Configuration

All configuration is done through environment variables on the MCP server entry.

| Variable | Default | Description |
|---|---|---|
| `HOTLINE_RATE_LIMIT_RPS` | `1` | Maximum outbound requests per second. Keep at 1 or below to avoid Cloudflare rate triggers. |
| `HOTLINE_CACHE_TTL_SEC` | `600` | In-memory response cache TTL in seconds (10 minutes). Increase to reduce repeat fetches for stable pages. |

---

## Verifying the installation

Ask Claude anything that exercises the tools, for example:

> "Search hotline.ua for iPhone 17 and show me the cheapest seller offers."

Claude will call `search_products` then `list_offers` and return structured price data.

If the server is not reachable you will see an MCP error in the Claude response. Run the binary directly to check for startup errors:

```bash
/path/to/bin/hotline-ua-mcp
```

It should block silently on stdin (MCP stdio transport). Press Ctrl-C to exit.

---

## Troubleshooting

### "hotline.ua blocked the request (Cloudflare challenge)"

The server uses a Chrome 133 TLS fingerprint to bypass Cloudflare bot checks. If you see this error:

- Wait 60 seconds and retry — Cloudflare challenges are often transient.
- Lower `HOTLINE_RATE_LIMIT_RPS` to `0.5` or less.
- Check that you are not running multiple instances simultaneously (each counts against the rate budget).

### Tools return no results for a search query

`search_products` is currently limited to the smartphones category on hotline.ua and applies client-side keyword filtering. Queries for product categories other than smartphones will return zero results. Multi-category search is planned for v0.3.

### "url must be on hotline.ua"

`get_product` and `list_offers` validate that the URL starts with `https://hotline.ua/`. Pass a full canonical URL from a `search_products` result, for example:

```
https://hotline.ua/ua/mobile-mobilnye-telefony-i-smartfony/apple-iphone-17-256gb-black/
```

---

## Updating

### Pre-built binary

Download the new archive from the Releases page, extract it over the existing directory, and restart Claude Code (or reload the MCP server from the settings panel).

### Source build

```bash
git -C /path/to/hotline-ua-mcp pull
go build -o bin/hotline-ua-mcp ./cmd/hotline-ua-mcp
```

Restart Claude Code to pick up the new binary.
