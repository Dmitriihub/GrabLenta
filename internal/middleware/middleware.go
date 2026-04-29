package middleware

import (
	"fmt"
	"log/slog"
	"math/rand"
	"net/http"
	"time"
)

// RoundTripFunc is an adapter to allow use of ordinary functions as http.RoundTripper.
type RoundTripFunc func(req *http.Request) (*http.Response, error)

func (f RoundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

// Chain wraps a base RoundTripper with a slice of middleware in order.
// The first middleware is the outermost (applied first to request, last to response).
func Chain(base http.RoundTripper, middlewares ...func(http.RoundTripper) http.RoundTripper) http.RoundTripper {
	rt := base
	for i := len(middlewares) - 1; i >= 0; i-- {
		rt = middlewares[i](rt)
	}
	return rt
}

// Logging logs every outgoing request and its response status/duration.
func Logging(log *slog.Logger) func(http.RoundTripper) http.RoundTripper {
	return func(next http.RoundTripper) http.RoundTripper {
		return RoundTripFunc(func(req *http.Request) (*http.Response, error) {
			start := time.Now()
			resp, err := next.RoundTrip(req)
			dur := time.Since(start)

			if err != nil {
				log.Error("http request failed",
					slog.String("method", req.Method),
					slog.String("url", req.URL.String()),
					slog.Duration("duration", dur),
					slog.String("error", err.Error()),
				)
				return nil, err
			}

			log.Debug("http request",
				slog.String("method", req.Method),
				slog.String("url", req.URL.String()),
				slog.Int("status", resp.StatusCode),
				slog.Duration("duration", dur),
			)
			return resp, nil
		})
	}
}

// Retry retries on transient errors and 429/5xx responses.
// It uses exponential back-off with jitter.
func Retry(attempts int, baseWait time.Duration, log *slog.Logger) func(http.RoundTripper) http.RoundTripper {
	return func(next http.RoundTripper) http.RoundTripper {
		return RoundTripFunc(func(req *http.Request) (*http.Response, error) {
			var (
				resp *http.Response
				err  error
			)
			for i := 0; i < attempts; i++ {
				// Clone request so body can be re-read on retry
				cloned := req.Clone(req.Context())

				resp, err = next.RoundTrip(cloned)
				if err == nil && !isRetryable(resp.StatusCode) {
					return resp, nil
				}

				status := 0
				if resp != nil {
					status = resp.StatusCode
					resp.Body.Close()
				}

				if i == attempts-1 {
					break
				}

				wait := jitter(baseWait * (1 << i)) // exponential back-off
				log.Warn("retrying request",
					slog.Int("attempt", i+1),
					slog.Int("status", status),
					slog.Duration("wait", wait),
					slog.String("url", req.URL.String()),
				)
				time.Sleep(wait)
			}

			if err != nil {
				return nil, err
			}
			if isRetryable(resp.StatusCode) {
				return nil, fmt.Errorf("request failed after %d attempts, last status: %d", attempts, resp.StatusCode)
			}
			return resp, nil
		})
	}
}

// Headers injects default headers into every request.
func Headers(defaults map[string]string) func(http.RoundTripper) http.RoundTripper {
	return func(next http.RoundTripper) http.RoundTripper {
		return RoundTripFunc(func(req *http.Request) (*http.Response, error) {
			// Clone to avoid mutating the original
			cloned := req.Clone(req.Context())
			for k, v := range defaults {
				if cloned.Header.Get(k) == "" {
					cloned.Header.Set(k, v)
				}
			}
			return next.RoundTrip(cloned)
		})
	}
}

// RateLimit adds a random delay between [minDelay, maxDelay] before each request.
func RateLimit(minDelay, maxDelay time.Duration) func(http.RoundTripper) http.RoundTripper {
	return func(next http.RoundTripper) http.RoundTripper {
		return RoundTripFunc(func(req *http.Request) (*http.Response, error) {
			if minDelay > 0 {
				d := minDelay
				if maxDelay > minDelay {
					d = minDelay + time.Duration(rand.Int63n(int64(maxDelay-minDelay)))
				}
				time.Sleep(d)
			}
			return next.RoundTrip(req)
		})
	}
}

func isRetryable(status int) bool {
	return status == http.StatusTooManyRequests ||
		status == http.StatusServiceUnavailable ||
		status == http.StatusBadGateway ||
		status == http.StatusGatewayTimeout ||
		status >= 500
}

func jitter(d time.Duration) time.Duration {
	jit := time.Duration(rand.Int63n(int64(d / 2)))
	return d + jit
}
