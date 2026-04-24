package api

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestRedactSensitiveJSON_MasksKnownSecretKeys(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want []string // substrings that MUST appear (redacted marker) AND MUST NOT appear (the secret)
		bad  []string
	}{
		{
			name: "database password",
			in:   `{"name":"src","config":{"database_password":"hunter2","database_hostname":"db.example.com"}}`,
			want: []string{`"database_password":"***REDACTED***"`, `"database_hostname":"db.example.com"`},
			bad:  []string{"hunter2"},
		},
		{
			name: "aws access key pair",
			in:   `{"config":{"aws_access_key_id":"AKIAIOSFODNN7EXAMPLE","aws_secret_access_key":"wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"}}`,
			want: []string{
				`"aws_access_key_id":"***REDACTED***"`,
				`"aws_secret_access_key":"***REDACTED***"`,
			},
			bad: []string{"AKIAIOSFODNN7EXAMPLE", "wJalrXUtnFEMI"},
		},
		{
			name: "client credentials and tokens",
			in:   `{"client_id":"abc","client_secret":"def","access_token":"xyz","refresh_token":"uvw"}`,
			want: []string{
				`"client_id":"***REDACTED***"`,
				`"client_secret":"***REDACTED***"`,
				`"access_token":"***REDACTED***"`,
				`"refresh_token":"***REDACTED***"`,
			},
			bad: []string{"abc", "def", "xyz", "uvw"},
		},
		{
			name: "nested kafka sasl",
			in:   `{"kafka":{"sasl_username":"u","sasl_password":"p"}}`,
			want: []string{`"sasl_password":"***REDACTED***"`, `"sasl_username":"***REDACTED***"`},
			bad:  []string{`"p"`},
		},
		{
			name: "pem private key",
			in:   `{"snowflake_private_key":"-----BEGIN PRIVATE KEY-----\nMIIE...\n-----END PRIVATE KEY-----","snowflake_private_key_passphrase":"secret!"}`,
			want: []string{
				`"snowflake_private_key":"***REDACTED***"`,
				`"snowflake_private_key_passphrase":"***REDACTED***"`,
			},
			bad: []string{"BEGIN PRIVATE KEY", "secret!"},
		},
		{
			name: "non-sensitive keys untouched",
			in:   `{"name":"mypipeline","database_hostname":"db.example.com","database_port":5432}`,
			want: []string{`"name":"mypipeline"`, `"database_hostname":"db.example.com"`, `"database_port":5432`},
			bad:  []string{"***REDACTED***"},
		},
		{
			name: "array of objects with secrets",
			in:   `{"items":[{"password":"p1"},{"password":"p2"}]}`,
			want: []string{`"password":"***REDACTED***"`},
			bad:  []string{"p1", "p2"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := redactSensitiveJSON([]byte(tc.in))
			// Verify it's still valid JSON.
			var v any
			if err := json.Unmarshal([]byte(got), &v); err != nil {
				t.Fatalf("redacted output not valid JSON: %v (out=%s)", err, got)
			}
			for _, s := range tc.want {
				if !strings.Contains(got, s) {
					t.Errorf("output missing required substring %q; got=%s", s, got)
				}
			}
			for _, s := range tc.bad {
				if strings.Contains(got, s) {
					t.Errorf("output leaked secret substring %q; got=%s", s, got)
				}
			}
		})
	}
}

func TestRedactSensitiveJSON_EmptyAndMalformed(t *testing.T) {
	if got := redactSensitiveJSON(nil); got != "" {
		t.Errorf("expected empty string for nil input, got %q", got)
	}
	if got := redactSensitiveJSON([]byte("")); got != "" {
		t.Errorf("expected empty string for empty input, got %q", got)
	}
	if got := redactSensitiveJSON([]byte("not json at all")); !strings.Contains(got, "non-JSON body") {
		t.Errorf("expected placeholder for malformed JSON, got %q", got)
	}
}
