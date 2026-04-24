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
	Query    string  `json:"query" jsonschema:"product search query in Ukrainian or English, e.g. 'iPhone 15'"`
	Limit    int     `json:"limit,omitempty" jsonschema:"max results to return (default 10, max 40)"`
	Page     int     `json:"page,omitempty" jsonschema:"page number for pagination (1-based, default 1)"`
	PriceMin float64 `json:"price_min,omitempty" jsonschema:"minimum price filter in UAH"`
	PriceMax float64 `json:"price_max,omitempty" jsonschema:"maximum price filter in UAH"`
}

type SearchProductsResult struct {
	Query      string                 `json:"query"`
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

		f := scrapers.SearchFilters{
			Page:     page,
			PriceMin: args.PriceMin,
			PriceMax: args.PriceMax,
		}
		u := scrapers.BuildSearchURLFiltered(args.Query, f)
		body, err := client.Get(ctx, u)
		if err != nil {
			return nil, nil, err
		}

		results, pagination, err := scrapers.ParseSearchPage(body, page)
		if err != nil {
			return nil, nil, err
		}
		results = scrapers.FilterByQuery(results, args.Query)
		if len(results) > limit {
			results = results[:limit]
		}

		payload := SearchProductsResult{
			Query:      args.Query,
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
