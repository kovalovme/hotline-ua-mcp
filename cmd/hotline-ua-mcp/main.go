// hotline-ua-mcp is an MCP server exposing hotline.ua public product search
// as tools callable from Claude Code.
//
// Transport: stdio. Register it in a Claude Code plugin's .mcp.json, or run
// the binary directly for manual testing.
package main

import (
	"context"
	"log"
	"os"

	"github.com/kovalovme/hotline-ua-mcp/internal/httpclient"
	"github.com/kovalovme/hotline-ua-mcp/internal/tools"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const version = "0.1.0"

func main() {
	logger := log.New(os.Stderr, "[hotline-ua-mcp] ", log.LstdFlags)

	client, err := httpclient.New()
	if err != nil {
		logger.Fatalf("init http client: %v", err)
	}

	server := mcp.NewServer(&mcp.Implementation{
		Name:    "hotline-ua",
		Version: version,
	}, nil)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "search_products",
		Description: "Search hotline.ua (Ukrainian price-comparison) for products by name. Returns a list of product summaries with price range, offer count, rating, and canonical URL.",
	}, tools.SearchProducts(client))

	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_product",
		Description: "Fetch full details for a single hotline.ua product page: title, aggregate price range, rating, specs.",
	}, tools.GetProduct(client))

	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_offers",
		Description: "List seller offers for a hotline.ua product, sorted by price. Each offer includes shop name, price, stock, and outbound URL.",
	}, tools.ListOffers(client))

	if err := server.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
		logger.Fatalf("server exited: %v", err)
	}
}
