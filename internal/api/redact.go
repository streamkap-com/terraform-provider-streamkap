package api

import (
	"context"
	"encoding/json"
	"html"
	"net/url"
	"regexp"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"
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
	`api[_.-]?key|private[_.-]?key|access[_.-]?key|consumer[_.-]?key|connection[_.-]?string|implementation)`)

// A secret shorter than minSubstringSecretLen is masked only where it stands
// as a whole word, so a short password is still masked without blanking the
// same letters inside other words.
const minSubstringSecretLen = 6

type sentSecretsKey struct{}

// WithSentSecrets names values a request on ctx carries as secrets, so its
// error masks them even under a request key that does not say so (a service
// account key, an Authorization header, an OAuth grant).
func WithSentSecrets(ctx context.Context, values ...string) context.Context {
	prior, _ := ctx.Value(sentSecretsKey{}).([]string)
	return context.WithValue(ctx, sentSecretsKey{}, append(slices.Clone(prior), values...))
}

// sentSecrets returns the secret values a JSON request body carries.
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
			if secret {
				secrets = append(secrets, val)
			}
		}
	}
	walk(obj, false)
	return secrets
}

// secretForms returns each secret as an error can repeat it: verbatim,
// JSON-escaped in a re-encoded body, URL-encoded in a quoted request, and
// HTML-escaped in a gateway page. Longest first, so a connection string is
// masked before a password inside it.
func secretForms(secrets []string) []string {
	var forms []string
	for _, secret := range secrets {
		if secret == "" {
			continue
		}
		forms = append(forms, secret, url.QueryEscape(secret), url.PathEscape(secret), html.EscapeString(secret))
		if encoded, err := json.Marshal(secret); err == nil {
			forms = append(forms, string(encoded[1:len(encoded)-1]))
		}
	}
	slices.SortFunc(forms, func(a, b string) int { return len(b) - len(a) })
	return slices.Compact(forms)
}

// maskSentSecrets masks each sent secret where an error message repeats it.
func maskSentSecrets(text string, secrets []string) string {
	for _, secret := range secrets {
		if len(secret) >= minSubstringSecretLen {
			text = strings.ReplaceAll(text, secret, redactedPlaceholder)
		} else {
			text = maskWholeWord(text, secret)
		}
	}
	return text
}

func maskWholeWord(text, word string) string {
	var out strings.Builder
	for {
		i := strings.Index(text, word)
		if i < 0 {
			out.WriteString(text)
			return out.String()
		}
		before, _ := utf8.DecodeLastRuneInString(text[:i])
		after, _ := utf8.DecodeRuneInString(text[i+len(word):])
		out.WriteString(text[:i])
		if (isWordRune(before) && !afterPercentEscape(text[:i])) || isWordRune(after) {
			out.WriteString(word)
		} else {
			out.WriteString(redactedPlaceholder)
		}
		text = text[i+len(word):]
	}
}

// afterPercentEscape reports whether text ends in a %XX escape, whose last hex
// digit is a separator ("%3D" is "="), not part of the word that follows.
func afterPercentEscape(text string) bool {
	n := len(text)
	return n >= 3 && text[n-3] == '%' && isHex(text[n-2]) && isHex(text[n-1])
}

func isHex(c byte) bool {
	return '0' <= c && c <= '9' || 'a' <= c && c <= 'f' || 'A' <= c && c <= 'F'
}

func isWordRune(r rune) bool {
	return r != utf8.RuneError && (unicode.IsLetter(r) || unicode.IsDigit(r))
}
