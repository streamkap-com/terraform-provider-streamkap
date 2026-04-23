package transform

import (
	"errors"
	"testing"
)

// TestIsJobNotDeployedError validates the string match that distinguishes
// an authoritative "no deployment exists" signal from a transient backend
// failure. The backend's /job_status handler rewrites HTTPException(404) into
// 400 with either "Job not found" or "Transform not found" — those two detail
// strings are the only signals we can rely on to demote connector_status to
// UNKNOWN without letting transient 5xx errors silently lose state.
func TestIsJobNotDeployedError(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want bool
	}{
		{name: "nil error is not a not-deployed signal", err: nil, want: false},
		{name: "job not found is authoritative", err: errors.New(`API error: {"detail":"Job not found"}`), want: true},
		{name: "transform not found is authoritative", err: errors.New(`API error: {"detail":"Transform not found"}`), want: true},
		{name: "job not found with request_id suffix", err: errors.New(`Job not found (request_id=abc123)`), want: true},
		{name: "transient 500 is not a not-deployed signal", err: errors.New("500 Internal Server Error"), want: false},
		{name: "network timeout is not a not-deployed signal", err: errors.New("context deadline exceeded"), want: false},
		{name: "unrelated 400 is not a not-deployed signal", err: errors.New("validation error: bad request"), want: false},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			got := isJobNotDeployedError(tc.err)
			if got != tc.want {
				t.Fatalf("isJobNotDeployedError(%v) = %v, want %v", tc.err, got, tc.want)
			}
		})
	}
}

func TestStripNullValues(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]any
		wantKeys []string
	}{
		{
			name:     "removes nil values",
			input:    map[string]any{"a": "hello", "b": nil, "c": 42},
			wantKeys: []string{"a", "c"},
		},
		{
			name:     "preserves non-nil values",
			input:    map[string]any{"a": "hello", "b": 0, "c": false, "d": ""},
			wantKeys: []string{"a", "b", "c", "d"},
		},
		{
			name:     "handles nested maps",
			input:    map[string]any{"nested": map[string]any{"a": 1, "b": nil}},
			wantKeys: []string{"nested"},
		},
		{
			name:     "removes empty nested maps after stripping",
			input:    map[string]any{"nested": map[string]any{"a": nil}},
			wantKeys: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := stripNullValues(tt.input)
			if len(result) != len(tt.wantKeys) {
				t.Errorf("stripNullValues() returned %d keys, want %d", len(result), len(tt.wantKeys))
			}
			for _, key := range tt.wantKeys {
				if _, ok := result[key]; !ok {
					t.Errorf("stripNullValues() missing expected key %q", key)
				}
			}
		})
	}
}

func TestIsImplementationSubset(t *testing.T) {
	tests := []struct {
		name      string
		stateJSON string
		apiImpl   map[string]any
		want      bool
	}{
		{
			name:      "state is subset of API (API added defaults)",
			stateJSON: `{"inputTables":[],"rollupSQL":"SELECT 1","keyFields":["id"]}`,
			apiImpl: map[string]any{
				"inputTables":         []any{},
				"rollupSQL":           "SELECT 1",
				"keyFields":           []any{"id"},
				"sourceIdleTimeoutMs": float64(30000),
				"stateTTLMs":          float64(86400000),
			},
			want: true,
		},
		{
			name:      "state equals API (no defaults added)",
			stateJSON: `{"rollupSQL":"SELECT 1","keyFields":["id"]}`,
			apiImpl: map[string]any{
				"rollupSQL": "SELECT 1",
				"keyFields": []any{"id"},
			},
			want: true,
		},
		{
			name:      "state field value differs from API (real drift)",
			stateJSON: `{"rollupSQL":"SELECT 1","keyFields":["id"]}`,
			apiImpl: map[string]any{
				"rollupSQL":           "SELECT 2",
				"keyFields":           []any{"id"},
				"sourceIdleTimeoutMs": float64(30000),
			},
			want: false,
		},
		{
			name:      "state has field missing from API",
			stateJSON: `{"rollupSQL":"SELECT 1","extraField":"value"}`,
			apiImpl: map[string]any{
				"rollupSQL": "SELECT 1",
			},
			want: false,
		},
		{
			name:      "nested objects match",
			stateJSON: `{"inputTables":[{"name":"orders","topicMatcherRegex":".*orders$"}]}`,
			apiImpl: map[string]any{
				"inputTables": []any{
					map[string]any{"name": "orders", "topicMatcherRegex": ".*orders$"},
				},
				"sourceIdleTimeoutMs": float64(30000),
			},
			want: true,
		},
		{
			name:      "nested object value differs",
			stateJSON: `{"inputTables":[{"name":"orders"}]}`,
			apiImpl: map[string]any{
				"inputTables": []any{
					map[string]any{"name": "products"},
				},
			},
			want: false,
		},
		{
			name:      "invalid state JSON returns false",
			stateJSON: `not-valid-json`,
			apiImpl:   map[string]any{"a": 1},
			want:      false,
		},
		{
			name:      "empty state is subset of any API response",
			stateJSON: `{}`,
			apiImpl: map[string]any{
				"sourceIdleTimeoutMs": float64(30000),
				"stateTTLMs":          float64(86400000),
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isImplementationSubset(tt.stateJSON, tt.apiImpl)
			if got != tt.want {
				t.Errorf("isImplementationSubset() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestJsonValuesEqual(t *testing.T) {
	tests := []struct {
		name string
		a, b any
		want bool
	}{
		{"both nil", nil, nil, true},
		{"one nil", nil, "hello", false},
		{"equal strings", "hello", "hello", true},
		{"different strings", "hello", "world", false},
		{"equal numbers", float64(42), float64(42), true},
		{"different numbers", float64(42), float64(43), false},
		{"equal bools", true, true, true},
		{"different bools", true, false, false},
		{"equal slices", []any{"a", "b"}, []any{"a", "b"}, true},
		{"different slice length", []any{"a"}, []any{"a", "b"}, false},
		{"different slice values", []any{"a", "b"}, []any{"a", "c"}, false},
		{
			"equal maps",
			map[string]any{"a": float64(1)},
			map[string]any{"a": float64(1)},
			true,
		},
		{
			"maps different values",
			map[string]any{"a": float64(1)},
			map[string]any{"a": float64(2)},
			false,
		},
		{
			"maps different keys (exact match required for nested)",
			map[string]any{"a": float64(1)},
			map[string]any{"a": float64(1), "b": float64(2)},
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := jsonValuesEqual(tt.a, tt.b)
			if got != tt.want {
				t.Errorf("jsonValuesEqual() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMarshalImplementation(t *testing.T) {
	t.Run("strips null values", func(t *testing.T) {
		impl := map[string]any{
			"rollupSQL": "SELECT 1",
			"nullField": nil,
			"keyFields": []any{"id"},
		}
		result, err := marshalImplementation(impl)
		if err != nil {
			t.Fatalf("marshalImplementation() error: %v", err)
		}
		if result == "" {
			t.Fatal("marshalImplementation() returned empty string")
		}
		// Should contain non-null fields
		if !containsSubstring(result, "rollupSQL") {
			t.Error("result should contain rollupSQL")
		}
		if !containsSubstring(result, "keyFields") {
			t.Error("result should contain keyFields")
		}
		// Should NOT contain null field
		if containsSubstring(result, "nullField") {
			t.Error("result should not contain nullField")
		}
	})
}

func containsSubstring(s, substr string) bool {
	return len(s) >= len(substr) && searchSubstring(s, substr)
}

func searchSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
