package main

import (
	"flag"
	"log"
	"os"

	"github.com/yourorg/grablenta/internal/client"
	"github.com/yourorg/grablenta/internal/config"
	"github.com/yourorg/grablenta/internal/exporter"
	"github.com/yourorg/grablenta/internal/parser"
	"github.com/yourorg/grablenta/pkg/logger"
)

func main() {
	cfgPath := flag.String("config", "config.yaml", "path to config file")
	flag.Parse()

	// ── Config ────────────────────────────────────────────────────────────────
	cfg, err := config.Load(*cfgPath)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	// ── Logger ────────────────────────────────────────────────────────────────
	slogger, closeLog, err := logger.New(cfg.Log.Level, cfg.Log.File)
	if err != nil {
		log.Fatalf("init logger: %v", err)
	}
	defer closeLog()

	// ── HTTP client ───────────────────────────────────────────────────────────
	httpClient, err := client.New(cfg, slogger)
	if err != nil {
		slogger.Error("build http client", "error", err)
		os.Exit(1)
	}

	// ── Parser ────────────────────────────────────────────────────────────────
	p := parser.New(httpClient, cfg, slogger)

	products, err := p.FetchAll()
	if err != nil {
		slogger.Error("fetch products", "error", err)
		os.Exit(1)
	}

	// ── Export ────────────────────────────────────────────────────────────────
	exp := exporter.NewCSV(cfg.Output.File, cfg.Output.Delimiter)

	n, err := exp.Export(products)
	if err != nil {
		slogger.Error("export csv", "error", err)
		os.Exit(1)
	}

	slogger.Info("done", "products", n, "file", cfg.Output.File)
}
