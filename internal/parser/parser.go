package parser

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/url"
	"strconv"

	"github.com/yourorg/grablenta/internal/config"
	"github.com/yourorg/grablenta/internal/model"
)

// Getter is the HTTP interface the parser depends on — easy to mock in tests.
type Getter interface {
	Get(path string, params url.Values) ([]byte, error)
	StoreSlug() string
}

// Parser fetches and converts raw API data into domain Products.
type Parser struct {
	client   Getter
	cfg      *config.Config
	log      *slog.Logger
}

// New creates a Parser.
func New(client Getter, cfg *config.Config, log *slog.Logger) *Parser {
	return &Parser{client: client, cfg: cfg, log: log}
}

// FetchAll collects products for every configured category.
func (p *Parser) FetchAll() ([]model.Product, error) {
	var all []model.Product

	for _, cat := range p.cfg.Categories {
		p.log.Info("parsing category", slog.String("name", cat.Name), slog.String("slug", cat.Slug))

		products, err := p.FetchCategory(cat)
		if err != nil {
			// Log and continue — don't abort the whole run on one bad category
			p.log.Error("failed to fetch category",
				slog.String("category", cat.Name),
				slog.String("error", err.Error()),
			)
			continue
		}

		p.log.Info("category done",
			slog.String("name", cat.Name),
			slog.Int("count", len(products)),
		)
		all = append(all, products...)
	}

	if len(all) == 0 {
		return nil, fmt.Errorf("no products collected across all categories")
	}

	return all, nil
}

// FetchCategory paginates through a single category and returns all products.
func (p *Parser) FetchCategory(cat model.Category) ([]model.Product, error) {
	pageSize := p.cfg.HTTP.PageSize
	var products []model.Product
	offset := 0

	for {
		params := url.Values{}
		params.Set("store", p.client.StoreSlug())
		params.Set("category", cat.Slug)
		params.Set("limit", strconv.Itoa(pageSize))
		params.Set("offset", strconv.Itoa(offset))
		params.Set("sort", "popular")

		body, err := p.client.Get("/api/v1/catalog/sku/list", params)
		if err != nil {
			return nil, fmt.Errorf("page offset=%d: %w", offset, err)
		}

		var resp model.ProductsResponse
		if err := json.Unmarshal(body, &resp); err != nil {
			return nil, fmt.Errorf("json decode at offset=%d: %w", offset, err)
		}

		for _, sku := range resp.SKUs {
			products = append(products, p.skuToProduct(sku, cat))
		}

		p.log.Debug("page fetched",
			slog.String("category", cat.Name),
			slog.Int("offset", offset),
			slog.Int("fetched", len(products)),
			slog.Int("total", resp.TotalCount),
		)

		offset += pageSize
		if offset >= resp.TotalCount || len(resp.SKUs) == 0 {
			break
		}
	}

	return products, nil
}

// skuToProduct maps a raw API SKU to a domain Product.
func (p *Parser) skuToProduct(sku model.SKU, cat model.Category) model.Product {
	slug := sku.Slug
	if slug == "" {
		slug = sku.ID
	}

	productURL := fmt.Sprintf("%s/catalog/%s/sku/%s/", p.cfg.Store.BaseURL, cat.Slug, slug)

	return model.Product{
		Category: cat.Name,
		Name:     sku.Name,
		Price:    sku.Prices.EffectivePrice(),
		URL:      productURL,
	}
}
