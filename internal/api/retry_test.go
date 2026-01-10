// internal/api/retry_test.go
package api

import (
	"context"
	"errors"
	"testing"
	"time"
)

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

	if cfg.MaxRetries != 3 {
		t.Errorf("Expected MaxRetries=3, got %d", cfg.MaxRetries)
	}
	if cfg.MinDelay != 10*time.Second {
		t.Errorf("Expected MinDelay=10s, got %v", cfg.MinDelay)
	}
	if cfg.MaxDelay != 60*time.Second {
		t.Errorf("Expected MaxDelay=60s, got %v", cfg.MaxDelay)
	}
}
