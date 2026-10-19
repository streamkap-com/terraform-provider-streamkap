package smt

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/streamkap-com/terraform-provider-streamkap/internal/api"
)

// Attribute names of the nested SMT surface on a connector resource.
const (
	AttrChain            = "smt_chain"
	AttrSecrets          = "smt_secrets"
	AttrPredicates       = "smt_predicates"
	AttrPredicateSecrets = "smt_predicate_secrets"
	AttrManagedOverrides = "smt_managed_overrides"

	attrKey                   = "key"
	attrType                  = "type"
	attrName                  = "name"
	attrEnabled               = "enabled"
	attrID                    = "id"
	attrAlias                 = "alias"
	attrSchemaVersion         = "schema_version"
	attrCodecVersion          = "codec_version"
	attrSecretVersion         = "secret_version"
	attrVersion               = "version"
	attrValues                = "values"
	attrPointer               = "pointer"
	attrValue                 = "value"
	attrClear                 = "clear"
	attrPredicate             = "predicate"
	attrNegate                = "negate"
	attrConfiguredSecretPaths = "configured_secret_paths"
)

// ChainSchema is the chain attribute derived from a catalog, with the typed
// config child of every type it admits.
type ChainSchema struct {
	Catalog            Catalog
	Children           map[string]*ConfigChild
	Attribute          schema.ListNestedAttribute
	EntryType          types.ObjectType
	PredicateCatalog   Catalog
	PredicateChildren  map[string]*ConfigChild
	PredicateAttribute schema.ListNestedAttribute
	PredicateEntryType types.ObjectType
}

