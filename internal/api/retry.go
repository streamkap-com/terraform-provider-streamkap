// internal/api/retry.go
package api

import (
	"context"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// IsRetryableError checks if an error is transient and should be retried.
// Note: The Streamkap backend already retries Kafka Connect operations internally
// (tries multiple KC servers on ReadTimeout). This function identifies errors
// that indicate the backend exhausted its retries OR infrastructure issues.
func IsRetryableError(err error) bool {
	if err == nil {
		return false
	}
	errStr := strings.ToLower(err.Error())

	// Patterns indicating backend exhausted its KC retries
	retryablePatterns := []string{
		"kafkaconnecttimeout",    // Backend's custom timeout exception
		"request timed out",      // KC timeout message
		"sockettimeoutexception", // Java socket timeout
	}

	// Gateway/infrastructure errors - Streamkap API issues
	gatewayPatterns := []string{
		"502", "503", "504", // Gateway errors
		"bad gateway",
		"service unavailable",
		"gateway timeout",
	}

	// Network errors - connection to Streamkap API
	networkPatterns := []string{
		"connection refused",
		"connection reset",
		"no such host",
		"network unreachable",
		"i/o timeout",
	}

	// Kafka-specific transient errors
	kafkaPatterns := []string{
		"rebalance_in_progress",
		"leader_not_available",
		"not_leader_for_partition",
	}

	allPatterns := append(retryablePatterns, gatewayPatterns...)
	allPatterns = append(allPatterns, networkPatterns...)
	allPatterns = append(allPatterns, kafkaPatterns...)

	for _, pattern := range allPatterns {
		if strings.Contains(errStr, pattern) {
			return true
		}
	}
	return false
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
		MaxRetries: 3,
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
			return lastErr // Non-retryable, fail immediately
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

	return lastErr
}
