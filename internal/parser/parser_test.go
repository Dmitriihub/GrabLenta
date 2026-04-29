package parser

import (
	"encoding/json"
	"fmt"
	"net/url"
	"testing"

	"github.com/yourorg/grablenta/internal/config"
	"github.com/yourorg/grablenta/internal/model"
	"log/slog"
	"os"
)

// mockClient implements Getter for testing — no real HTTP.
type mockClient struct {
	responses map[string][]byte // key: "offset" as string
	slug      string
	callCount int
}

func (m *mockClient) StoreSlug() string { return m.slug }

func (m *mockClient) Get(_ string, params url.Values) ([]byte, error) {
	offset := params.Get("offset")
	resp, ok := m.responses[offset]
	if !ok {
		return nil, fmt.Errorf("unexpected offset %q in mock", offset)
	}
	m.callCount++
	return resp, nil
}

func newTestLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
}

func makeResponse(skus []model.SKU, total int) []byte {
	r := model.ProductsResponse{SKUs: skus, TotalCount: total}
	b, _ := json.Marshal(r)
	return b
}

func testConfig() *config.Config {
	return &config.Config{
		Store: config.StoreConfig{
			Slug:    "test-store",
			BaseURL: "https://example.com",
		},
		HTTP: config.HTTPConfig{PageSize: 2},
		Categories: []model.Category{
			{Slug: "moloko", Name: "Молочка"},
		},
	}
}

func TestFetchCategory_SinglePage(t *testing.T) {
	skus := []model.SKU{
		{ID: "1", Name: "Молоко", Slug: "moloko-1l", Prices: model.Prices{Price: 89.9}},
		{ID: "2", Name: "Кефир", Slug: "kefir-1l", Prices: model.Prices{Price: 79.9}},
	}
	mc := &mockClient{
		slug: "test-store",
		responses: map[string][]byte{
			"0": makeResponse(skus, 2),
		},
	}
	p := New(mc, testConfig(), newTestLogger())
	cat := model.Category{Slug: "moloko", Name: "Молочка"}

	products, err := p.FetchCategory(cat)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(products) != 2 {
		t.Fatalf("expected 2 products, got %d", len(products))
	}
	if products[0].Name != "Молоко" {
		t.Errorf("expected first product 'Молоко', got %q", products[0].Name)
	}
	if products[0].Price != 89.9 {
		t.Errorf("expected price 89.9, got %v", products[0].Price)
	}
	if products[0].URL != "https://example.com/catalog/moloko/sku/moloko-1l/" {
		t.Errorf("unexpected URL: %s", products[0].URL)
	}
}

func TestFetchCategory_Pagination(t *testing.T) {
	page1 := []model.SKU{
		{ID: "1", Name: "Товар 1", Prices: model.Prices{Price: 10}},
		{ID: "2", Name: "Товар 2", Prices: model.Prices{Price: 20}},
	}
	page2 := []model.SKU{
		{ID: "3", Name: "Товар 3", Prices: model.Prices{Price: 30}},
	}
	mc := &mockClient{
		slug: "test-store",
		responses: map[string][]byte{
			"0": makeResponse(page1, 3),
			"2": makeResponse(page2, 3),
		},
	}
	p := New(mc, testConfig(), newTestLogger())
	cat := model.Category{Slug: "test", Name: "Тест"}

	products, err := p.FetchCategory(cat)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(products) != 3 {
		t.Fatalf("expected 3 products, got %d", len(products))
	}
	if mc.callCount != 2 {
		t.Errorf("expected 2 API calls (pagination), got %d", mc.callCount)
	}
}

func TestFetchCategory_FallsBackToIDWhenNoSlug(t *testing.T) {
	skus := []model.SKU{
		{ID: "abc123", Name: "Без слага", Prices: model.Prices{Price: 50}},
	}
	mc := &mockClient{
		slug:      "test-store",
		responses: map[string][]byte{"0": makeResponse(skus, 1)},
	}
	p := New(mc, testConfig(), newTestLogger())
	cat := model.Category{Slug: "cat", Name: "Кат"}

	products, err := p.FetchCategory(cat)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if products[0].URL != "https://example.com/catalog/cat/sku/abc123/" {
		t.Errorf("unexpected URL: %s", products[0].URL)
	}
}

func TestFetchCategory_NetworkError(t *testing.T) {
	mc := &mockClient{
		slug:      "test-store",
		responses: map[string][]byte{}, // no valid responses → error
	}
	p := New(mc, testConfig(), newTestLogger())
	cat := model.Category{Slug: "bad", Name: "Bad"}

	_, err := p.FetchCategory(cat)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestFetchAll_SkipsBadCategoryAndContinues(t *testing.T) {
	cfg := testConfig()
	cfg.Categories = []model.Category{
		{Slug: "bad-cat", Name: "Ломаная"},
		{Slug: "good-cat", Name: "Рабочая"},
	}

	goodSKU := []model.SKU{{ID: "1", Name: "Норм товар", Prices: model.Prices{Price: 55}}}
	mc := &mockClient{
		slug: "test-store",
		responses: map[string][]byte{
			// Only "good-cat" at offset 0 has a response; "bad-cat" will error
			"0": makeResponse(goodSKU, 1),
		},
	}

	// Swap mock to intercept by category — simulate via custom mock
	mc2 := &categoryMockClient{
		slug: "test-store",
		data: map[string][]byte{
			"good-cat:0": makeResponse(goodSKU, 1),
		},
	}

	p := New(mc2, cfg, newTestLogger())
	products, err := p.FetchAll()
	if err != nil {
		t.Fatalf("FetchAll should not fail when at least one category succeeds, got: %v", err)
	}
	if len(products) != 1 {
		t.Fatalf("expected 1 product from good category, got %d", len(products))
	}
}

// categoryMockClient routes by category slug + offset.
type categoryMockClient struct {
	slug string
	data map[string][]byte
}

func (c *categoryMockClient) StoreSlug() string { return c.slug }

func (c *categoryMockClient) Get(_ string, params url.Values) ([]byte, error) {
	key := params.Get("category") + ":" + params.Get("offset")
	resp, ok := c.data[key]
	if !ok {
		return nil, fmt.Errorf("no mock data for key %q", key)
	}
	return resp, nil
}
