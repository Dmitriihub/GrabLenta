package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
	"log/slog"
	"os"
)

func noopLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
}

func TestHeaders_InjectsDefaults(t *testing.T) {
	var got http.Header

	base := RoundTripFunc(func(req *http.Request) (*http.Response, error) {
		got = req.Header.Clone()
		return &http.Response{StatusCode: 200, Body: http.NoBody}, nil
	})

	rt := Headers(map[string]string{"X-Test": "hello"})(base)

	req := httptest.NewRequest(http.MethodGet, "http://example.com", nil)
	rt.RoundTrip(req)

	if got.Get("X-Test") != "hello" {
		t.Errorf("expected X-Test: hello, got %q", got.Get("X-Test"))
	}
}

func TestHeaders_DoesNotOverwriteExisting(t *testing.T) {
	var got http.Header

	base := RoundTripFunc(func(req *http.Request) (*http.Response, error) {
		got = req.Header.Clone()
		return &http.Response{StatusCode: 200, Body: http.NoBody}, nil
	})

	rt := Headers(map[string]string{"X-Custom": "default"})(base)

	req := httptest.NewRequest(http.MethodGet, "http://example.com", nil)
	req.Header.Set("X-Custom", "already-set")
	rt.RoundTrip(req)

	if got.Get("X-Custom") != "already-set" {
		t.Errorf("header should not be overwritten, got %q", got.Get("X-Custom"))
	}
}

func TestRetry_RetriesOnServerError(t *testing.T) {
	attempts := 0

	base := RoundTripFunc(func(req *http.Request) (*http.Response, error) {
		attempts++
		if attempts < 3 {
			return &http.Response{StatusCode: 503, Body: http.NoBody}, nil
		}
		return &http.Response{StatusCode: 200, Body: http.NoBody}, nil
	})

	rt := Retry(3, 10*time.Millisecond, noopLogger())(base)

	req := httptest.NewRequest(http.MethodGet, "http://example.com", nil)
	resp, err := rt.RoundTrip(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
	if attempts != 3 {
		t.Errorf("expected 3 attempts, got %d", attempts)
	}
}

func TestRetry_FailsAfterMaxAttempts(t *testing.T) {
	base := RoundTripFunc(func(req *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 429, Body: http.NoBody}, nil
	})

	rt := Retry(3, 10*time.Millisecond, noopLogger())(base)

	req := httptest.NewRequest(http.MethodGet, "http://example.com", nil)
	_, err := rt.RoundTrip(req)
	if err == nil {
		t.Fatal("expected error after max retries, got nil")
	}
}

func TestChain_OrderIsCorrect(t *testing.T) {
	var order []string

	base := RoundTripFunc(func(req *http.Request) (*http.Response, error) {
		order = append(order, "base")
		return &http.Response{StatusCode: 200, Body: http.NoBody}, nil
	})

	wrap := func(name string) func(http.RoundTripper) http.RoundTripper {
		return func(next http.RoundTripper) http.RoundTripper {
			return RoundTripFunc(func(req *http.Request) (*http.Response, error) {
				order = append(order, name+":before")
				resp, err := next.RoundTrip(req)
				order = append(order, name+":after")
				return resp, err
			})
		}
	}

	rt := Chain(base, wrap("A"), wrap("B"))
	req := httptest.NewRequest(http.MethodGet, "http://example.com", nil)
	rt.RoundTrip(req)

	// A wraps B which wraps base
	expected := []string{"A:before", "B:before", "base", "B:after", "A:after"}
	for i, v := range expected {
		if order[i] != v {
			t.Errorf("chain order[%d]: expected %q, got %q", i, v, order[i])
		}
	}
}
