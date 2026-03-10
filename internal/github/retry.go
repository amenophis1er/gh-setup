package github

import (
	"bytes"
	"io"
	"math"
	"net/http"
	"strconv"
	"time"

	"github.com/charmbracelet/log"
)

// retryTransport wraps an http.RoundTripper with automatic retry, exponential
// backoff, and GitHub rate-limit awareness.
type retryTransport struct {
	base       http.RoundTripper
	maxRetries int
}

func newRetryTransport(base http.RoundTripper, maxRetries int) *retryTransport {
	if base == nil {
		base = http.DefaultTransport
	}
	if maxRetries < 1 {
		maxRetries = 3
	}
	return &retryTransport{base: base, maxRetries: maxRetries}
}

func (t *retryTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	var resp *http.Response
	var err error

	// Buffer the body so we can replay it on retries.
	var bodyBytes []byte
	if req.Body != nil {
		bodyBytes, err = io.ReadAll(req.Body)
		if err != nil {
			return nil, err
		}
		req.Body.Close()
	}

	for attempt := 0; attempt <= t.maxRetries; attempt++ {
		// Reset body for each attempt.
		if bodyBytes != nil {
			req.Body = io.NopCloser(bytes.NewReader(bodyBytes))
			req.ContentLength = int64(len(bodyBytes))
		}

		resp, err = t.base.RoundTrip(req)
		if err != nil {
			// Network error — retry with backoff.
			if attempt < t.maxRetries {
				wait := backoff(attempt)
				log.Debug("Retrying after network error", "attempt", attempt+1, "wait", wait, "err", err)
				time.Sleep(wait)
				continue
			}
			return nil, err
		}

		if !shouldRetry(resp) {
			return resp, nil
		}

		wait := retryDelay(resp, attempt)
		log.Debug("Retrying request", "status", resp.StatusCode, "attempt", attempt+1, "wait", wait)

		// Drain and close the body so the connection can be reused.
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()

		if attempt < t.maxRetries {
			time.Sleep(wait)
		}
	}

	return resp, err
}

func shouldRetry(resp *http.Response) bool {
	switch {
	case resp.StatusCode == http.StatusTooManyRequests: // 429
		return true
	case resp.StatusCode == http.StatusForbidden && isRateLimited(resp):
		return true
	case resp.StatusCode >= 500:
		return true
	default:
		return false
	}
}

func isRateLimited(resp *http.Response) bool {
	return resp.Header.Get("X-RateLimit-Remaining") == "0" ||
		resp.Header.Get("Retry-After") != ""
}

func retryDelay(resp *http.Response, attempt int) time.Duration {
	// Respect Retry-After header (secondary rate limit / abuse detection).
	if ra := resp.Header.Get("Retry-After"); ra != "" {
		if seconds, err := strconv.Atoi(ra); err == nil {
			return time.Duration(seconds) * time.Second
		}
	}

	// Primary rate limit — wait until reset time.
	if reset := resp.Header.Get("X-RateLimit-Reset"); reset != "" {
		if ts, err := strconv.ParseInt(reset, 10, 64); err == nil {
			wait := time.Until(time.Unix(ts, 0))
			if wait > 0 {
				return wait
			}
		}
	}

	// Fallback: exponential backoff.
	return backoff(attempt)
}

func backoff(attempt int) time.Duration {
	return time.Duration(math.Pow(2, float64(attempt))) * time.Second
}
