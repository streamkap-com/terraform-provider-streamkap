package smt

import (
	"fmt"
	"sort"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
)

// EscapeToken applies JSON Pointer escaping to one row-key value: `~`
// becomes `~0` and `/` becomes `~1`, in that order.
func EscapeToken(s string) string {
	return strings.ReplaceAll(strings.ReplaceAll(s, "~", "~0"), "/", "~1")
}

// UnescapeToken reverses EscapeToken: `~1` first, then `~0`.
func UnescapeToken(s string) string {
	return strings.ReplaceAll(strings.ReplaceAll(s, "~1", "/"), "~0", "~")
}

// InstancePointer replaces each `*` of a schema-level template with the
// escaped row key at that list position.
func InstancePointer(template string, rowKeys ...string) string {
	parts := strings.Split(template, "/")
	i := 0
	for n, part := range parts {
		if part == "*" && i < len(rowKeys) {
			parts[n] = EscapeToken(rowKeys[i])
			i++
		}
	}
	return strings.Join(parts, "/")
}

// SecretAddress is a validated instance-level pointer: the template it
// matched and the row keys it names, in order.
type SecretAddress struct {
	Pointer  string
	Template string
	RowKeys  []string
}

// ResolvePointer matches an instance-level pointer against the type's
// secret templates. Each `*` consumes one escaped row-key token; every other
// token must match literally. A pointer that names no template, or names a
// list position by index, is rejected.
func ResolvePointer(spec *TypeSpec, pointer string) (*SecretAddress, error) {
	if !strings.HasPrefix(pointer, "/") {
		return nil, fmt.Errorf("secret pointer %q must start with /", pointer)
	}
	tokens := strings.Split(pointer, "/")[1:]
	for _, tmpl := range spec.SecretPaths {
		want := strings.Split(tmpl, "/")[1:]
		if len(want) != len(tokens) {
			continue
		}
		var keys []string
		ok := true
		for i, w := range want {
			if w == "*" {
				keys = append(keys, UnescapeToken(tokens[i]))
				continue
			}
			if w != tokens[i] {
				ok = false
				break
			}
		}
		if ok {
			return &SecretAddress{Pointer: pointer, Template: tmpl, RowKeys: keys}, nil
		}
	}
	return nil, fmt.Errorf("secret pointer %q matches none of %s's secret paths %v", pointer, spec.TypeID, spec.SecretPaths)
}

// RowExists walks the wire-shaped config along the template's containers,
// checking that a row with each named key exists at every list position. The
// row key property comes from the corpus row_keys, never from an assumed
// name. The secret leaf itself is never in the config, so the walk stops at
// its parent.
func RowExists(spec *TypeSpec, addr *SecretAddress, config map[string]any) bool {
	var cur any = config
	tokens := strings.Split(addr.Template, "/")[1:]
	listPath := ""
	keyIdx := 0
	for _, tok := range tokens[:len(tokens)-1] {
		if tok == "*" {
			rows, ok := cur.([]any)
			if !ok {
				return false
			}
			keyProp, ok := spec.RowKeys[listPath]
			if !ok {
				return false
			}
			found := false
			for _, r := range rows {
				row, _ := r.(map[string]any)
				if row[keyProp] == addr.RowKeys[keyIdx] {
					cur = row
					found = true
					break
				}
			}
			if !found {
				return false
			}
			keyIdx++
			listPath += "/*"
			continue
		}
		listPath += "/" + tok
		obj, ok := cur.(map[string]any)
		if !ok {
			return false
		}
		next, ok := obj[tok]
		if !ok {
			return false
		}
		cur = next
	}
	_, isObject := cur.(map[string]any)
	return isObject
}

// SecretEntry is one configured smt_secrets element, read from req.Config.
// Values is nil when the write-only list is unset.
type SecretEntry struct {
	Key     string
	Version int64
	Values  []SecretValue
}

