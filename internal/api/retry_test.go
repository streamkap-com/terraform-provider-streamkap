// internal/api/retry_test.go
package api

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"testing"
	"time"
)

// TestIsRetryableError_APIError covers the classification that matters in
// production: the status class decides, not the wording. The string-matching
// version retried a 400 whose detail merely contained "4290" and never retried
// a 429 whose detail omitted the digits.
func TestIsRetryableError_APIError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{"429 without the digits in the detail", &APIError{StatusCode: http.StatusTooManyRequests, Detail: "rate limit exceeded for this tenant"}, true},
		{"502", &APIError{StatusCode: http.StatusBadGateway, Detail: "upstream connect error"}, true},
		{"503", &APIError{StatusCode: http.StatusServiceUnavailable, Detail: "no healthy upstream"}, true},
		{"504", &APIError{StatusCode: http.StatusGatewayTimeout, Detail: "upstream request timeout"}, true},
		{"400 whose detail contains a status-like number", &APIError{StatusCode: http.StatusBadRequest, Detail: "port 4290 invalid"}, false},
		{"400 mentioning 502 in prose", &APIError{StatusCode: http.StatusBadRequest, Detail: "hostname db-502.example.com is unreachable"}, false},
		{"401", &APIError{StatusCode: http.StatusUnauthorized, Detail: "Unauthorized"}, false},
		{"404", &APIError{StatusCode: http.StatusNotFound, Detail: "Source not found"}, false},
		{"422", &APIError{StatusCode: http.StatusUnprocessableEntity, Detail: "Validation error: missing required field"}, false},
		{"plain 500", &APIError{StatusCode: http.StatusInternalServerError, Detail: "Internal server error"}, false},
		{"500 carrying a Kafka Connect rebalance", &APIError{StatusCode: http.StatusInternalServerError, Detail: "Request cannot be completed because a rebalance is expected"}, true},
		{"500 carrying a Kafka Connect 409", &APIError{StatusCode: http.StatusInternalServerError, Detail: "Kafka Connect API call failed with status code 409 and response ..."}, true},
		{"500 carrying a KC timeout", &APIError{StatusCode: http.StatusInternalServerError, Detail: "KafkaConnectTimeout: timed out on all nodes"}, true},
		{"wrapped 429", fmt.Errorf("CreateSource: %w", &APIError{StatusCode: http.StatusTooManyRequests, Detail: "slow down"}), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsRetryableError(tt.err); got != tt.expected {
				t.Errorf("IsRetryableError(%v) = %v, want %v", tt.err, got, tt.expected)
			}
		})
	}
}

func TestIsRetryableError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{"nil error", nil, false},
		// KC timeout - backend exhausted retries
		{"KC timeout", errors.New("KafkaConnectTimeout"), true},
		{"request timeout", errors.New("Request timed out"), true},
		{"socket timeout", errors.New("SocketTimeoutException: connect timed out"), true},
		// Rate limit errors
		{"429 error", errors.New("429 Too Many Requests"), true},
		{"too many requests", errors.New("too many requests"), true},
		// Gateway errors - infrastructure issues
		{"503 error", errors.New("503 Service Unavailable"), true},
		{"502 error", errors.New("502 Bad Gateway"), true},
		{"504 error", errors.New("504 Gateway Timeout"), true},
		// Connection errors - network issues
		{"connection refused", errors.New("connection refused"), true},
		{"connection reset", errors.New("connection reset by peer"), true},
		{"i/o timeout", errors.New("i/o timeout"), true},
		// Kafka-specific retryable errors
		{"rebalance", errors.New("REBALANCE_IN_PROGRESS"), true},
		{"leader not available", errors.New("LEADER_NOT_AVAILABLE"), true},
		// Non-retryable errors
		{"auth error", errors.New("401 Unauthorized"), false},
		{"validation error", errors.New("Invalid configuration"), false},
		{"not found", errors.New("404 Not Found"), false},
		{"bad request", errors.New("400 Bad Request"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsRetryableError(tt.err)
			if result != tt.expected {
				t.Errorf("IsRetryableError(%v) = %v, want %v", tt.err, result, tt.expected)
			}
		})
	}
}

func TestRetryWithBackoff_SucceedsOnFirstTry(t *testing.T) {
	ctx := context.Background()
	cfg := RetryConfig{MaxRetries: 3, MinDelay: 10 * time.Millisecond, MaxDelay: 100 * time.Millisecond}

	attempts := 0
	err := RetryWithBackoff(ctx, cfg, func() error {
		attempts++
		return nil
	})

	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
	if attempts != 1 {
		t.Errorf("Expected 1 attempt, got %d", attempts)
	}
}

func TestRetryWithBackoff_RetriesOnTransientError(t *testing.T) {
	ctx := context.Background()
	cfg := RetryConfig{MaxRetries: 3, MinDelay: 10 * time.Millisecond, MaxDelay: 100 * time.Millisecond}

	attempts := 0
	err := RetryWithBackoff(ctx, cfg, func() error {
		attempts++
		if attempts < 3 {
			return errors.New("503 Service Unavailable")
		}
		return nil
	})

	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
	if attempts != 3 {
		t.Errorf("Expected 3 attempts, got %d", attempts)
	}
}

func TestRetryWithBackoff_FailsImmediatelyOnNonRetryable(t *testing.T) {
	ctx := context.Background()
	cfg := RetryConfig{MaxRetries: 3, MinDelay: 10 * time.Millisecond, MaxDelay: 100 * time.Millisecond}

	attempts := 0
	err := RetryWithBackoff(ctx, cfg, func() error {
		attempts++
		return errors.New("400 Bad Request")
	})

	if err == nil {
		t.Error("Expected error, got nil")
	}
	if attempts != 1 {
		t.Errorf("Expected 1 attempt (no retry), got %d", attempts)
	}
}

func TestRetryWithBackoff_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cfg := RetryConfig{MaxRetries: 5, MinDelay: 100 * time.Millisecond, MaxDelay: 1 * time.Second}

	attempts := 0
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	err := RetryWithBackoff(ctx, cfg, func() error {
		attempts++
		return errors.New("503 Service Unavailable")
	})

	if !errors.Is(err, context.Canceled) {
		t.Errorf("Expected context.Canceled error, got: %v", err)
	}
}

func TestRetryWithBackoff_MaxRetriesExhausted(t *testing.T) {
	ctx := context.Background()
	cfg := RetryConfig{MaxRetries: 2, MinDelay: 10 * time.Millisecond, MaxDelay: 100 * time.Millisecond}

	attempts := 0
	err := RetryWithBackoff(ctx, cfg, func() error {
		attempts++
		return errors.New("503 Service Unavailable")
	})

	if err == nil {
		t.Error("Expected error after max retries, got nil")
	}
	// MaxRetries=2 means 3 total attempts (initial + 2 retries)
	if attempts != 3 {
		t.Errorf("Expected 3 attempts (1 + 2 retries), got %d", attempts)
	}
}

func TestDefaultRetryConfig(t *testing.T) {
	cfg := DefaultRetryConfig()

	if cfg.MaxRetries != 5 {
		t.Errorf("Expected MaxRetries=5, got %d", cfg.MaxRetries)
	}
	if cfg.MinDelay != 10*time.Second {
		t.Errorf("Expected MinDelay=10s, got %v", cfg.MinDelay)
	}
	if cfg.MaxDelay != 60*time.Second {
		t.Errorf("Expected MaxDelay=60s, got %v", cfg.MaxDelay)
	}
}
