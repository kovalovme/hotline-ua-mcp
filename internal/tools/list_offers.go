package tools

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/kovalovme/hotline-ua-mcp/internal/httpclient"
	"github.com/kovalovme/hotline-ua-mcp/internal/scrapers"
	"github.com/kovalovme/hotline-ua-mcp/internal/types"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type ListOffersArgs struct {
	ProductURL string `json:"product_url" jsonschema:"full hotline.ua product page URL whose offers to list"`
	Sort       string `json:"sort,omitempty" jsonschema:"sort order: 'price_asc' (default) or 'price_desc'"`
	Limit      int    `json:"limit,omitempty" jsonschema:"max offers to return (default 20, max 100)"`
	InStock    bool   `json:"in_stock,omitempty" jsonschema:"if true, only return in-stock offers"`
}

type ListOffersResult struct {
	ProductURL string        `json:"product_url"`
	Count      int           `json:"count"`
	Offers     []types.Offer `json:"offers"`
}

func ListOffers(client *httpclient.Client) func(context.Context, *mcp.CallToolRequest, ListOffersArgs) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, args ListOffersArgs) (*mcp.CallToolResult, any, error) {
		if args.ProductURL == "" {
			return nil, nil, fmt.Errorf("product_url is required")
		}
		if !strings.HasPrefix(args.ProductURL, "https://hotline.ua/") {
			return nil, nil, fmt.Errorf("product_url must be on hotline.ua")
		}
		limit := args.Limit
		if limit <= 0 {
			limit = 20
		}
		if limit > 100 {
			limit = 100
		}

		// TODO(fixture): once the real offers JSON endpoint is confirmed,
		// fetch it here first; fall back to HTML only if that fails.
		body, err := client.Get(ctx, args.ProductURL)
		if err != nil {
			return nil, nil, err
		}
		offers, err := scrapers.ParseOffersHTML(body)
		if err != nil {
			return nil, nil, err
		}

		if args.InStock {
			filtered := offers[:0]
			for _, o := range offers {
				if o.InStock {
					filtered = append(filtered, o)
				}
			}
			offers = filtered
		}

		sort.Slice(offers, func(i, j int) bool {
			if args.Sort == "price_desc" {
				return offers[i].Price > offers[j].Price
			}
			return offers[i].Price < offers[j].Price
		})

		if len(offers) > limit {
			offers = offers[:limit]
		}

		return textJSON(ListOffersResult{
			ProductURL: args.ProductURL,
			Count:      len(offers),
			Offers:     offers,
		})
	}
}