// BuildChain derives smt_chain from the catalog. Each entry carries an
// immutable provider key, the type, a mutable display name the wire
// requires, the enabled flag, the computed server identity, and one config
// child per catalog type, of which exactly the one named by `type` may be
// set.
func BuildChain(cat Catalog) (*ChainSchema, error) {
	transforms := cat.ForRole("transform")
	predicates := cat.ForRole("predicate")
	cs := &ChainSchema{Catalog: transforms, Children: map[string]*ConfigChild{}, PredicateCatalog: predicates, PredicateChildren: map[string]*ConfigChild{}}
	attrs := map[string]schema.Attribute{
		attrKey: schema.StringAttribute{
			Required:    true,
			Description: "Immutable provider-side key of this instance. Changing it creates a new instance; it never changes the server id it was matched to.",
			Validators:  []validator.String{stringvalidator.LengthAtLeast(1)},
		},
		attrType: schema.StringAttribute{
			Required:    true,
			Description: typeDescription(transforms.TypeIDs()),
			Validators:  []validator.String{stringvalidator.OneOf(transforms.TypeIDs()...)},
		},
		attrName: schema.StringAttribute{
			Required:    true,
			Description: "Mutable display name.",
			Validators:  []validator.String{stringvalidator.LengthAtLeast(1)},
		},
		attrEnabled: schema.BoolAttribute{
			Optional:    true,
			Computed:    true,
			Default:     booldefault.StaticBool(true),
			Description: "Whether the instance runs. Defaults to true.",
		},
		attrID:            schema.StringAttribute{Computed: true, Description: "Server-owned instance id."},
		attrAlias:         schema.StringAttribute{Computed: true, Description: "Server-owned Kafka Connect alias."},
		attrSchemaVersion: schema.Int64Attribute{Computed: true, Description: "Catalog schema version the instance was persisted with."},
		attrCodecVersion:  schema.Int64Attribute{Computed: true, Description: "Catalog codec version the instance was persisted with."},
		attrSecretVersion: schema.Int64Attribute{Computed: true, Description: "Last rotation version applied to this instance; 0 before any rotation. Follows the key, so a removed and re-added secret entry cannot restart below it."},
		attrPredicate: schema.SingleNestedAttribute{
			Optional:    true,
			Description: "Reference to one shared predicate by provider key. Negate changes only this attachment.",
			Attributes: map[string]schema.Attribute{
				attrKey:    schema.StringAttribute{Required: true, Description: "Predicate key in " + AttrPredicates + "."},
				attrNegate: schema.BoolAttribute{Optional: true, Computed: true, Default: booldefault.StaticBool(false), Description: "Invert the predicate result for this transform. Defaults to false."},
			},
		},
	}
	for _, id := range transforms.TypeIDs() {
		spec, ok := transforms.Current("transform", id)
		if !ok {
			continue
		}
		child, err := BuildConfigChild(spec)
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

	predicateAttrs := map[string]schema.Attribute{
		attrKey:                   schema.StringAttribute{Required: true, Description: "Immutable provider-side predicate key.", Validators: []validator.String{stringvalidator.LengthAtLeast(1)}},
		attrType:                  schema.StringAttribute{Required: true, Description: typeDescription(predicates.TypeIDs()), Validators: []validator.String{stringvalidator.OneOf(predicates.TypeIDs()...)}},
		attrID:                    schema.StringAttribute{Computed: true, Description: "Server-owned predicate id."},
		attrAlias:                 schema.StringAttribute{Computed: true, Description: "Server-owned Kafka Connect predicate alias."},
		attrSchemaVersion:         schema.Int64Attribute{Computed: true, Description: "Catalog schema version the predicate was persisted with."},
		attrCodecVersion:          schema.Int64Attribute{Computed: true, Description: "Catalog codec version the predicate was persisted with."},
		attrSecretVersion:         schema.Int64Attribute{Computed: true, Description: "Last predicate secret rotation version applied under this key."},
		attrConfiguredSecretPaths: schema.ListAttribute{Computed: true, ElementType: types.StringType, Description: "Canonical secret paths configured on the server; values are never returned."},
	}
	for _, id := range predicates.TypeIDs() {
		spec, ok := predicates.Current("predicate", id)
		if !ok {
			continue
		}
		child, err := BuildConfigChild(spec)
		if err != nil {
			return nil, fmt.Errorf("BuildChain predicate %s: %w", id, err)
		}
		child.Attribute.Description = "Configuration when predicate type is " + id + "."
		predicateAttrs[id] = child.Attribute
		cs.PredicateChildren[id] = child
	}
	cs.PredicateAttribute = schema.ListNestedAttribute{
		Optional:     true,
		Computed:     true,
		Description:  "Shared immutable predicate records referenced by transform attachments.",
		NestedObject: schema.NestedAttributeObject{Attributes: predicateAttrs},
	}
	cs.PredicateEntryType = cs.PredicateAttribute.NestedObject.Type().(types.ObjectType)
	return cs, nil
}

// typeDescription lists the admissible types the way the generated enum
// attributes do, and says so when the pinned catalog publishes none.
func typeDescription(ids []string) string {
	if len(ids) == 0 {
		return "Catalog type id. Cannot change under an existing key. The pinned catalog publishes no type yet, so no value is accepted."
	}
	quoted := make([]string, len(ids))
	for i, id := range ids {
		quoted[i] = "`" + id + "`"
	}
	return "Catalog type id. Cannot change under an existing key. Valid values: " + strings.Join(quoted, ", ") + "."
}

// SecretsAttribute is the write-only secret input: per instance key, a
// stateful rotation version and a write-only list of pointer operations.
// Only the version ever reaches plan or state.
func SecretsAttribute() schema.ListNestedAttribute {
	return secretsAttribute(AttrChain)
}

func PredicateSecretsAttribute() schema.ListNestedAttribute {
	return secretsAttribute(AttrPredicates)
}

func secretsAttribute(target string) schema.ListNestedAttribute {
	return schema.ListNestedAttribute{
		Optional:    true,
		Description: "Secret values for instances, addressed by canonical pointer. Values are write-only; bump version to rotate.",
		NestedObject: schema.NestedAttributeObject{
			Attributes: map[string]schema.Attribute{
				attrKey: schema.StringAttribute{Required: true, Description: "Instance key in " + target + "."},
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
							attrValue:   schema.StringAttribute{Optional: true, WriteOnly: true, Sensitive: true, Description: "New secret value. Exactly one of value and clear is set."},
							attrClear:   schema.BoolAttribute{Optional: true, WriteOnly: true, Description: "Clear the secret instead of replacing it."},
						},
					},
				},
			},
		},
	}
}

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
			CodecVersion:  i64(obj[attrCodecVersion]),
			SecretVersion: i64(obj[attrSecretVersion]),
		})
	}
	return out
}

