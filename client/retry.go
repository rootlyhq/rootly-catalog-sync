package client

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"math"
	"math/rand"
	"net/http"
	"strconv"
	"time"
)

// retryHTTPClient wraps an *http.Client with retry logic.
// It implements the rootly.HttpRequestDoer interface (Do(*http.Request) (*http.Response, error)).
type retryHTTPClient struct {
	inner      *http.Client
	maxRetries int
}

func (rc *retryHTTPClient) Do(req *http.Request) (*http.Response, error) {
	ctx := req.Context()

	// Buffer the request body so it can be replayed on retries.
	var bodyBytes []byte
	if req.Body != nil {
		var err error
		bodyBytes, err = io.ReadAll(req.Body)
		if err != nil {
			return nil, fmt.Errorf("reading request body for retry: %w", err)
		}
		_ = req.Body.Close()
	}

	var lastErr error
	for attempt := 0; attempt <= rc.maxRetries; attempt++ {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		// Reset the body for each attempt.
		if bodyBytes != nil {
			req.Body = io.NopCloser(bytes.NewReader(bodyBytes))
			req.ContentLength = int64(len(bodyBytes))
		}

		resp, err := rc.inner.Do(req)
		if err != nil {
			lastErr = err
			if attempt < rc.maxRetries {
				sleepWithJitter(ctx, attempt)
				continue
			}
			return nil, fmt.Errorf("after %d retries: %w", rc.maxRetries, lastErr)
		}

		if !isRetryable(resp.StatusCode) {
			return resp, nil
		}

		_ = resp.Body.Close()
		lastErr = fmt.Errorf("HTTP %d", resp.StatusCode)

		if attempt < rc.maxRetries {
			wait := retryDelay(resp, attempt)
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(wait):
			}
		}
	}

	return nil, fmt.Errorf("after %d retries: %w", rc.maxRetries, lastErr)
}

func isRetryable(status int) bool {
	switch status {
	case http.StatusTooManyRequests,
		http.StatusInternalServerError,
		http.StatusBadGateway,
		http.StatusServiceUnavailable,
		http.StatusGatewayTimeout:
		return true
	}
	return false
}

func retryDelay(resp *http.Response, attempt int) time.Duration {
	if resp.StatusCode == http.StatusTooManyRequests {
		if ra := resp.Header.Get("Retry-After"); ra != "" {
			if secs, err := strconv.Atoi(ra); err == nil && secs > 0 {
				return time.Duration(secs) * time.Second
			}
		}
	}
	return backoffWithJitter(attempt)
}

func backoffWithJitter(attempt int) time.Duration {
	base := math.Pow(2, float64(attempt)) * 500
	jitter := rand.Float64() * base * 0.5
	return time.Duration(base+jitter) * time.Millisecond
}

func sleepWithJitter(ctx context.Context, attempt int) {
	select {
	case <-ctx.Done():
	case <-time.After(backoffWithJitter(attempt)):
	}
}