// SecretValue is one write-only row: Replace when Value is set, Clear when
// Clear is set.
type SecretValue struct {
	Pointer string
	Value   *string
	Clear   bool
}

// SecretOp is one operation to execute against a persisted instance.
type SecretOp struct {
	Key     string
	Pointer string
	Clear   bool
	Value   string
}

// PlanSecretOps decides which operations an apply executes. prior holds the
// last rotation version applied to each instance key, as the chain entry
// records it (0 or absent for a key never rotated, which is also every key on
// create). It is not the version of the smt_secrets entry currently in
// state: that entry can be removed and re-added while the server instance
// lives on. An increased version executes every configured row exactly
// once; an unchanged version executes nothing even when values are still
// configured; everything else is rejected before any request is sent. next
// is prior with every executed rotation applied, for the chain to record.
func PlanSecretOps(prior map[string]int64, entries []SecretEntry, chain map[string]string, cat Catalog) (ops []SecretOp, next map[string]int64, diags diag.Diagnostics) {
	next = make(map[string]int64, len(prior))
	for k, v := range prior {
		next[k] = v
	}
	seen := map[string]bool{}
	for i, e := range entries {
		p := path.Root(AttrSecrets).AtListIndex(i)
		if seen[e.Key] {
			diags.AddAttributeError(p.AtName("key"), "Duplicate secret entry", fmt.Sprintf("instance key %q appears more than once", e.Key))
			continue
		}
		seen[e.Key] = true
		typeID, inChain := chain[e.Key]
		if !inChain {
			diags.AddAttributeError(p.AtName("key"), "Secret entry for unknown instance", fmt.Sprintf("instance key %q is not in %s", e.Key, AttrChain))
			continue
		}
		spec, ok := cat[typeID]
		if !ok {
			diags.AddAttributeError(p.AtName("key"), "Unknown transform type", typeID)
			continue
		}
		if e.Version < 1 {
			diags.AddAttributeError(p.AtName("version"), "Invalid rotation version", "version must be a positive integer")
			continue
		}
		before := prior[e.Key]
		switch {
		case e.Version < before:
			diags.AddAttributeError(p.AtName("version"), "Rotation version decreased", fmt.Sprintf("version %d is below the %d last applied to %q; versions only increase", e.Version, before, e.Key))
			continue
		case e.Version == before:
			// Unchanged: no operation, whatever values are configured. A value
			// edited without a version bump is invisible here by design.
			continue
		}
		if len(e.Values) == 0 {
			diags.AddAttributeError(p.AtName("version"), "Rotation without operations", fmt.Sprintf("version %d is new for %q but no values are configured", e.Version, e.Key))
			continue
		}
		next[e.Key] = e.Version
		rowSeen := map[string]bool{}
		for j, v := range e.Values {
			vp := p.AtName("values").AtListIndex(j)
			if _, err := ResolvePointer(spec, v.Pointer); err != nil {
				diags.AddAttributeError(vp.AtName("pointer"), "Invalid secret pointer", err.Error())
				continue
			}
			if rowSeen[v.Pointer] {
				diags.AddAttributeError(vp.AtName("pointer"), "Duplicate secret pointer", v.Pointer)
				continue
			}
			rowSeen[v.Pointer] = true
			if (v.Value == nil) == !v.Clear {
				diags.AddAttributeError(vp, "Ambiguous secret operation", "set exactly one of value or clear")
				continue
			}
			op := SecretOp{Key: e.Key, Pointer: v.Pointer, Clear: v.Clear}
			if v.Value != nil {
				op.Value = *v.Value
			}
			ops = append(ops, op)
		}
	}
	sort.SliceStable(ops, func(a, b int) bool {
		if ops[a].Key != ops[b].Key {
			return ops[a].Key < ops[b].Key
		}
		return ops[a].Pointer < ops[b].Pointer
	})
	return ops, next, diags
}
