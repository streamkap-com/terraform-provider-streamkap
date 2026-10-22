package api

import (
	"encoding/json"
	"regexp"
	"sort"
	"strings"
)

// sensitiveKeyRegex matches JSON keys that are likely to carry secrets. We
// err on the side of over-redaction: it is always better to mask a harmless
// identifier in a debug log than to leak a credential. Matches on full or
// partial key substrings, case-insensitively.
//
// The separator class must include `.`: connector configs travel with dotted
// Kafka-Connect field names (`api.key`, `snowflake.private.key`), not the
// underscored Terraform attribute names, and those dotted keys are what this
// sees. Matching only `[_-]` silently passed both of those through in the clear.
var sensitiveKeyRegex = regexp.MustCompile(`(?i)` +
	`(password|passwd|secret|token|credential|passphrase|` +
	`api[_.-]?key|private[_.-]?key|public[_.-]?key|` +
	`access[_.-]?key|consumer[_.-]?key|service[_.-]?account|keyfile|connection[_.-]?string|` +
	`auth|bearer|session|cookie|` +
	`client[_.-]?secret|client[_.-]?id|sasl|pem|implementation)`)

// Note: `implementation` masks the whole transform implementation subtree
// (language, topic patterns and the JS/SQL body together), because an enrich
// transform's code can embed an outbound credential. Debugging a transform body
// therefore has to go through the API, not provider debug logs.

const redactedPlaceholder = "***REDACTED***"

// redactSensitiveJSON takes a JSON request body and returns its textual form
// with sensitive values masked. Used by the tflog.Debug call sites in
// api/*.go so TRACE/DEBUG logs remain useful for request inspection without
// leaking credentials into user logs, CI output, or support bundles.
//
// Anti-goals: this is best-effort sanitation, not a security boundary. Do
// not rely on it to make a debug log safe to publish — treat any provider
// debug log as sensitive and restrict access accordingly.
func redactSensitiveJSON(body []byte) string {
	return redactJSON(body, false, "<non-JSON body omitted from logs>")
}

// redactSensitiveErrorJSON also masks Pydantic's `input` and `ctx` fields,
// which can contain rejected request values in structured validation errors.
func redactSensitiveErrorJSON(body []byte) string {
	return redactJSON(body, true, "<error response omitted from logs>")
}

func redactJSON(body []byte, errorResponse bool, malformedPlaceholder string) string {
	if len(body) == 0 {
		return ""
	}
	var obj any
	if err := json.Unmarshal(body, &obj); err != nil {
		return malformedPlaceholder
	}
	redacted := redactValue(obj, errorResponse)
	out, err := json.Marshal(redacted)
	if err != nil {
		return "<body redaction failed; omitted from logs>"
	}
	return string(out)
}

func redactValue(v any, errorResponse bool) any {
	switch val := v.(type) {
	case map[string]any:
		for k, inner := range val {
			if sensitiveKeyRegex.MatchString(k) ||
				(errorResponse && (strings.EqualFold(k, "input") || strings.EqualFold(k, "ctx"))) {
				val[k] = redactedPlaceholder
				continue
			}
			val[k] = redactValue(inner, errorResponse)
		}
		return val
	case []any:
		for i, inner := range val {
			val[i] = redactValue(inner, errorResponse)
		}
		return val
	default:
		return v
	}
}

// secretValueKeyRegex picks the request values masked in an error message. It
// is narrower than sensitiveKeyRegex: that one's `auth` and `client_id` would
// mask ordinary words such as an auth_mode value wherever the backend's prose
// repeats them.
var secretValueKeyRegex = regexp.MustCompile(`(?i)` +
	`(password|passwd|secret|token|passphrase|credential|keyfile|sasl|` +
	`api[_.-]?key|private[_.-]?key|access[_.-]?key|consumer[_.-]?key|connection[_.-]?string)`)

// minMaskedSecretLen keeps short values, which also occur as plain words,
// from blanking out parts of a message.
const minMaskedSecretLen = 6

// sentSecrets returns the secret values a JSON request body carries, longest
// first, so a connection string is masked before a password inside it. Each
// also comes JSON-escaped, the form a re-encoded error body repeats it in.
func sentSecrets(body []byte) []string {
	var obj any
	if json.Unmarshal(body, &obj) != nil {
		return nil
	}
	var secrets []string
	var walk func(v any, secret bool)
	walk = func(v any, secret bool) {
		switch val := v.(type) {
		case map[string]any:
			for k, inner := range val {
				walk(inner, secret || secretValueKeyRegex.MatchString(k))
			}
		case []any:
			for _, inner := range val {
				walk(inner, secret)
			}
		case string:
			if secret && len(val) >= minMaskedSecretLen {
				secrets = append(secrets, val)
				if encoded, err := json.Marshal(val); err == nil && string(encoded[1:len(encoded)-1]) != val {
					secrets = append(secrets, string(encoded[1:len(encoded)-1]))
				}
			}
		}
	}
	walk(obj, false)
	sort.Slice(secrets, func(i, j int) bool { return len(secrets[i]) > len(secrets[j]) })
	return secrets
}

// maskSentSecrets masks each sent secret where an error message repeats it.
func maskSentSecrets(text string, secrets []string) string {
	for _, secret := range secrets {
		text = strings.ReplaceAll(text, secret, redactedPlaceholder)
	}
	return text
}
