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

const searchMenuURL = httpclient.BaseURL + "/svc/search/api/json-rpc"

type SearchProductsArgs struct {
	Query    string  `json:"query" jsonschema:"product search query in Ukrainian or English, e.g. 'iPhone 15'"`
	Limit    int     `json:"limit,omitempty" jsonschema:"max results to return (default 10, max 40)"`
	Page     int     `json:"page,omitempty" jsonschema:"page number for pagination (1-based, default 1)"`
	PriceMin float64 `json:"price_min,omitempty" jsonschema:"minimum price filter in UAH"`
	PriceMax float64 `json:"price_max,omitempty" jsonschema:"maximum price filter in UAH"`
}

type SearchProductsResult struct {
	Query      string                 `json:"query"`
	Category   string                 `json:"category,omitempty"`
	Count      int                    `json:"count"`
	Results    []types.ProductSummary `json:"results"`
	Pagination types.PaginationInfo   `json:"pagination"`
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
		page := args.Page
		if page <= 0 {
			page = 1
		}

		// Discover the best category for this query via search.menu.
		menuBody, _ := json.Marshal(map[string]any{
			"jsonrpc": "2.0",
			"method":  "search.menu",
			"params":  map[string]any{"q": args.Query, "lang": "uk", "vendor_ids": nil},
			"id":      1,
		})
		menuResp, err := client.PostJSON(ctx, searchMenuURL, menuBody)
		if err != nil {
			return nil, nil, fmt.Errorf("search menu: %w", err)
		}
		categoryPath, err := scrapers.ParseSearchMenuResponse(menuResp)
		if err != nil {
			return nil, nil, err
		}

		f := scrapers.SearchFilters{
			Page:     page,
			PriceMin: args.PriceMin,
			PriceMax: args.PriceMax,
		}
		u := scrapers.BuildCategorySearchURL(categoryPath, args.Query, f)
		body, err := client.Get(ctx, u)
		if err != nil {
			return nil, nil, err
		}

		results, pagination, err := scrapers.ParseSearchPage(body, page)
		if err != nil {
			return nil, nil, err
		}
		if len(results) > limit {
			results = results[:limit]
		}

		payload := SearchProductsResult{
			Query:      args.Query,
			Category:   categoryPath,
			Count:      len(results),
			Results:    results,
			Pagination: pagination,
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
