package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/kovalovme/hotline-ua-mcp/internal/httpclient"
	"github.com/kovalovme/hotline-ua-mcp/internal/scrapers"
	"github.com/kovalovme/hotline-ua-mcp/internal/types"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type ListCategoryArgs struct {
	Slug     string  `json:"slug" jsonschema:"category path segment(s) after /ua/, e.g. 'mobile/mobilnye-telefony-i-smartfony'"`
	Page     int     `json:"page,omitempty" jsonschema:"page number (1-based, default 1)"`
	PriceMin float64 `json:"price_min,omitempty" jsonschema:"minimum price filter in UAH"`
	PriceMax float64 `json:"price_max,omitempty" jsonschema:"maximum price filter in UAH"`
	Limit    int     `json:"limit,omitempty" jsonschema:"max results to return per page (default 24, max 100)"`
}

type ListCategoryResult struct {
	Slug       string                 `json:"slug"`
	Count      int                    `json:"count"`
	Results    []types.ProductSummary `json:"results"`
	Pagination types.PaginationInfo   `json:"pagination"`
}

func ListCategory(client *httpclient.Client) func(context.Context, *mcp.CallToolRequest, ListCategoryArgs) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, args ListCategoryArgs) (*mcp.CallToolResult, any, error) {
		slug := strings.TrimSpace(args.Slug)
		if slug == "" {
			return nil, nil, fmt.Errorf("slug is required")
		}
		limit := args.Limit
		if limit <= 0 {
			limit = 24
		}
		if limit > 100 {
			limit = 100
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
		u := scrapers.BuildCategoryURL(slug, f)
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

		return textJSON(ListCategoryResult{
			Slug:       slug,
			Count:      len(results),
			Results:    results,
			Pagination: pagination,
		})
	}
}
