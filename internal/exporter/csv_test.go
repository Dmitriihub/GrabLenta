package exporter

import (
	"encoding/csv"
	"os"
	"path/filepath"
	"testing"

	"github.com/yourorg/grablenta/internal/model"
)

func sampleProducts() []model.Product {
	return []model.Product{
		{Category: "Молочка", Name: "Молоко 1л", Price: 89.90, URL: "https://lenta.com/sku/1/"},
		{Category: "Хлеб", Name: "Батон", Price: 44.90, URL: "https://lenta.com/sku/2/"},
	}
}

func TestCSVExporter_WritesHeaderAndRows(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "products.csv")

	exp := NewCSV(out, ";")
	n, err := exp.Export(sampleProducts())
	if err != nil {
		t.Fatalf("export: %v", err)
	}
	if n != 2 {
		t.Errorf("expected 2 rows, got %d", n)
	}

	// Parse back and verify
	f, err := os.Open(out)
	if err != nil {
		t.Fatalf("open output: %v", err)
	}
	defer f.Close()

	// Skip BOM
	bom := make([]byte, 3)
	f.Read(bom)

	r := csv.NewReader(f)
	r.Comma = ';'

	records, err := r.ReadAll()
	if err != nil {
		t.Fatalf("read csv: %v", err)
	}

	// 1 header + 2 data rows
	if len(records) != 3 {
		t.Fatalf("expected 3 records (header+2), got %d", len(records))
	}

	header := records[0]
	if header[0] != "Категория" {
		t.Errorf("expected 'Категория' header, got %q", header[0])
	}

	row1 := records[1]
	if row1[1] != "Молоко 1л" {
		t.Errorf("expected 'Молоко 1л', got %q", row1[1])
	}
	if row1[2] != "89.90" {
		t.Errorf("expected price '89.90', got %q", row1[2])
	}
}

func TestCSVExporter_EmptyProductsReturnsError(t *testing.T) {
	exp := NewCSV("/tmp/empty_test.csv", ";")
	_, err := exp.Export(nil)
	if err == nil {
		t.Fatal("expected error for empty products, got nil")
	}
}

func TestCSVExporter_CustomDelimiter(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "comma.csv")

	exp := NewCSV(out, ",")
	_, err := exp.Export(sampleProducts())
	if err != nil {
		t.Fatalf("export: %v", err)
	}

	f, _ := os.Open(out)
	defer f.Close()
	bom := make([]byte, 3)
	f.Read(bom)

	r := csv.NewReader(f)
	r.Comma = ','
	records, err := r.ReadAll()
	if err != nil {
		t.Fatalf("read csv with comma delimiter: %v", err)
	}
	if len(records) != 3 {
		t.Fatalf("expected 3 records, got %d", len(records))
	}
}
