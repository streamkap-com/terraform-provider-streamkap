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
		{
			name: "transform implementation",
			in:   `{"implementation":{"language":"JavaScript","value_transform":"return fetch('https://user:secret@example.com');"}}`,
			want: []string{`"implementation":"***REDACTED***"`},
			bad:  []string{"value_transform", "user:secret"},
		},
		{
			// Connector configs go over the wire with dotted Kafka-Connect field
			// names, not the underscored Terraform attribute names. Those are the
			// keys redaction actually sees.
			name: "dotted wire field names",
			in: `{"config":{"api.key":"wh-secret","snowflake.private.key":"MIIEv","` +
				`database.password":"pw","http.headers.authorization":"Bearer t"}}`,
			want: []string{
				`"api.key":"***REDACTED***"`,
				`"snowflake.private.key":"***REDACTED***"`,
				`"database.password":"***REDACTED***"`,
				`"http.headers.authorization":"***REDACTED***"`,
			},
			bad: []string{"wh-secret", "MIIEv", `"pw"`, "Bearer t"},
		},
		{
			name: "dotted names do not over-redact plain identifiers",
			in:   `{"config":{"database.hostname":"db.example.com","topic.prefix":"tp"}}`,
			want: []string{`"database.hostname":"db.example.com"`, `"topic.prefix":"tp"`},
			bad:  []string{"***REDACTED***"},
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

func TestRedactSensitiveErrorJSON_MasksValidationInput(t *testing.T) {
	in := `[{"type":"string_too_short","loc":["body","password"],"msg":"too short","input":"plain-text-secret","ctx":{"error":"embedded-secret"}}]`
	got := redactSensitiveErrorJSON([]byte(in))

	if !strings.Contains(got, `"input":"***REDACTED***"`) {
		t.Fatalf("validation input was not redacted: %s", got)
	}
	if strings.Contains(got, "plain-text-secret") {
		t.Fatalf("validation input leaked: %s", got)
	}
	if !strings.Contains(got, `"ctx":"***REDACTED***"`) || strings.Contains(got, "embedded-secret") {
		t.Fatalf("validation context was not redacted: %s", got)
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

// A transform's code can carry an outbound credential, so the implementation
// subtree's values are masked like a secret's.
func TestSentSecrets_IncludeTransformImplementation(t *testing.T) {
	code := "fetch(url, {headers: {key: 'outbound-credential'}})"
	body, _ := json.Marshal(map[string]any{"implementation": map[string]any{"value_transform": code}})
	if got := maskSentSecrets("bad code: "+code, sentSecrets(body)); got != "bad code: "+redactedPlaceholder {
		t.Fatalf("implementation code left unmasked: %s", got)
	}
}

// A short secret right after a JSON escape is still a word of its own, and
// a secret quoted the way Python or a lower-case encoder writes it is masked.
func TestMaskSentSecrets_EscapesAndEncoderVariants(t *testing.T) {
	cases := map[string]struct{ secret, text, want string }{
		"after a JSON newline":  {"k9z", `"line1\nk9z"`, `"line1\n***REDACTED***"`},
		"after a JSON \\u":      {"k9z", `"<k9z"`, `"<***REDACTED***"`},
		"inside a longer word":  {"k9z", "k9zz", "k9zz"},
		"Python quote":          {"ab/cd+efgh", "x=ab/cd%2Befgh", "x=***REDACTED***"},
		"lower-case hex escape": {"a=b?c&d1", "q=a%3db%3fc%26d1", "q=***REDACTED***"},
	}
	for name, c := range cases {
		if got := maskSentSecrets(c.text, secretForms([]string{c.secret})); got != c.want {
			t.Errorf("%s: got %s, want %s", name, got, c.want)
		}
	}
}

// A transform's language names the runtime, not a credential; masking it
// would blank the word from every message about the transform.
func TestSentSecrets_KeepTheTransformLanguage(t *testing.T) {
	body, _ := json.Marshal(map[string]any{"implementation": map[string]any{"language": "JAVASCRIPT", "value_transform": "return 'outbound-credential'"}})
	got := maskSentSecrets("JAVASCRIPT failed on return 'outbound-credential'", secretForms(sentSecrets(body)))
	if got != "JAVASCRIPT failed on "+redactedPlaceholder {
		t.Fatalf("got %s", got)
	}
}
