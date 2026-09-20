package smtproto

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Attribute names of the nested SMT surface on a connector resource.
const (
	AttrChain   = "smt_chain"
	AttrSecrets = "smt_secrets"

	attrKey           = "key"
	attrType          = "type"
	attrName          = "name"
	attrID            = "id"
	attrAlias         = "alias"
	attrSchemaVersion = "schema_version"
	attrVersion       = "version"
	attrValues        = "values"
	attrPointer       = "pointer"
	attrValue         = "value"
	attrClear         = "clear"
)

// ChainSchema is the chain attribute derived from a catalog, with the typed
// config child of every type it admits.
type ChainSchema struct {
	Catalog   Catalog
	Children  map[string]*ConfigChild
	Attribute schema.ListNestedAttribute
	EntryType types.ObjectType
}

// BuildChain derives smt_chain from the catalog. Each entry carries an
// immutable provider key, the type, a mutable display name, the computed
// server identity, and one config child per catalog type, of which exactly
// the one named by `type` may be set.
func BuildChain(cat Catalog) (*ChainSchema, error) {
	cs := &ChainSchema{Catalog: cat, Children: map[string]*ConfigChild{}}
	attrs := map[string]schema.Attribute{
		attrKey: schema.StringAttribute{
			Required:    true,
			Description: "Immutable provider-side key of this instance. Changing it creates a new instance; it never changes the server id it was matched to.",
			Validators:  []validator.String{stringvalidator.LengthAtLeast(1)},
		},
		attrType: schema.StringAttribute{
			Required:    true,
			Description: "Catalog type id. Cannot change under an existing key.",
			Validators:  []validator.String{stringvalidator.OneOf(cat.TypeIDs()...)},
		},
		attrName: schema.StringAttribute{
			Optional:    true,
			Description: "Mutable display name.",
		},
		attrID:            schema.StringAttribute{Computed: true, Description: "Server-owned instance id."},
		attrAlias:         schema.StringAttribute{Computed: true, Description: "Server-owned Kafka Connect alias."},
		attrSchemaVersion: schema.Int64Attribute{Computed: true, Description: "Catalog schema version the instance was persisted with."},
	}
	for _, id := range cat.TypeIDs() {
		child, err := BuildConfigChild(cat[id])
		if err != nil {
			return nil, fmt.Errorf("BuildChain %s: %w", id, err)
		}
		child.Attribute.Description = "Configuration when type is " + id + "."
		attrs[id] = child.Attribute
		cs.Children[id] = child
	}
	cs.Attribute = schema.ListNestedAttribute{
		Optional:     true,
		Description:  "Ordered transform chain. Entries are correlated to server instances by key.",
		NestedObject: schema.NestedAttributeObject{Attributes: attrs},
	}
	cs.EntryType = cs.Attribute.NestedObject.Type().(types.ObjectType)
	return cs, nil
}

// SecretsAttribute is the write-only secret input: per instance key, a
// stateful rotation version and a write-only list of pointer operations.
// Only the version ever reaches plan or state.
func SecretsAttribute() schema.ListNestedAttribute {
	return schema.ListNestedAttribute{
		Optional:    true,
		Description: "Secret values for chain instances, addressed by canonical pointer. Values are write-only; bump version to rotate.",
		NestedObject: schema.NestedAttributeObject{
			Attributes: map[string]schema.Attribute{
				attrKey: schema.StringAttribute{Required: true, Description: "Instance key in " + AttrChain + "."},
				attrVersion: schema.Int64Attribute{
					Required:    true,
					Description: "Positive, monotonic rotation version. Increase it to apply the configured values once.",
					Validators:  []validator.Int64{int64validator.AtLeast(1)},
				},
				attrValues: schema.ListNestedAttribute{
					Optional:    true,
					WriteOnly:   true,
					Description: "Write-only operations. Each row replaces or clears the secret at one canonical pointer.",
					NestedObject: schema.NestedAttributeObject{
						Attributes: map[string]schema.Attribute{
							attrPointer: schema.StringAttribute{Required: true, WriteOnly: true, Description: "Canonical secret pointer, for example /routes/row-east/token."},
							attrValue:   schema.StringAttribute{Optional: true, WriteOnly: true, Sensitive: true},
							attrClear:   schema.BoolAttribute{Optional: true, WriteOnly: true},
						},
					},
				},
			},
		},
	}
}

