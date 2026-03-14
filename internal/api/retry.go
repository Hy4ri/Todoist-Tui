package api

import (
	"math"
	"time"
)

// doWithRetry performs an HTTP request with exponential backoff retry logic.
//
// Retries are performed for:
//   - Network errors (connection refused, timeout, etc.)
//   - HTTP 429 Too Many Requests (rate limited)
//   - HTTP 5xx Server Errors
//
// Client errors (4xx except 429) are NOT retried — they are application bugs
// that won't resolve on their own.
//
// Backoff schedule: 500ms → 1s → 2s (doubles each attempt, capped for UX).
func (c *Client) doWithRetry(method, path string, body interface{}, result interface{}) error {
	maxRetries := c.MaxRetries
	if maxRetries <= 0 {
		maxRetries = 3
	}
	baseDelay := c.RetryBaseDelay
	if baseDelay <= 0 {
		baseDelay = 500 * time.Millisecond
	}

	var lastErr error
	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			// Exponential backoff: 500ms, 1s, 2s, ...
			delay := time.Duration(float64(baseDelay) * math.Pow(2, float64(attempt-1)))
			time.Sleep(delay)
		}

		err := c.do(method, path, body, result)
		if err == nil {
			return nil
		}

		lastErr = err

		// Only retry on transient errors
		if !isRetryable(err) {
			return err
		}
	}

	return lastErr
}

// isRetryable returns true if the error warrants a retry.
// Network errors are always retryable; API errors retry only on 429 and 5xx.
func isRetryable(err error) bool {
	if err == nil {
		return false
	}

	// Check for API errors (HTTP status codes)
	apiErr, ok := IsAPIError(err)
	if !ok {
		// Non-API error = network/transport error → always retry
		return true
	}

	return apiErr.IsRateLimited() || apiErr.IsServerError()
}
