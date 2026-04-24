// Package tools wires MCP tool handlers to the HTTP client and scrapers.
package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/kovalovme/hotline-ua-mcp/internal/httpclient"
	"github.com/kovalovme/hotline-ua-mcp/internal/scrapers"
	"github.com/kovalovme/hotline-ua-mcp/internal/types"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type SearchProductsArgs struct {
	Query string `json:"query" jsonschema:"product search query in Ukrainian or English, e.g. 'iPhone 15'"`
	Limit int    `json:"limit,omitempty" jsonschema:"max results to return (default 10, max 40)"`
}

type SearchProductsResult struct {
	Query   string                  `json:"query"`
	Count   int                     `json:"count"`
	Results []types.ProductSummary  `json:"results"`
}

func SearchProducts(client *httpclient.Client) func(context.Context, *mcp.CallToolRequest, SearchProductsArgs) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, args SearchProductsArgs) (*mcp.CallToolResult, any, error) {
		if args.Query == "" {
			return nil, nil, fmt.Errorf("query is required")
		}
		limit := args.Limit
		if limit <= 0 {
			limit = 10
		}
		if limit > 40 {
			limit = 40
		}

		u := scrapers.BuildSearchURL(args.Query)
		body, err := client.Get(ctx, u)
		if err != nil {
			return nil, nil, err
		}

		results, err := scrapers.ParseSearchHTML(body)
		if err != nil {
			return nil, nil, err
		}
		results = scrapers.FilterByQuery(results, args.Query)
		if len(results) > limit {
			results = results[:limit]
		}

		payload := SearchProductsResult{
			Query:   args.Query,
			Count:   len(results),
			Results: results,
		}
		return textJSON(payload)
	}
}

func textJSON(v any) (*mcp.CallToolResult, any, error) {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return nil, nil, err
	}
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: string(b)}},
	}, v, nil
}