// SecretEntriesFromList reads smt_secrets from req.Config, the only place
// the write-only values exist. At plan any part can be unknown, for example a
// value taken from a resource created in the same apply; that part is flagged
// rather than read as null, zero or "". An unknown list or entry reads as one
// entry whose key is unknown: it may rotate any instance.
func SecretEntriesFromList(list types.List) []SecretEntry {
	if list.IsNull() {
		return nil
	}
	if list.IsUnknown() {
		return []SecretEntry{{KeyUnknown: true}}
	}
	out := make([]SecretEntry, 0, len(list.Elements()))
	for _, e := range list.Elements() {
		if e.IsUnknown() {
			out = append(out, SecretEntry{KeyUnknown: true})
			continue
		}
		obj := e.(types.Object).Attributes()
		entry := SecretEntry{
			Key:            str(obj[attrKey]),
			Version:        i64(obj[attrVersion]),
			KeyUnknown:     isUnknown(obj[attrKey]),
			VersionUnknown: isUnknown(obj[attrVersion]),
			ValuesUnknown:  isUnknown(obj[attrValues]),
		}
		if values, ok := obj[attrValues].(types.List); ok && !values.IsNull() && !values.IsUnknown() {
			entry.Values = []SecretValue{}
			for _, ve := range values.Elements() {
				if ve.IsUnknown() {
					entry.Values = append(entry.Values, SecretValue{PointerUnknown: true, ValueUnknown: true, ClearUnknown: true})
					continue
				}
				row := ve.(types.Object).Attributes()
				v := SecretValue{
					Pointer:        str(row[attrPointer]),
					PointerUnknown: isUnknown(row[attrPointer]),
					ValueUnknown:   isUnknown(row[attrValue]),
					ClearUnknown:   isUnknown(row[attrClear]),
				}
				if s, ok := row[attrValue].(types.String); ok && !s.IsNull() && !s.IsUnknown() {
					val := s.ValueString()
					v.Value = &val
				}
				if b, ok := row[attrClear].(types.Bool); ok && !b.IsNull() && !b.IsUnknown() && b.ValueBool() {
					v.Clear = true
				}
				entry.Values = append(entry.Values, v)
			}
		}
		out = append(out, entry)
	}
	return out
}

func isUnknown(v attr.Value) bool {
	return v != nil && v.IsUnknown()
}