// secretsEntryType is the object type of one smt_secrets element.
var secretsEntryType = SecretsAttribute().NestedObject.Type().(types.ObjectType)

// PlannedFromList reads the key and type of every configured entry.
func PlannedFromList(list types.List) []PlannedInstance {
	if list.IsNull() || list.IsUnknown() {
		return nil
	}
	out := make([]PlannedInstance, 0, len(list.Elements()))
	for _, e := range list.Elements() {
		obj := e.(types.Object).Attributes()
		out = append(out, PlannedInstance{Key: str(obj[attrKey]), Type: str(obj[attrType])})
	}
	return out
}

// PriorFromList reads the identity of every entry recorded in state.
func PriorFromList(list types.List) []PriorInstance {
	if list.IsNull() || list.IsUnknown() {
		return nil
	}
	out := make([]PriorInstance, 0, len(list.Elements()))
	for _, e := range list.Elements() {
		obj := e.(types.Object).Attributes()
		out = append(out, PriorInstance{
			Key:           str(obj[attrKey]),
			Type:          str(obj[attrType]),
			ID:            str(obj[attrID]),
			Alias:         str(obj[attrAlias]),
			SchemaVersion: i64(obj[attrSchemaVersion]),
		})
	}
	return out
}

// SecretEntriesFromList reads smt_secrets from req.Config, the only place
// the write-only values exist.
func SecretEntriesFromList(list types.List) []SecretEntry {
	if list.IsNull() || list.IsUnknown() {
		return nil
	}
	out := make([]SecretEntry, 0, len(list.Elements()))
	for _, e := range list.Elements() {
		obj := e.(types.Object).Attributes()
		entry := SecretEntry{Key: str(obj[attrKey]), Version: i64(obj[attrVersion])}
		if values, ok := obj[attrValues].(types.List); ok && !values.IsNull() && !values.IsUnknown() {
			entry.Values = []SecretValue{}
			for _, ve := range values.Elements() {
				row := ve.(types.Object).Attributes()
				v := SecretValue{Pointer: str(row[attrPointer])}
				if s, ok := row[attrValue].(types.String); ok && !s.IsNull() && !s.IsUnknown() {
					val := s.ValueString()
					v.Value = &val
				}
				if b, ok := row[attrClear].(types.Bool); ok && !b.IsNull() && b.ValueBool() {
					v.Clear = true
				}
				entry.Values = append(entry.Values, v)
			}
		}
		out = append(out, entry)
	}
	return out
}

// InstanceWrite is the instance DTO the API receives: the provider key and
// the computed alias and schema version are gone, the id is echoed only for a
// matched instance, and the config is the selected child, unwrapped.
type InstanceWrite struct {
	ID     string         `json:"id,omitempty"`
	Type   string         `json:"type"`
	Name   *string        `json:"name,omitempty"`
	Config map[string]any `json:"config"`
}

// InstanceRead is what the API returns for one persisted instance.
type InstanceRead struct {
	ID            string         `json:"id"`
	Alias         string         `json:"alias"`
	Type          string         `json:"type"`
	SchemaVersion int64          `json:"schema_version"`
	Name          *string        `json:"name,omitempty"`
	Config        map[string]any `json:"config"`
}

// Project turns the planned chain into instance DTOs, joined to the prior
// identities. The entry's config child must be the one its type names and
// no other child may be set.
func (cs *ChainSchema) Project(planned types.List, correlated []Correlated) ([]InstanceWrite, diag.Diagnostics) {
	var diags diag.Diagnostics
	if planned.IsNull() || planned.IsUnknown() {
		return nil, nil
	}
	elems := planned.Elements()
	if len(elems) != len(correlated) {
		diags.AddError("Chain correlation mismatch", fmt.Sprintf("%d planned entries, %d correlated", len(elems), len(correlated)))
		return nil, diags
	}
	out := make([]InstanceWrite, 0, len(elems))
	for i, e := range elems {
		at := path.Root(AttrChain).AtListIndex(i)
		obj := e.(types.Object).Attributes()
		c := correlated[i]
		w := InstanceWrite{Type: c.Type}
		if c.Matched {
			w.ID = c.ID
		}
		if n, ok := obj[attrName].(types.String); ok && !n.IsNull() && !n.IsUnknown() {
			name := n.ValueString()
			w.Name = &name
		}
		for _, id := range cs.Catalog.TypeIDs() {
			child := obj[id]
			if id == c.Type {
				if child == nil || child.IsNull() {
					diags.AddAttributeError(at.AtName(id), "Missing config child", fmt.Sprintf("entry %q has type %s but no %s block", c.Key, c.Type, id))
					continue
				}
				wire, err := ToWire(child, at.AtName(id))
				if err != nil {
					diags.AddAttributeError(at.AtName(id), "Config not known at apply", err.Error())
					continue
				}
				w.Config, _ = wire.(map[string]any)
				continue
			}
			if child != nil && !child.IsNull() {
				diags.AddAttributeError(at.AtName(id), "Config child does not match type", fmt.Sprintf("entry %q has type %s; remove the %s block", c.Key, c.Type, id))
			}
		}
		out = append(out, w)
	}
	return out, diags
}

