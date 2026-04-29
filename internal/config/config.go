package config

import (
	"fmt"
	"os"
	"time"

	"github.com/yourorg/grablenta/internal/model"
	"gopkg.in/yaml.v3"
)

// Config is the root configuration structure.
type Config struct {
	Store      StoreConfig  `yaml:"store"`
	Categories []model.Category `yaml:"categories"`
	HTTP       HTTPConfig   `yaml:"http"`
	Proxy      ProxyConfig  `yaml:"proxy"`
	Output     OutputConfig `yaml:"output"`
	Log        LogConfig    `yaml:"log"`
}

type StoreConfig struct {
	Slug    string `yaml:"slug"`
	BaseURL string `yaml:"base_url"`
}

type HTTPConfig struct {
	TimeoutSeconds int `yaml:"timeout_seconds"`
	PageSize       int `yaml:"page_size"`
	DelayMinMs     int `yaml:"delay_min_ms"`
	DelayMaxMs     int `yaml:"delay_max_ms"`
	RetryCount     int `yaml:"retry_count"`
	RetryWaitMs    int `yaml:"retry_wait_ms"`
}

type ProxyConfig struct {
	URL string `yaml:"url"`
}

type OutputConfig struct {
	File      string `yaml:"file"`
	Delimiter string `yaml:"delimiter"`
}

type LogConfig struct {
	Level string `yaml:"level"`
	File  string `yaml:"file"`
}

// Timeout returns HTTP timeout as time.Duration.
func (h HTTPConfig) Timeout() time.Duration {
	return time.Duration(h.TimeoutSeconds) * time.Second
}

// RetryWait returns retry wait as time.Duration.
func (h HTTPConfig) RetryWait() time.Duration {
	return time.Duration(h.RetryWaitMs) * time.Millisecond
}

// Load reads config from a YAML file.
// It also honours environment variable overrides:
//   PROXY_URL overrides proxy.url
func Load(path string) (*Config, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open config %q: %w", path, err)
	}
	defer f.Close()

	var cfg Config
	if err := yaml.NewDecoder(f).Decode(&cfg); err != nil {
		return nil, fmt.Errorf("decode config: %w", err)
	}

	// Environment variable overrides
	if v := os.Getenv("PROXY_URL"); v != "" {
		cfg.Proxy.URL = v
	}
	if v := os.Getenv("STORE_SLUG"); v != "" {
		cfg.Store.Slug = v
	}

	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	return &cfg, nil
}

func (c *Config) validate() error {
	if c.Store.Slug == "" {
		return fmt.Errorf("store.slug is required")
	}
	if c.Store.BaseURL == "" {
		return fmt.Errorf("store.base_url is required")
	}
	if len(c.Categories) == 0 {
		return fmt.Errorf("at least one category is required")
	}
	if c.HTTP.PageSize <= 0 {
		c.HTTP.PageSize = 48
	}
	if c.HTTP.RetryCount <= 0 {
		c.HTTP.RetryCount = 3
	}
	return nil
}