// Project turns the planned chain into instance DTOs, joined to the prior
// identities, with each key's secret operations carried by its instance. The
// entry's config child must be the one its type names and no other child
// may be set.
func (cs *ChainSchema) Project(planned types.List, correlated []Correlated, ops []SecretOp) ([]api.SMTInstanceWrite, diag.Diagnostics) {
	var diags diag.Diagnostics
	if planned.IsNull() || planned.IsUnknown() {
		return nil, nil
	}
	elems := planned.Elements()
	if len(elems) != len(correlated) {
		diags.AddError("Chain correlation mismatch", fmt.Sprintf("%d planned entries, %d correlated", len(elems), len(correlated)))
		return nil, diags
	}
	out := make([]api.SMTInstanceWrite, 0, len(elems))
	for i := 0; i < len(elems) && i < len(correlated); i++ {
		e := elems[i]
		at := path.Root(AttrChain).AtListIndex(i)
		obj := e.(types.Object).Attributes()
		c := correlated[i]
		w := api.SMTInstanceWrite{Type: c.Type, Name: str(obj[attrName]), Enabled: true}
		if c.Matched {
			w.ID = c.ID
		}
		if b, ok := obj[attrEnabled].(types.Bool); ok && !b.IsNull() && !b.IsUnknown() {
			w.Enabled = b.ValueBool()
		}
		for _, op := range ops {
			if op.Key != c.Key {
				continue
			}
			wire := api.SMTSecretOperation{Pointer: op.Pointer, Operation: api.SMTSecretReplace, Value: op.Value}
			if op.Clear {
				wire = api.SMTSecretOperation{Pointer: op.Pointer, Operation: api.SMTSecretClear}
			}
			w.SecretOperations = append(w.SecretOperations, wire)
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

// ProjectGraph adds transform attachments after both identity namespaces have
// been correlated. A new predicate is referenced by request key; an existing
// predicate is referenced by server id.
func (cs *ChainSchema) ProjectGraph(planned types.List, correlated []Correlated, ops []SecretOp, predicates map[string]Correlated) ([]api.SMTInstanceWrite, diag.Diagnostics) {
	out, diags := cs.Project(planned, correlated, ops)
	if diags.HasError() || planned.IsNull() || planned.IsUnknown() {
		return out, diags
	}
	elements := planned.Elements()
	if len(correlated) != len(elements) {
		diags.AddAttributeError(path.Root(AttrChain), "Chain correlation is incomplete", fmt.Sprintf("%d chain entries were planned but %d were correlated", len(elements), len(correlated)))
		return out, diags
	}
	referenced := map[string]bool{}
	for i, entry := range correlated {
		attachment, ok := elements[i].(types.Object).Attributes()[attrPredicate].(types.Object)
		if !ok || attachment.IsNull() {
			continue
		}
		if attachment.IsUnknown() {
			diags.AddAttributeError(path.Root(AttrChain).AtListIndex(i).AtName(attrPredicate), "Predicate attachment is unknown", "the predicate attachment must be fully known before the chain can be written")
			continue
		}
		attrs := attachment.Attributes()
		if attrs[attrKey].IsNull() || attrs[attrKey].IsUnknown() {
			diags.AddAttributeError(path.Root(AttrChain).AtListIndex(i).AtName(attrPredicate).AtName(attrKey), "Predicate attachment key is unknown", "the predicate key must be known before the chain can be written")
			continue
		}
		if attrs[attrNegate].IsNull() || attrs[attrNegate].IsUnknown() {
			diags.AddAttributeError(path.Root(AttrChain).AtListIndex(i).AtName(attrPredicate).AtName(attrNegate), "Predicate attachment negate is unknown", "negate must be known before the chain can be written")
			continue
		}
		key := str(attrs[attrKey])
		predicate, exists := predicates[key]
		if !exists {
			diags.AddAttributeError(path.Root(AttrChain).AtListIndex(i).AtName(attrPredicate).AtName(attrKey), "Unknown predicate key", fmt.Sprintf("predicate key %q is not in %s", key, AttrPredicates))
			continue
		}
		referenced[key] = true
		negate := boolValue(attrs[attrNegate])
		predicateSpec, ok := cs.PredicateCatalog.Current("predicate", predicate.Type)
		if predicate.Matched {
			var pinDiags diag.Diagnostics
			predicateSpec, pinDiags = cs.pinnedChild("predicate", CatalogPin{
				Role:          "predicate",
				TypeID:        predicate.Type,
				SchemaVersion: predicate.SchemaVersion,
				CodecVersion:  predicate.CodecVersion,
			})
			diags.Append(pinDiags...)
			ok = !pinDiags.HasError()
		}
		if ok {
			validatePredicateAttachmentSpec(&diags, path.Root(AttrChain).AtListIndex(i).AtName(attrPredicate), entry.Type, predicateSpec, negate)
		}
		ref := &api.SMTPredicateAttachment{Negate: negate}
		if predicate.Matched {
			ref.ID = predicate.ID
		} else {
			ref.RequestKey = predicate.Key
		}
		out[i].Predicate = ref
	}
	keys := make([]string, 0, len(predicates))
	for key := range predicates {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		if !referenced[key] {
			diags.AddAttributeError(path.Root(AttrPredicates), "Predicate has no attachment", fmt.Sprintf("predicate key %q is not referenced by any transform; remove it from %s in the same plan", key, AttrPredicates))
		}
	}
	return out, diags
}

// ProjectPredicates builds the immutable predicate pool. Matched records are
// retained by id; new records carry their typed config and one-shot secrets.
func (cs *ChainSchema) ProjectPredicates(planned types.List, correlated []Correlated, ops []SecretOp) ([]api.SMTPredicateWrite, diag.Diagnostics) {
	var diags diag.Diagnostics
	if planned.IsNull() {
		return []api.SMTPredicateWrite{}, nil
	}
	if planned.IsUnknown() {
		return nil, nil
	}
	if len(planned.Elements()) != len(correlated) {
		diags.AddError("Predicate correlation mismatch", fmt.Sprintf("%d planned entries, %d correlated", len(planned.Elements()), len(correlated)))
		return nil, diags
	}
	out := make([]api.SMTPredicateWrite, 0, len(correlated))
	for i, c := range correlated {
		if c.Matched {
			out = append(out, api.SMTPredicateWrite{ID: c.ID})
			continue
		}
		obj := planned.Elements()[i].(types.Object).Attributes()
		at := path.Root(AttrPredicates).AtListIndex(i)
		write := api.SMTPredicateWrite{RequestKey: c.Key, Type: c.Type}
		child := obj[c.Type]
		if child == nil || child.IsNull() {
			diags.AddAttributeError(at.AtName(c.Type), "Missing predicate config child", fmt.Sprintf("predicate %q has type %s but no %s block", c.Key, c.Type, c.Type))
		} else if wire, err := ToWire(child, at.AtName(c.Type)); err != nil {
			diags.AddAttributeError(at.AtName(c.Type), "Predicate config not known at apply", err.Error())
		} else {
			write.Config, _ = wire.(map[string]any)
		}
		for _, op := range ops {
			if op.Key != c.Key {
				continue
			}
			wire := api.SMTSecretOperation{Pointer: op.Pointer, Operation: api.SMTSecretReplace, Value: op.Value}
			if op.Clear {
				wire = api.SMTSecretOperation{Pointer: op.Pointer, Operation: api.SMTSecretClear}
			}
			write.SecretOperations = append(write.SecretOperations, wire)
		}
		out = append(out, write)
	}
	return out, diags
}

// EntryValue builds the state entry for one server instance under the
// provider key it correlates to. secretVersion is provider-owned state the
// server never sees.
func (cs *ChainSchema) EntryValue(ctx context.Context, key string, read api.SMTInstanceRead, secretVersion int64) (types.Object, diag.Diagnostics) {
	return cs.entryValue(ctx, key, read, secretVersion, nil)
}

func (cs *ChainSchema) entryValue(ctx context.Context, key string, read api.SMTInstanceRead, secretVersion int64, predicateKeys map[string]string) (types.Object, diag.Diagnostics) {
	var diags diag.Diagnostics
	attrTypes := cs.EntryType.AttributeTypes()
	values := map[string]attr.Value{
		attrKey:           types.StringValue(key),
		attrType:          types.StringValue(read.Type),
		attrName:          types.StringValue(read.Name),
		attrEnabled:       types.BoolValue(read.Enabled),
		attrID:            types.StringValue(read.ID),
		attrAlias:         types.StringValue(read.Alias),
		attrSchemaVersion: types.Int64Value(read.SchemaVersion),
		attrCodecVersion:  types.Int64Value(read.CodecVersion),
		attrSecretVersion: types.Int64Value(secretVersion),
	}
	values[attrPredicate] = nullOf(attrTypes[attrPredicate])
	if read.HasPredicate() {
		var attachment api.SMTPredicateAttachment
		if err := json.Unmarshal(read.Predicate, &attachment); err != nil {
			diags.AddAttributeError(path.Root(AttrChain), "Invalid predicate attachment", err.Error())
		} else if predicateKey, ok := predicateKeys[attachment.ID]; !ok {
			diags.AddAttributeError(path.Root(AttrChain), "Unknown predicate attachment", fmt.Sprintf("transform %s references predicate id %q outside the returned predicate pool", read.ID, attachment.ID))
		} else {
			attachmentType := attrTypes[attrPredicate].(types.ObjectType)
			v, d := types.ObjectValue(attachmentType.AttributeTypes(), map[string]attr.Value{attrKey: types.StringValue(predicateKey), attrNegate: types.BoolValue(attachment.Negate)})
			diags.Append(d...)
			values[attrPredicate] = v
		}
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
// under the given keys. Keys and reads pair by position; versions holds the
// last applied rotation version per key, absent meaning never rotated.
func (cs *ChainSchema) ChainValue(ctx context.Context, keys []string, reads []api.SMTInstanceRead, versions map[string]int64) (types.List, diag.Diagnostics) {
	return cs.chainValue(ctx, keys, reads, versions, nil)
}

func (cs *ChainSchema) chainValue(ctx context.Context, keys []string, reads []api.SMTInstanceRead, versions map[string]int64, predicateKeys map[string]string) (types.List, diag.Diagnostics) {
	var diags diag.Diagnostics
	if len(keys) != len(reads) {
		diags.AddError("Chain length mismatch", fmt.Sprintf("%d keys for %d server instances", len(keys), len(reads)))
		return types.ListNull(cs.EntryType), diags
	}
	elems := make([]attr.Value, 0, len(reads))
	for i, r := range reads {
		obj, d := cs.entryValue(ctx, keys[i], r, versions[keys[i]], predicateKeys)
		diags.Append(d...)
		elems = append(elems, obj)
	}
	list, d := types.ListValue(cs.EntryType, elems)
	diags.Append(d...)
	return list, diags
}

// PredicateValue builds predicate state without secret values. The configured
// path list records only server-owned presence needed to distinguish retained
// records from fresh copy-on-write records.
func (cs *ChainSchema) PredicateValue(ctx context.Context, key string, read api.SMTPredicateRead, secretVersion int64) (types.Object, diag.Diagnostics) {
	var diags diag.Diagnostics
	attrTypes := cs.PredicateEntryType.AttributeTypes()
	configured, d := types.ListValueFrom(ctx, types.StringType, read.ConfiguredSecretPaths)
	diags.Append(d...)
	values := map[string]attr.Value{
		attrKey: types.StringValue(key), attrType: types.StringValue(read.Type), attrID: types.StringValue(read.ID), attrAlias: types.StringValue(read.Alias),
		attrSchemaVersion: types.Int64Value(read.SchemaVersion), attrCodecVersion: types.Int64Value(read.CodecVersion), attrSecretVersion: types.Int64Value(secretVersion), attrConfiguredSecretPaths: configured,
	}
	for _, id := range cs.PredicateCatalog.TypeIDs() {
		if id != read.Type {
			values[id] = nullOf(attrTypes[id])
			continue
		}
		v, childDiags := FromWire(ctx, attrTypes[id], read.Config, path.Root(AttrPredicates).AtName(id))
		diags.Append(childDiags...)
		values[id] = v
	}
	obj, objectDiags := types.ObjectValue(attrTypes, values)
	diags.Append(objectDiags...)
	return obj, diags
}

func (cs *ChainSchema) PredicatesValue(ctx context.Context, keys []string, reads []api.SMTPredicateRead, versions map[string]int64) (types.List, diag.Diagnostics) {
	var diags diag.Diagnostics
	if len(keys) != len(reads) {
		diags.AddError("Predicate length mismatch", fmt.Sprintf("%d keys for %d predicate records", len(keys), len(reads)))
		return types.ListNull(cs.PredicateEntryType), diags
	}
	elems := make([]attr.Value, 0, len(reads))
	for i, read := range reads {
		obj, d := cs.PredicateValue(ctx, keys[i], read, versions[keys[i]])
		diags.Append(d...)
		elems = append(elems, obj)
	}
	list, d := types.ListValue(cs.PredicateEntryType, elems)
	diags.Append(d...)
	return list, diags
}

func (cs *ChainSchema) pinnedChild(role string, pin CatalogPin) (*TypeSpec, diag.Diagnostics) {
	var diags diag.Diagnostics
	catalog := cs.Catalog
	children := cs.Children
	if role == "predicate" {
		catalog = cs.PredicateCatalog
		children = cs.PredicateChildren
	}
	spec, ok := catalog.Lookup(pin)
	current := children[pin.TypeID]
	currentSpec, currentOK := catalog.Current(role, pin.TypeID)
	if !ok || current == nil || !currentOK {
		diags.AddError("Pinned "+role+" artifact cannot be represented", fmt.Sprintf("catalog pin %s/%s schema %d codec %d is not available in this provider", pin.Role, pin.TypeID, pin.SchemaVersion, pin.CodecVersion))
		return nil, diags
	}
	historical, err := BuildConfigChild(spec)
	semanticsMatch := err == nil && retainedSemantics(spec, historical) == retainedSemantics(currentSpec, current)
	if err != nil || !historical.Attribute.GetType().Equal(current.Attribute.GetType()) || !semanticsMatch {
		detail := "its Terraform shape differs from the current authoring schema"
		if err != nil {
			detail = err.Error()
		} else if !semanticsMatch {
			detail = "its decoding and write semantics differ from the current authoring schema"
		}
		diags.AddError("Pinned "+role+" artifact cannot be represented", fmt.Sprintf("catalog pin %s/%s schema %d codec %d cannot be decoded safely: %s", pin.Role, pin.TypeID, pin.SchemaVersion, pin.CodecVersion, detail))
		return nil, diags
	}
	return spec, diags
}

func retainedSemantics(spec *TypeSpec, child *ConfigChild) string {
	encoded, _ := json.Marshal(struct {
		Schema          string `json:"schema"`
		Secrets         int    `json:"secrets"`
		RequiredSecrets int    `json:"required_secrets"`
	}{
		Schema:          spec.schemaSemantics,
		Secrets:         len(child.SecretTemplates),
		RequiredSecrets: len(child.RequiredSecretTemplates),
	})
	return string(encoded)
}

// ImportKeys derives a provider key for each server instance from its exact
// id. Ids are unique per connector, so the keys cannot collide, and the same
// ids always derive the same keys.
func ImportKeys(reads []api.SMTInstanceRead) ([]string, error) {
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

func boolValue(v attr.Value) bool {
	if b, ok := v.(types.Bool); ok && !b.IsNull() && !b.IsUnknown() {
		return b.ValueBool()
	}
	return false
}
