package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/kovalovme/hotline-ua-mcp/internal/httpclient"
	"github.com/kovalovme/hotline-ua-mcp/internal/scrapers"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type GetProductArgs struct {
	URL string `json:"url" jsonschema:"full hotline.ua product page URL (UA locale), e.g. https://hotline.ua/ua/mobile/apple-iphone-15-128gb/"`
}

func GetProduct(client *httpclient.Client) func(context.Context, *mcp.CallToolRequest, GetProductArgs) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, args GetProductArgs) (*mcp.CallToolResult, any, error) {
		if args.URL == "" {
			return nil, nil, fmt.Errorf("url is required")
		}
		if !strings.HasPrefix(args.URL, "https://hotline.ua/") {
			return nil, nil, fmt.Errorf("url must be on hotline.ua")
		}

		body, err := client.Get(ctx, args.URL)
		if err != nil {
			return nil, nil, err
		}
		product, err := scrapers.ParseProductHTML(body)
		if err != nil {
			return nil, nil, err
		}
		return textJSON(product)
	}
}
