package config

import (
	"os"
	"path/filepath"
	"testing"
)

func writeTemp(t *testing.T, content string) string {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "config*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	f.WriteString(content)
	f.Close()
	return f.Name()
}

const validYAML = `
store:
  slug: "test-store"
  base_url: "https://lenta.com"
categories:
  - slug: "moloko"
    name: "Молочка"
http:
  timeout_seconds: 30
  page_size: 48
  delay_min_ms: 100
  delay_max_ms: 500
  retry_count: 3
  retry_wait_ms: 1000
proxy:
  url: ""
output:
  file: "products.csv"
  delimiter: ";"
log:
  level: "info"
  file: ""
`

func TestLoad_ValidConfig(t *testing.T) {
	path := writeTemp(t, validYAML)
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Store.Slug != "test-store" {
		t.Errorf("expected slug 'test-store', got %q", cfg.Store.Slug)
	}
	if len(cfg.Categories) != 1 {
		t.Errorf("expected 1 category, got %d", len(cfg.Categories))
	}
}

func TestLoad_MissingFile(t *testing.T) {
	_, err := Load(filepath.Join(t.TempDir(), "nonexistent.yaml"))
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestLoad_MissingSlug(t *testing.T) {
	yaml := `
store:
  slug: ""
  base_url: "https://lenta.com"
categories:
  - slug: "x"
    name: "X"
`
	path := writeTemp(t, yaml)
	_, err := Load(path)
	if err == nil {
		t.Fatal("expected validation error for empty slug")
	}
}

func TestLoad_NoCategories(t *testing.T) {
	yaml := `
store:
  slug: "test"
  base_url: "https://lenta.com"
categories: []
`
	path := writeTemp(t, yaml)
	_, err := Load(path)
	if err == nil {
		t.Fatal("expected validation error for empty categories")
	}
}

func TestLoad_ProxyEnvOverride(t *testing.T) {
	path := writeTemp(t, validYAML)
	t.Setenv("PROXY_URL", "http://proxy:8080")

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Proxy.URL != "http://proxy:8080" {
		t.Errorf("expected proxy from env, got %q", cfg.Proxy.URL)
	}
}
