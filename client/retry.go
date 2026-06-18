package client

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"net/http"
	"strconv"
	"time"
)

func (c *Client) doWithRetry(ctx context.Context, req *http.Request) (*http.Response, error) {
	var lastErr error

	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = err
			if attempt < c.maxRetries {
				sleepWithJitter(ctx, attempt)
				continue
			}
			return nil, fmt.Errorf("after %d retries: %w", c.maxRetries, lastErr)
		}

		if !isRetryable(resp.StatusCode) {
			return resp, nil
		}

		_ = resp.Body.Close()
		lastErr = fmt.Errorf("HTTP %d", resp.StatusCode)

		if attempt < c.maxRetries {
			wait := retryDelay(resp, attempt)
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(wait):
			}
		}
	}

	return nil, fmt.Errorf("after %d retries: %w", c.maxRetries, lastErr)
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
