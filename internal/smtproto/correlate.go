package smtproto

import (
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
)

// PriorInstance is one chain entry as recorded in state.
type PriorInstance struct {
	Key           string
	Type          string
	ID            string
	Alias         string
	SchemaVersion int64
	SecretVersion int64
}

// PlannedInstance is one chain entry as configured.
type PlannedInstance struct {
	Key  string
	Type string
}

// Correlated is a planned entry joined to its prior identity by key.
type Correlated struct {
	Key           string
	Type          string
	ID            string
	Alias         string
	SchemaVersion int64
	SecretVersion int64
	Matched       bool
}

// Correlate joins the planned chain to prior state by provider key only.
// Position, display name and server ids play no part: a matched key keeps
// its id and alias, a new key starts with none, and a key whose type changed
// is rejected because the server instance under that id cannot change type.
func Correlate(prior []PriorInstance, planned []PlannedInstance) ([]Correlated, diag.Diagnostics) {
	var diags diag.Diagnostics
	byKey := make(map[string]PriorInstance, len(prior))
	for _, p := range prior {
		byKey[p.Key] = p
	}
	seen := make(map[string]bool, len(planned))
	out := make([]Correlated, 0, len(planned))
	for i, p := range planned {
		at := path.Root(AttrChain).AtListIndex(i)
		if p.Key == "" {
			diags.AddAttributeError(at.AtName("key"), "Missing instance key", "every chain entry needs an immutable key")
			continue
		}
		if seen[p.Key] {
			diags.AddAttributeError(at.AtName("key"), "Duplicate instance key", fmt.Sprintf("%q is used by more than one chain entry", p.Key))
			continue
		}
		seen[p.Key] = true
		c := Correlated{Key: p.Key, Type: p.Type}
		if old, ok := byKey[p.Key]; ok {
			if old.Type != p.Type {
				diags.AddAttributeError(at.AtName("type"),
					"Transform type change is not allowed in place",
					fmt.Sprintf("instance %q is %s on the server (id %s) and cannot become %s. Remove the entry in one apply, then add the %s instance under a new key.", p.Key, old.Type, old.ID, p.Type, p.Type))
				continue
			}
			c.ID, c.Alias, c.SchemaVersion, c.SecretVersion, c.Matched = old.ID, old.Alias, old.SchemaVersion, old.SecretVersion, true
		}
		out = append(out, c)
	}
	return out, diags
}
