// Package types defines the shared data shapes returned by scrapers and tools.
package types

type ProductSummary struct {
	ID          string  `json:"id"`
	Title       string  `json:"title"`
	URL         string  `json:"url"`
	ImageURL    string  `json:"image_url,omitempty"`
	PriceMin    float64 `json:"price_min,omitempty"`
	PriceMax    float64 `json:"price_max,omitempty"`
	Currency    string  `json:"currency,omitempty"`
	OffersCount int     `json:"offers_count,omitempty"`
	Rating      float64 `json:"rating,omitempty"`
	ReviewCount int     `json:"review_count,omitempty"`
	ShortSpecs  string  `json:"short_specs,omitempty"`
}

type Product struct {
	ID          string            `json:"id"`
	Title       string            `json:"title"`
	URL         string            `json:"url"`
	ImageURL    string            `json:"image_url,omitempty"`
	PriceMin    float64           `json:"price_min,omitempty"`
	PriceMax    float64           `json:"price_max,omitempty"`
	Currency    string            `json:"currency,omitempty"`
	OffersCount int               `json:"offers_count,omitempty"`
	Rating      float64           `json:"rating,omitempty"`
	ReviewCount int               `json:"review_count,omitempty"`
	Description string            `json:"description,omitempty"`
	Specs       map[string]string `json:"specs,omitempty"`
}

type Offer struct {
	ShopName  string  `json:"shop_name"`
	ShopURL   string  `json:"shop_url,omitempty"`
	OfferURL  string  `json:"offer_url"`
	Price     float64 `json:"price"`
	Currency  string  `json:"currency"`
	InStock   bool    `json:"in_stock"`
	Condition string  `json:"condition,omitempty"`
	Guarantee string  `json:"guarantee,omitempty"`
}

// PaginationInfo describes one page of paginated search/category results.
type PaginationInfo struct {
	TotalItems  int  `json:"total_items"`
	TotalPages  int  `json:"total_pages"`
	CurrentPage int  `json:"current_page"`
	HasNextPage bool `json:"has_next_page"`
	NextPage    *int `json:"next_page,omitempty"`
}
