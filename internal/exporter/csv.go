package exporter

import (
	"encoding/csv"
	"fmt"
	"os"
	"strconv"

	"github.com/yourorg/grablenta/internal/model"
)

// CSVExporter writes products to a CSV file.
type CSVExporter struct {
	filePath  string
	delimiter rune
}

// NewCSV creates a CSVExporter.
// delimiter is typically ';' (Excel-friendly) or ','.
func NewCSV(filePath string, delimiter string) *CSVExporter {
	d := ';'
	if len(delimiter) > 0 {
		d = rune(delimiter[0])
	}
	return &CSVExporter{filePath: filePath, delimiter: d}
}

// Export writes all products to the configured CSV file.
// It returns the number of rows written.
func (e *CSVExporter) Export(products []model.Product) (int, error) {
	if len(products) == 0 {
		return 0, fmt.Errorf("no products to export")
	}

	f, err := os.Create(e.filePath)
	if err != nil {
		return 0, fmt.Errorf("create file %q: %w", e.filePath, err)
	}
	defer f.Close()

	// UTF-8 BOM — required for correct display in Microsoft Excel
	if _, err := f.WriteString("\xEF\xBB\xBF"); err != nil {
		return 0, fmt.Errorf("write BOM: %w", err)
	}

	w := csv.NewWriter(f)
	w.Comma = e.delimiter

	header := []string{"Категория", "Наименование товара", "Цена (руб.)", "Ссылка"}
	if err := w.Write(header); err != nil {
		return 0, fmt.Errorf("write header: %w", err)
	}

	for _, p := range products {
		row := []string{
			p.Category,
			p.Name,
			strconv.FormatFloat(p.Price, 'f', 2, 64),
			p.URL,
		}
		if err := w.Write(row); err != nil {
			return 0, fmt.Errorf("write row: %w", err)
		}
	}

	w.Flush()
	if err := w.Error(); err != nil {
		return 0, fmt.Errorf("flush csv: %w", err)
	}

	return len(products), nil
}
