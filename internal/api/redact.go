package api

import (
	"encoding/json"
	"regexp"
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
	`access[_.-]?key|auth|bearer|session|cookie|` +
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
