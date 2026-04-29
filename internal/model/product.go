package model

// Category represents a product category to parse.
type Category struct {
	Slug string `yaml:"slug"`
	Name string `yaml:"name"`
}

// Product is the parsed domain object — what we export.
type Product struct {
	Category string
	Name     string
	Price    float64
	URL      string
}

// SKU is the raw API response shape for a single product.
type SKU struct {
	ID     string  `json:"id"`
	Name   string  `json:"name"`
	Slug   string  `json:"slug"`
	Prices Prices  `json:"prices"`
}

// Prices holds pricing info from the API.
type Prices struct {
	Price         float64  `json:"price"`
	PricePerUnit  float64  `json:"pricePerUnit"`
	DiscountPrice *float64 `json:"discountPrice"`
}

// EffectivePrice returns discount price if available, otherwise regular price.
func (p Prices) EffectivePrice() float64 {
	if p.DiscountPrice != nil && *p.DiscountPrice > 0 {
		return *p.DiscountPrice
	}
	return p.Price
}

// ProductsResponse is the top-level API response for a product list.
type ProductsResponse struct {
	SKUs       []SKU `json:"skus"`
	TotalCount int   `json:"totalCount"`
}
