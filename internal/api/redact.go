package api

import (
	"encoding/json"
	"regexp"
)

// sensitiveKeyRegex matches JSON keys that are likely to carry secrets. We
// err on the side of over-redaction: it is always better to mask a harmless
// identifier in a debug log than to leak a credential. Matches on full or
// partial key substrings, case-insensitively.
var sensitiveKeyRegex = regexp.MustCompile(`(?i)` +
	`(password|passwd|secret|token|credential|passphrase|` +
	`api[_-]?key|private[_-]?key|public[_-]?key|` +
	`access[_-]?key|auth|bearer|session|cookie|` +
	`client[_-]?secret|client[_-]?id|sasl|pem)`)

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
	if len(body) == 0 {
		return ""
	}
	var obj any
	if err := json.Unmarshal(body, &obj); err != nil {
		return "<non-JSON body omitted from logs>"
	}
	redacted := redactValue(obj)
	out, err := json.Marshal(redacted)
	if err != nil {
		return "<body redaction failed; omitted from logs>"
	}
	return string(out)
}

func redactValue(v any) any {
	switch val := v.(type) {
	case map[string]any:
		for k, inner := range val {
			if sensitiveKeyRegex.MatchString(k) {
				val[k] = redactedPlaceholder
				continue
			}
			val[k] = redactValue(inner)
		}
		return val
	case []any:
		for i, inner := range val {
			val[i] = redactValue(inner)
		}
		return val
	default:
		return v
	}
}
