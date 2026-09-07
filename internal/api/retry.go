// internal/api/retry.go
package api

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// transientMessages match failures that are retryable whatever status they
// arrive with: the Streamkap API answers 500 for Kafka Connect problems that
// clear on their own, and transport failures carry no status at all.
//
// Every pattern is a word, never a bare status number. Matching "429"/"502"
// against the message text used to retry any error whose detail happened to
// contain those digits ("port 4290 invalid") for minutes — status classes are
// now read off APIError instead.
var transientMessages = []string{
	// The backend exhausted its own Kafka Connect retries.
	"kafkaconnecttimeout",    // Backend's custom timeout exception
	"request timed out",      // KC timeout message
	"timed out on all nodes", // Backend exhausted all KC servers
	"sockettimeoutexception", // Java socket timeout

	// Kafka Connect transient states, reported inside a 500 detail
	// ("Kafka Connect API call failed with status code NNN and response ...").
	"rebalance_in_progress",
	"rebalance is expected", // KC 500: "Request cannot be completed because a rebalance is expected"
	"leader_not_available",
	"not_leader_for_partition",
	"kafka connect api call failed with status code 404", // Connector not yet deployed (PENDING destination/source)
	"kafka connect api call failed with status code 409", // Conflict during concurrent connector updates

	// Gateway wording embedded in a 5xx detail. A non-2xx always becomes an
	// APIError, so a real 429/502/503/504 is already caught by status; these
	// only fire when a proxy's wording is relayed inside another status' body.
	"too many requests",
	"bad gateway",
	"service unavailable",
	"gateway timeout",

	// Transport errors on the way to the Streamkap API.
	"connection refused",
	"connection reset",
	"no such host",
	"network unreachable",
	"i/o timeout",
}

// isRetryableStatus reports the statuses that are always worth another attempt:
// the caller is throttled, or an intermediary failed to reach the API at all.
// 5xx is deliberately not blanket-retryable — a plain 500 is a backend bug, and
// only the transient details above justify hammering it again.
func isRetryableStatus(code int) bool {
	switch code {
	case http.StatusTooManyRequests,
		http.StatusBadGateway,
		http.StatusServiceUnavailable,
		http.StatusGatewayTimeout:
		return true
	default:
		return false
	}
}

func matchesTransientMessage(msg string) bool {
	msg = strings.ToLower(msg)
	for _, pattern := range transientMessages {
		if strings.Contains(msg, pattern) {
			return true
		}
	}
	return false
}

// IsRetryableError checks if an error is transient and should be retried.
// Note: The Streamkap backend already retries Kafka Connect operations internally
// (tries multiple KC servers on ReadTimeout). This function identifies errors
// that indicate the backend exhausted its retries OR infrastructure issues.
func IsRetryableError(err error) bool {
	if err == nil {
		return false
	}

	var apiErr *APIError
	if errors.As(err, &apiErr) {
		if isRetryableStatus(apiErr.StatusCode) {
			return true
		}
		return matchesTransientMessage(apiErr.Detail)
	}

	return matchesTransientMessage(err.Error())
}

// RetryConfig holds retry configuration
type RetryConfig struct {
	MaxRetries int
	MinDelay   time.Duration
	MaxDelay   time.Duration
}

// DefaultRetryConfig returns sensible defaults for Streamkap API operations.
// Uses conservative delays because the backend already retries KC operations.
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxRetries: 5,
		MinDelay:   10 * time.Second, // Conservative: backend may be retrying
		MaxDelay:   60 * time.Second, // Cap to avoid excessive waits
	}
}

// RetryWithBackoff retries an operation with exponential backoff.
// Only retries transient errors; validation/auth errors fail immediately.
func RetryWithBackoff(ctx context.Context, cfg RetryConfig, operation func() error) error {
	var lastErr error
	delay := cfg.MinDelay

	for attempt := 0; attempt <= cfg.MaxRetries; attempt++ {
		lastErr = operation()
		if lastErr == nil {
			return nil
		}

		if !IsRetryableError(lastErr) {
			// Non-retryable, fail immediately. If we already replayed, say so:
			// a create that reached the backend before a transient gateway error
			// comes back as 422 "already exists" on the replay, and the caller's
			// recovery guidance otherwise describes the wrong situation.
			return annotateReplay(lastErr, attempt)
		}

		if attempt == cfg.MaxRetries {
			break // Last attempt failed
		}

		tflog.Debug(ctx, "Retryable error, will retry", map[string]any{
			"attempt": attempt + 1,
			"delay":   delay.String(),
			"error":   lastErr.Error(),
		})

		// Wait with context cancellation support
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
		}

		// Exponential backoff with cap
		delay = min(delay*2, cfg.MaxDelay)
	}

	return annotateReplay(lastErr, cfg.MaxRetries)
}

// annotateReplay records that the request was sent more than once, so an error
// produced by a replay is not read as the outcome of a single attempt.
func annotateReplay(err error, replays int) error {
	if err == nil || replays < 1 {
		return err
	}
	return fmt.Errorf(
		"%w (the provider replayed this request %d more time(s) after a transient failure; "+
			"if this was a create, an earlier attempt may already have created the record)",
		err, replays,
	)
}
