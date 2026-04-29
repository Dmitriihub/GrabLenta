package client

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"time"

	"github.com/yourorg/grablenta/internal/config"
	"github.com/yourorg/grablenta/internal/middleware"
)

const userAgent = "Mozilla/5.0 (Linux; Android 13; Pixel 7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Mobile Safari/537.36"

// Client wraps http.Client with store-aware helpers.
type Client struct {
	http      *http.Client
	baseURL   string
	storeSlug string
	log       *slog.Logger
}

// New builds a Client from config, wiring proxy and the full middleware stack.
func New(cfg *config.Config, log *slog.Logger) (*Client, error) {
	transport, err := buildTransport(cfg.Proxy.URL)
	if err != nil {
		return nil, fmt.Errorf("build transport: %w", err)
	}

	minDelay := time.Duration(cfg.HTTP.DelayMinMs) * time.Millisecond
	maxDelay := time.Duration(cfg.HTTP.DelayMaxMs) * time.Millisecond

	// Middleware is applied outermost-first: Logging → Retry → RateLimit → Headers → transport
	chained := middleware.Chain(
		transport,
		middleware.Logging(log),
		middleware.Retry(cfg.HTTP.RetryCount, cfg.HTTP.RetryWait(), log),
		middleware.RateLimit(minDelay, maxDelay),
		middleware.Headers(map[string]string{
			"User-Agent":      userAgent,
			"Accept":          "application/json",
			"Accept-Language": "ru-RU,ru;q=0.9",
			"Referer":         cfg.Store.BaseURL + "/",
			"X-Store-Slug":    cfg.Store.Slug,
		}),
	)

	httpClient := &http.Client{
		Transport: chained,
		Timeout:   cfg.HTTP.Timeout(),
	}

	if cfg.Proxy.URL != "" {
		log.Info("proxy configured", slog.String("proxy", maskProxyURL(cfg.Proxy.URL)))
	} else {
		log.Warn("no proxy configured — running without proxy")
	}

	return &Client{
		http:      httpClient,
		baseURL:   cfg.Store.BaseURL,
		storeSlug: cfg.Store.Slug,
		log:       log,
	}, nil
}

// Get performs a GET request to baseURL+path with the given query params
// and returns the raw response body.
func (c *Client) Get(path string, params url.Values) ([]byte, error) {
	fullURL := c.baseURL + path
	if len(params) > 0 {
		fullURL += "?" + params.Encode()
	}

	req, err := http.NewRequest(http.MethodGet, fullURL, nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status %d for %s", resp.StatusCode, fullURL)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}

	return body, nil
}

// StoreSlug returns the configured store slug.
func (c *Client) StoreSlug() string { return c.storeSlug }

// buildTransport creates an http.Transport, optionally with a proxy.
func buildTransport(proxyAddr string) (*http.Transport, error) {
	t := &http.Transport{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 10,
		IdleConnTimeout:     90 * time.Second,
	}

	if proxyAddr != "" {
		proxyURL, err := url.Parse(proxyAddr)
		if err != nil {
			return nil, fmt.Errorf("parse proxy URL: %w", err)
		}
		t.Proxy = http.ProxyURL(proxyURL)
	}

	return t, nil
}

// maskProxyURL hides credentials in log output.
func maskProxyURL(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return "invalid-url"
	}
	u.User = url.User("***")
	return u.String()
}