// EntryValue builds the state entry for one server instance under the
// provider key it correlates to.
func (cs *ChainSchema) EntryValue(ctx context.Context, key string, read InstanceRead) (types.Object, diag.Diagnostics) {
	var diags diag.Diagnostics
	attrTypes := cs.EntryType.AttributeTypes()
	values := map[string]attr.Value{
		attrKey:           types.StringValue(key),
		attrType:          types.StringValue(read.Type),
		attrName:          types.StringPointerValue(read.Name),
		attrID:            types.StringValue(read.ID),
		attrAlias:         types.StringValue(read.Alias),
		attrSchemaVersion: types.Int64Value(read.SchemaVersion),
	}
	for _, id := range cs.Catalog.TypeIDs() {
		if id != read.Type {
			values[id] = nullOf(attrTypes[id])
			continue
		}
		v, d := FromWire(ctx, attrTypes[id], read.Config, path.Root(AttrChain).AtName(id))
		diags.Append(d...)
		values[id] = v
	}
	obj, d := types.ObjectValue(attrTypes, values)
	diags.Append(d...)
	return obj, diags
}

// ChainValue builds the whole state list from server reads, in server order,
// under the given keys. Keys and reads pair by position.
func (cs *ChainSchema) ChainValue(ctx context.Context, keys []string, reads []InstanceRead) (types.List, diag.Diagnostics) {
	var diags diag.Diagnostics
	if len(keys) != len(reads) {
		diags.AddError("Chain length mismatch", fmt.Sprintf("%d keys for %d server instances", len(keys), len(reads)))
		return types.ListNull(cs.EntryType), diags
	}
	elems := make([]attr.Value, 0, len(reads))
	for i, r := range reads {
		obj, d := cs.EntryValue(ctx, keys[i], r)
		diags.Append(d...)
		elems = append(elems, obj)
	}
	list, d := types.ListValue(cs.EntryType, elems)
	diags.Append(d...)
	return list, diags
}

// ImportKeys derives a provider key for each server instance from its exact
// id. Ids are unique per connector, so the keys cannot collide, and the same
// ids always derive the same keys.
func ImportKeys(reads []InstanceRead) ([]string, error) {
	keys := make([]string, 0, len(reads))
	seen := map[string]bool{}
	for _, r := range reads {
		if r.ID == "" {
			return nil, fmt.Errorf("ImportKeys: server instance without id")
		}
		if seen[r.ID] {
			return nil, fmt.Errorf("ImportKeys: duplicate server id %q", r.ID)
		}
		seen[r.ID] = true
		keys = append(keys, r.ID)
	}
	return keys, nil
}

// FlatSMTAttributes lists the released flat attributes that address the same
// backend surface as the chain: every mapping into transforms.* or
// predicates.*. Derived from the released field mappings, not hand-listed.
func FlatSMTAttributes(mappings map[string]string) []string {
	var names []string
	for tfAttr, apiField := range mappings {
		if strings.HasPrefix(apiField, "transforms.") || strings.HasPrefix(apiField, "predicates.") {
			names = append(names, tfAttr)
		}
	}
	sort.Strings(names)
	return names
}

func str(v attr.Value) string {
	if s, ok := v.(types.String); ok && !s.IsNull() && !s.IsUnknown() {
		return s.ValueString()
	}
	return ""
}

func i64(v attr.Value) int64 {
	if n, ok := v.(types.Int64); ok && !n.IsNull() && !n.IsUnknown() {
		return n.ValueInt64()
	}
	return 0
}
