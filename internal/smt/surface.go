package smt

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/streamkap-com/terraform-provider-streamkap/internal/api"
)

// AttrRevision is the computed chain revision the last read returned; an
// update supplies it as the expected revision.
const AttrRevision = "smt_chain_revision"

// RevisionAttribute is the computed revision attribute. It is pinned by
// ModifyPlan and never populated for a connector whose chain is null.
func RevisionAttribute() schema.StringAttribute {
	return schema.StringAttribute{
		Computed:    true,
		Description: "Chain revision returned by the last read. An update supplies it as the expected revision, so an edit made elsewhere since the last refresh is rejected instead of overwritten.",
	}
}

func ManagedOverridesAttribute() schema.MapAttribute {
	description := "Explicit overrides for optional managed transform slots, keyed by the stable slot id. Omit this attribute to preserve server ownership; set an empty map to restore automatic behavior."
	return schema.MapAttribute{
		Optional:            true,
		Computed:            true,
		ElementType:         types.BoolType,
		Description:         description,
		MarkdownDescription: description,
	}
}

// Surface is the chain behaviour a connector resource composes into its
// lifecycle once its generated schema opts in: the typed chain, the secret
// input, the revision, and the list of released flat attributes that address
// the same backend surface.
type Surface struct {
	Chain *ChainSchema
	// FlatSMT is every released attribute mapped into transforms.* or
	// predicates.*, derived from the runtime field mappings so hand-maintained
	// aliases are classified too.
	FlatSMT []string
	// apiKeys is the wire key of every FlatSMT attribute, the set the flat
	// projection drops when the chain owns the surface.
	apiKeys map[string]bool
}

// NewSurface derives the flat-attribute classification from the runtime
// mappings and pairs it with the chain schema.
func NewSurface(chain *ChainSchema, mappings map[string]string) *Surface {
	s := &Surface{Chain: chain, FlatSMT: FlatSMTAttributes(mappings), apiKeys: map[string]bool{}}
	for _, name := range s.FlatSMT {
		s.apiKeys[mappings[name]] = true
	}
	return s
}

// AddAttributes puts the chain, the secret input and the revision into a
// connector schema.
func (s *Surface) AddAttributes(attrs map[string]schema.Attribute) {
	attrs[AttrChain] = s.Chain.Attribute
	attrs[AttrSecrets] = SecretsAttribute()
	attrs[AttrPredicates] = s.Chain.PredicateAttribute
	attrs[AttrPredicateSecrets] = PredicateSecretsAttribute()
	attrs[AttrManagedOverrides] = ManagedOverridesAttribute()
	attrs[AttrRevision] = RevisionAttribute()
}

// StripFlat removes every flat transforms.* and predicates.* key from the
// wire config. The released flat attributes carry client-side defaults that
// are present in every plan, so when the chain owns the surface they must not
// reach the API at all.
func (s *Surface) StripFlat(configMap map[string]any) {
	for key := range configMap {
		if s.apiKeys[key] {
			delete(configMap, key)
		}
	}
}

// ValidateConfig rejects a configuration that drives the same backend surface
// from both the released flat attributes and the chain, a chain with missing
// or duplicate keys, and a secret entry whose key is not in the chain. All of
// it runs before any plan or request.
func (s *Surface) ValidateConfig(ctx context.Context, cfg tfsdk.Config) diag.Diagnostics {
	var diags diag.Diagnostics
	var chain types.List
	diags.Append(cfg.GetAttribute(ctx, path.Root(AttrChain), &chain)...)
	if diags.HasError() {
		return diags
	}
	if !chain.IsNull() {
		for _, name := range s.FlatSMT {
			var v attr.Value
			diags.Append(cfg.GetAttribute(ctx, path.Root(name), &v)...)
			if v != nil && !v.IsNull() {
				diags.AddAttributeError(path.Root(name),
					"Flat and nested transform attributes cannot be mixed",
					fmt.Sprintf("%s addresses the transform chain through the released flat attributes while %s is also configured. Move it into a %s entry or remove %s.", name, AttrChain, AttrChain, AttrChain))
			}
		}
	}
	if !instanceIdentityKnown(chain) {
		return diags
	}
	planned := PlannedFromList(chain)
	_, d := Correlate(nil, planned)
	diags.Append(d...)
	var predicates types.List
	diags.Append(cfg.GetAttribute(ctx, path.Root(AttrPredicates), &predicates)...)
	if predicates.IsUnknown() {
		return diags
	}
	if chain.IsNull() && !predicates.IsNull() {
		diags.AddAttributeError(path.Root(AttrPredicates), "Predicate graph needs a managed chain", fmt.Sprintf("configure %s when %s is configured", AttrChain, AttrPredicates))
	}
	predicatePlanned := PlannedFromList(predicates)
	predicateCorrelated, predicateDiags := correlateAt(AttrPredicates, nil, predicatePlanned)
	diags.Append(predicateDiags...)
	predicateKeys := map[string]bool{}
	predicatesByKey := map[string]Correlated{}
	for _, predicate := range predicateCorrelated {
		predicateKeys[predicate.Key] = true
		predicatesByKey[predicate.Key] = predicate
	}
	if !chain.IsNull() && !chain.IsUnknown() {
		for i, value := range chain.Elements() {
			attachment, ok := value.(types.Object).Attributes()[attrPredicate].(types.Object)
			if !ok || attachment.IsNull() || attachment.IsUnknown() {
				continue
			}
			key := str(attachment.Attributes()[attrKey])
			if !predicateKeys[key] {
				diags.AddAttributeError(path.Root(AttrChain).AtListIndex(i).AtName(attrPredicate).AtName(attrKey), "Unknown predicate key", fmt.Sprintf("predicate key %q is not in %s", key, AttrPredicates))
				continue
			}
			predicate := predicatesByKey[key]
			validatePredicateAttachment(&diags, path.Root(AttrChain).AtListIndex(i).AtName(attrPredicate), planned[i].Type, predicate.Type, boolValue(attachment.Attributes()[attrNegate]), s.Chain.PredicateCatalog)
		}
	}

	var secrets types.List
	diags.Append(cfg.GetAttribute(ctx, path.Root(AttrSecrets), &secrets)...)
	keys := map[string]bool{}
	for _, p := range planned {
		keys[p.Key] = true
	}
	if !secrets.IsNull() && !secrets.IsUnknown() && !chain.IsUnknown() {
		for i, e := range SecretEntriesFromList(secrets) {
			if e.Key != "" && !keys[e.Key] {
				diags.AddAttributeError(path.Root(AttrSecrets).AtListIndex(i).AtName(attrKey),
					"Secret entry for unknown instance", fmt.Sprintf("instance key %q is not in %s", e.Key, AttrChain))
			}
		}
	}
	var predicateSecrets types.List
	diags.Append(cfg.GetAttribute(ctx, path.Root(AttrPredicateSecrets), &predicateSecrets)...)
	for i, e := range SecretEntriesFromList(predicateSecrets) {
		if e.Key != "" && !predicateKeys[e.Key] {
			diags.AddAttributeError(path.Root(AttrPredicateSecrets).AtListIndex(i).AtName(attrKey), "Secret entry for unknown predicate", fmt.Sprintf("predicate key %q is not in %s", e.Key, AttrPredicates))
		}
	}
	return diags
}

// ModifyPlan correlates the planned chain to prior state by key, pins the
// matched server identities and the rotation versions into the plan, rejects
// a type change under an existing key and any invalid rotation before a
// request can be sent, nulls the flat attributes the chain supersedes, and
// pins the revision when the apply cannot move it.
func (s *Surface) ModifyPlan(ctx context.Context, cfg tfsdk.Config, state tfsdk.State, plan *tfsdk.Plan) diag.Diagnostics {
	var diags diag.Diagnostics
	var planned, prior types.List
	var priorRevision types.String
	priorOverrides := types.MapNull(types.BoolType)
	diags.Append(cfg.GetAttribute(ctx, path.Root(AttrChain), &planned)...)
	if !state.Raw.IsNull() {
		diags.Append(state.GetAttribute(ctx, path.Root(AttrChain), &prior)...)
		diags.Append(state.GetAttribute(ctx, path.Root(AttrManagedOverrides), &priorOverrides)...)
		diags.Append(state.GetAttribute(ctx, path.Root(AttrRevision), &priorRevision)...)
	} else {
		priorRevision = types.StringNull()
	}
	if diags.HasError() || !instanceIdentityKnown(planned) {
		return diags
	}
	var configuredOverrides types.Map
	diags.Append(cfg.GetAttribute(ctx, path.Root(AttrManagedOverrides), &configuredOverrides)...)
	if diags.HasError() {
		return diags
	}
	if configuredOverrides.IsNull() {
		diags.Append(plan.SetAttribute(ctx, path.Root(AttrManagedOverrides), priorOverrides)...)
	}
	if planned.IsNull() {
		// The graph is not managed here. The predicate attribute's empty-list
		// default must not turn a released null graph into partial ownership.
		diags.Append(plan.SetAttribute(ctx, path.Root(AttrPredicates), types.ListNull(s.Chain.PredicateEntryType))...)
		if configuredOverrides.IsNull() {
			// The revision follows owned chain state. It remains null when neither
			// the user graph nor managed-slot overrides are configured.
			diags.Append(plan.SetAttribute(ctx, path.Root(AttrRevision), types.StringNull())...)
		} else if !priorOverrides.IsNull() && configuredOverrides.Equal(priorOverrides) {
			diags.Append(plan.SetAttribute(ctx, path.Root(AttrRevision), priorRevision)...)
		}
		return diags
	}
	// smt_predicates carries no schema default: a default differs from the
	// null a released state holds, which makes the framework plan every
	// computed null attribute unknown and the released resource never
	// converges. An owned graph with no configured predicates owns none.
	var configuredPredicates types.List
	diags.Append(cfg.GetAttribute(ctx, path.Root(AttrPredicates), &configuredPredicates)...)
	if diags.HasError() {
		return diags
	}
	if configuredPredicates.IsNull() {
		diags.Append(plan.SetAttribute(ctx, path.Root(AttrPredicates), types.ListValueMust(s.Chain.PredicateEntryType, []attr.Value{}))...)
	}

	correlated, d := Correlate(PriorFromList(prior), PlannedFromList(planned))
	diags.Append(d...)
	if diags.HasError() {
		return diags
	}
	for i, c := range correlated {
		if !c.Matched {
			continue
		}
		at := path.Root(AttrChain).AtListIndex(i)
		diags.Append(plan.SetAttribute(ctx, at.AtName(attrID), types.StringValue(c.ID))...)
		diags.Append(plan.SetAttribute(ctx, at.AtName(attrAlias), types.StringValue(c.Alias))...)
		diags.Append(plan.SetAttribute(ctx, at.AtName(attrSchemaVersion), types.Int64Value(c.SchemaVersion))...)
		diags.Append(plan.SetAttribute(ctx, at.AtName(attrCodecVersion), types.Int64Value(c.CodecVersion))...)
	}
	var plannedPredicates, priorPredicates types.List
	diags.Append(cfg.GetAttribute(ctx, path.Root(AttrPredicates), &plannedPredicates)...)
	if !state.Raw.IsNull() {
		diags.Append(state.GetAttribute(ctx, path.Root(AttrPredicates), &priorPredicates)...)
	}
	if diags.HasError() || !instanceIdentityKnown(plannedPredicates) {
		return diags
	}
	predicateCorrelated, predicateDiags := correlateAt(AttrPredicates, PriorFromList(priorPredicates), PlannedFromList(plannedPredicates))
	diags.Append(predicateDiags...)
	if diags.HasError() {
		return diags
	}
	priorPredicateValues := valuesByKey(priorPredicates)
	for i, c := range predicateCorrelated {
		if !c.Matched {
			continue
		}
		at := path.Root(AttrPredicates).AtListIndex(i)
		diags.Append(plan.SetAttribute(ctx, at.AtName(attrID), types.StringValue(c.ID))...)
		diags.Append(plan.SetAttribute(ctx, at.AtName(attrAlias), types.StringValue(c.Alias))...)
		diags.Append(plan.SetAttribute(ctx, at.AtName(attrSchemaVersion), types.Int64Value(c.SchemaVersion))...)
		diags.Append(plan.SetAttribute(ctx, at.AtName(attrCodecVersion), types.Int64Value(c.CodecVersion))...)
		if priorObject, exists := priorPredicateValues[c.Key]; exists {
			diags.Append(plan.SetAttribute(ctx, at.AtName(attrConfiguredSecretPaths), priorObject[attrConfiguredSecretPaths])...)
		}
	}
	diags.Append(validateRetainedPredicateConfigs(plannedPredicates, priorPredicates, predicateCorrelated)...)

	entries, d := secretInputs(ctx, cfg)
	diags.Append(d...)
	if diags.HasError() {
		return diags
	}
	ops, next, pending, d := planSecretOpsPinnedAt(AttrSecrets, AttrChain, secretVersions(correlated), entries, chainPins(correlated, "transform", s.Chain.Catalog), s.Chain.Catalog)
	diags.Append(d...)
	if diags.HasError() {
		return diags
	}
	// The rotation version an apply will record is known now: the prior one
	// unless this plan rotates the key, and 0 for a key never rotated. It is
	// unknown while the secret input that decides it is.
	for i, c := range correlated {
		at := path.Root(AttrChain).AtListIndex(i)
		diags.Append(plan.SetAttribute(ctx, at.AtName(attrSecretVersion), plannedSecretVersion(next, pending, c.Key))...)
	}
	predicateEntries, predicateInputDiags := secretInputsAt(ctx, cfg, AttrPredicateSecrets)
	diags.Append(predicateInputDiags...)
	predicateOps, predicateNext, predicatePending, predicateSecretDiags := planSecretOpsPinnedAt(AttrPredicateSecrets, AttrPredicates, secretVersions(predicateCorrelated), predicateEntries, chainPins(predicateCorrelated, "predicate", s.Chain.PredicateCatalog), s.Chain.PredicateCatalog)
	diags.Append(predicateSecretDiags...)
	diags.Append(refuseRetainedPredicateSecrets(predicateCorrelated, predicateOps)...)
	for i, c := range predicateCorrelated {
		at := path.Root(AttrPredicates).AtListIndex(i)
		diags.Append(plan.SetAttribute(ctx, at.AtName(attrSecretVersion), plannedSecretVersion(predicateNext, predicatePending, c.Key))...)
		if !c.Matched && !predicatePending.opsUnknown(c.Key) {
			requireNewPredicateSecrets(&diags, at, s.Chain.PredicateChildren[c.Type], c.Key, predicateOps)
		}
	}
	// A pointer must name a row the planned config actually has. Skipped
	// while the child is still unknown; the apply repeats the check.
	byKey := map[string]types.Object{}
	for _, e := range planned.Elements() {
		obj := e.(types.Object)
		byKey[str(obj.Attributes()[attrKey])] = obj
	}
	for _, op := range ops {
		obj := byKey[op.Key]
		typeID := str(obj.Attributes()[attrType])
		child := obj.Attributes()[typeID]
		if child == nil || child.IsNull() || child.IsUnknown() {
			continue
		}
		wire, err := ToWire(child, path.Root(AttrChain))
		if err != nil {
			continue
		}
		pin := chainPins(correlated, "transform", s.Chain.Catalog)[op.Key]
		spec, ok := s.Chain.Catalog.Lookup(pin)
		if !ok {
			continue
		}
		addr, _ := ResolvePointer(spec, op.Pointer)
		if !RowExists(spec, addr, wire.(map[string]any)) {
			diags.AddAttributeError(path.Root(AttrSecrets), "Secret pointer names a missing row",
				fmt.Sprintf("%s does not exist in the planned config of %q", op.Pointer, op.Key))
		}
	}

	// The flat attributes the chain supersedes are never sent and never read
	// back, so an unknown one would only ever resolve to null.
	for _, name := range s.FlatSMT {
		var v attr.Value
		diags.Append(plan.GetAttribute(ctx, path.Root(name), &v)...)
		if v != nil && v.IsUnknown() {
			diags.Append(plan.SetAttribute(ctx, path.Root(name), nullOf(v.Type(ctx)))...)
		}
	}

	// The revision only moves when the chain is written or a secret rotates.
	var pinned types.List
	diags.Append(plan.GetAttribute(ctx, path.Root(AttrChain), &pinned)...)
	var pinnedPredicates types.List
	diags.Append(plan.GetAttribute(ctx, path.Root(AttrPredicates), &pinnedPredicates)...)
	overridesUnchanged := configuredOverrides.IsNull() || (!priorOverrides.IsNull() && configuredOverrides.Equal(priorOverrides))
	if !diags.HasError() && len(ops) == 0 && len(predicateOps) == 0 && pending.none() && predicatePending.none() && !prior.IsNull() && pinned.Equal(prior) && pinnedPredicates.Equal(priorPredicates) && overridesUnchanged {
		diags.Append(plan.SetAttribute(ctx, path.Root(AttrRevision), priorRevision)...)
	}
	return diags
}

func instanceIdentityKnown(chain types.List) bool {
	if chain.IsUnknown() {
		return false
	}
	if chain.IsNull() {
		return true
	}
	for _, value := range chain.Elements() {
		if value.IsUnknown() {
			return false
		}
		if value.IsNull() {
			continue
		}
		entry := value.(types.Object).Attributes()
		if entry[attrKey].IsUnknown() || entry[attrType].IsUnknown() {
			return false
		}
	}
	return true
}

// Write is what one apply sends for the chain and optional managed-slot overrides.
type Write struct {
	// Configured is false when neither the user chain nor managed overrides are
	// configured: nothing is written and both surfaces stay unmanaged.
	Configured bool
	// ChainConfigured distinguishes user-graph ownership from an override-only
	// write, which must omit transforms and predicates on the wire.
	ChainConfigured bool
	// Unchanged is true when the planned chain equals the chain in state and
	// no secret rotates, so nothing is sent and the revision stays.
	Unchanged           bool
	Transforms          []api.SMTInstanceWrite
	Predicates          []api.SMTPredicateWrite
	ManagedOverrides    *map[string]bool
	plannedOverrides    types.Map
	planned             types.List
	correlated          []Correlated
	versions            map[string]int64
	plannedPredicates   types.List
	predicateCorrelated []Correlated
	predicateVersions   map[string]int64
}

// PlanWrite projects the planned graph with its secret operations. Both prior
// lists come from state and are null on create.
func (s *Surface) PlanWrite(ctx context.Context, plan tfsdk.Plan, prior types.List, cfg tfsdk.Config, priorPredicates types.List, priorOverrides types.Map) (*Write, diag.Diagnostics) {
	var diags diag.Diagnostics
	var planned types.List
	diags.Append(plan.GetAttribute(ctx, path.Root(AttrChain), &planned)...)
	if diags.HasError() {
		return nil, diags
	}
	var configuredOverrides types.Map
	diags.Append(cfg.GetAttribute(ctx, path.Root(AttrManagedOverrides), &configuredOverrides)...)
	if diags.HasError() {
		return nil, diags
	}
	if configuredOverrides.IsUnknown() {
		diags.AddAttributeError(path.Root(AttrManagedOverrides), "Managed overrides are unknown", "managed transform overrides must be fully known before the chain can be written")
		return nil, diags
	}
	stateOverrides := priorOverrides
	if !configuredOverrides.IsNull() {
		stateOverrides = configuredOverrides
	}
	w := &Write{ChainConfigured: !planned.IsNull(), planned: planned, plannedOverrides: stateOverrides, plannedPredicates: types.ListNull(s.Chain.PredicateEntryType)}
	if !configuredOverrides.IsNull() {
		overrides := map[string]bool{}
		for key, value := range configuredOverrides.Elements() {
			boolean, ok := value.(types.Bool)
			if !ok || boolean.IsNull() || boolean.IsUnknown() {
				diags.AddAttributeError(path.Root(AttrManagedOverrides), "Managed override is unknown", fmt.Sprintf("the override for slot %q must be a known boolean", key))
				continue
			}
			overrides[key] = boolean.ValueBool()
		}
		w.ManagedOverrides = &overrides
	}
	var plannedPredicates types.List
	diags.Append(plan.GetAttribute(ctx, path.Root(AttrPredicates), &plannedPredicates)...)
	if diags.HasError() {
		return nil, diags
	}
	if plannedPredicates.IsUnknown() {
		diags.AddAttributeError(path.Root(AttrPredicates), "Predicate graph is unknown", "the predicate graph must be fully known before the chain can be written")
		return nil, diags
	}
	if planned.IsNull() {
		if !plannedPredicates.IsNull() && len(plannedPredicates.Elements()) > 0 {
			diags.AddAttributeError(path.Root(AttrPredicates), "Predicate graph needs a managed chain", fmt.Sprintf("set %s before authoring shared predicates", AttrChain))
		}
		w.Configured = w.ManagedOverrides != nil
		w.Unchanged = w.Configured && !priorOverrides.IsNull() && configuredOverrides.Equal(priorOverrides)
		return w, diags
	}
	w.Configured = true
	w.plannedPredicates = plannedPredicates
	var d diag.Diagnostics
	w.correlated, d = Correlate(PriorFromList(prior), PlannedFromList(planned))
	diags.Append(d...)
	if w.plannedPredicates.IsNull() {
		diags.AddAttributeError(path.Root(AttrPredicates), "Predicate graph must be explicit", fmt.Sprintf("set %s to [] when the managed chain has no predicates; null leaves predicate ownership ambiguous", AttrPredicates))
		return nil, diags
	}
	w.predicateCorrelated, d = correlateAt(AttrPredicates, PriorFromList(priorPredicates), PlannedFromList(w.plannedPredicates))
	diags.Append(d...)
	diags.Append(validateRetainedPredicateConfigs(w.plannedPredicates, priorPredicates, w.predicateCorrelated)...)
	entries, d := secretInputs(ctx, cfg)
	diags.Append(d...)
	if diags.HasError() {
		return nil, diags
	}
	ops, versions, pending, d := planSecretOpsPinnedAt(AttrSecrets, AttrChain, secretVersions(w.correlated), entries, chainPins(w.correlated, "transform", s.Chain.Catalog), s.Chain.Catalog)
	diags.Append(d...)
	refusePendingSecrets(&diags, AttrSecrets, pending)
	w.versions = versions
	predicateEntries, predicateEntryDiags := secretInputsAt(ctx, cfg, AttrPredicateSecrets)
	diags.Append(predicateEntryDiags...)
	predicateOps, predicateVersions, predicatePending, predicateDiags := planSecretOpsPinnedAt(AttrPredicateSecrets, AttrPredicates, secretVersions(w.predicateCorrelated), predicateEntries, chainPins(w.predicateCorrelated, "predicate", s.Chain.PredicateCatalog), s.Chain.PredicateCatalog)
	diags.Append(predicateDiags...)
	refusePendingSecrets(&diags, AttrPredicateSecrets, predicatePending)
	diags.Append(refuseRetainedPredicateSecrets(w.predicateCorrelated, predicateOps)...)
	w.predicateVersions = predicateVersions
	for i, c := range w.predicateCorrelated {
		if !c.Matched {
			requireNewPredicateSecrets(&diags, path.Root(AttrPredicates).AtListIndex(i), s.Chain.PredicateChildren[c.Type], c.Key, predicateOps)
		}
	}
	predicateByKey := make(map[string]Correlated, len(w.predicateCorrelated))
	for _, predicate := range w.predicateCorrelated {
		predicateByKey[predicate.Key] = predicate
	}
	w.Predicates, d = s.Chain.ProjectPredicates(w.plannedPredicates, w.predicateCorrelated, predicateOps)
	diags.Append(d...)
	w.Transforms, d = s.Chain.ProjectGraph(planned, w.correlated, ops, predicateByKey)
	diags.Append(d...)
	if diags.HasError() {
		return nil, diags
	}
	graphUnchanged := !prior.IsNull() && planned.Equal(prior) && w.plannedPredicates.Equal(priorPredicates) && len(ops) == 0 && len(predicateOps) == 0
	overridesUnchanged := w.ManagedOverrides == nil || (!priorOverrides.IsNull() && configuredOverrides.Equal(priorOverrides))
	w.Unchanged = graphUnchanged && overridesUnchanged
	return w, nil
}

func plannedSecretVersion(next map[string]int64, pending secretsPending, key string) types.Int64 {
	if pending.versionUnknown(key) {
		return types.Int64Unknown()
	}
	return types.Int64Value(next[key])
}

// refusePendingSecrets stops an apply whose secret input is still not fully
// known; Terraform resolves configuration before apply, so this never sends a
// placeholder value.
func refusePendingSecrets(diags *diag.Diagnostics, attribute string, pending secretsPending) {
	if !pending.none() {
		diags.AddAttributeError(path.Root(attribute), "Secret input unknown at apply", fmt.Sprintf("every key, version, pointer, value and clear in %s must be known before the chain can be written", attribute))
	}
}

func refuseRetainedPredicateSecrets(correlated []Correlated, ops []SecretOp) diag.Diagnostics {
	var diags diag.Diagnostics
	retained := map[string]bool{}
	for _, predicate := range correlated {
		retained[predicate.Key] = predicate.Matched
	}
	for _, op := range ops {
		if retained[op.Key] {
			diags.AddAttributeError(path.Root(AttrPredicateSecrets), "Predicate records are immutable", fmt.Sprintf("predicate %q already identifies a server record. Add a new predicate key and provide that new identity's replacement secrets.", op.Key))
		}
	}
	return diags
}

func validateRetainedPredicateConfigs(planned, prior types.List, correlated []Correlated) diag.Diagnostics {
	var diags diag.Diagnostics
	priorValues := valuesByKey(prior)
	for i, c := range correlated {
		if !c.Matched {
			continue
		}
		plannedObject := planned.Elements()[i].(types.Object).Attributes()
		if priorObject, exists := priorValues[c.Key]; exists && !plannedObject[c.Type].Equal(priorObject[c.Type]) {
			diags.AddAttributeError(path.Root(AttrPredicates).AtListIndex(i).AtName(c.Type), "Predicate records are immutable", fmt.Sprintf("predicate %q already identifies server record %s. Add the edited predicate under a new key and update only the intended transform attachments.", c.Key, c.ID))
		}
	}
	return diags
}

func validatePredicateAttachment(diags *diag.Diagnostics, at path.Path, transformType, predicateType string, negate bool, catalog Catalog) {
	spec, ok := catalog.Current("predicate", predicateType)
	if !ok {
		return
	}
	validatePredicateAttachmentSpec(diags, at, transformType, spec, negate)
}

func validatePredicateAttachmentSpec(diags *diag.Diagnostics, at path.Path, transformType string, spec *TypeSpec, negate bool) {
	compatible := false
	for _, candidate := range spec.CompatibleTransformTypes {
		if candidate == transformType {
			compatible = true
			break
		}
	}
	if !compatible {
		diags.AddAttributeError(at.AtName(attrKey), "Predicate is incompatible with transform", fmt.Sprintf("predicate type %q cannot attach to transform type %q", spec.TypeID, transformType))
	}
	if negate && !spec.SupportsNegate {
		diags.AddAttributeError(at.AtName(attrNegate), "Predicate does not support negate", fmt.Sprintf("predicate type %q does not support negated attachments", spec.TypeID))
	}
}

// Apply sends the chain replacement, secret operations included, and
// returns the state values. expectedRevision is null on create. An unchanged
// chain with no rotation sends nothing. On error the returned values are
// meaningless and the caller keeps what state already held.
func (s *Surface) Apply(ctx context.Context, client api.SMTChainAPI, kind, connectorID string, w *Write, expectedRevision types.String) (types.List, types.String, diag.Diagnostics) {
	state, diags := s.ApplyGraph(ctx, client, kind, connectorID, w, expectedRevision)
	return state.Transforms, state.Revision, diags
}

type GraphState struct {
	Transforms       types.List
	Predicates       types.List
	ManagedOverrides types.Map
	Revision         types.String
}

func (s *Surface) ApplyGraph(ctx context.Context, client api.SMTChainAPI, kind, connectorID string, w *Write, expectedRevision types.String) (GraphState, diag.Diagnostics) {
	if state, unsent := s.Unsent(w, expectedRevision); unsent {
		return state, nil
	}
	req, diags := s.WriteRequest(w, expectedRevision)
	if diags.HasError() {
		return GraphState{}, diags
	}
	read, err := client.PutSMTChain(ctx, kind, connectorID, *req)
	if err != nil {
		diags.Append(s.WriteError(err, expectedRevision, w)...)
		return GraphState{}, diags
	}
	return s.AppliedState(ctx, w, read)
}

// Unsent reports whether w needs no request, and if so the state the apply
// records: an unmanaged surface, or the unchanged graph under its revision.
func (s *Surface) Unsent(w *Write, expectedRevision types.String) (GraphState, bool) {
	if !w.Configured {
		return GraphState{Transforms: types.ListNull(s.Chain.EntryType), Predicates: w.plannedPredicates, ManagedOverrides: w.plannedOverrides, Revision: types.StringNull()}, true
	}
	if w.Unchanged {
		return GraphState{Transforms: w.planned, Predicates: w.plannedPredicates, ManagedOverrides: w.plannedOverrides, Revision: expectedRevision}, true
	}
	return GraphState{}, false
}

// WriteRequest is the chain write for w: the PUT body on a destination, the
// embedded smt_chain of a source create or update. Call it only when Unsent
// reports false. expectedRevision is null on the write that creates the chain.
func (s *Surface) WriteRequest(w *Write, expectedRevision types.String) (*api.SMTChainWrite, diag.Diagnostics) {
	var diags diag.Diagnostics
	contractVersion, digest, err := s.Chain.Catalog.WritePrecondition()
	if err != nil {
		diags.AddAttributeError(writeRoot(w), "SMT catalog release cannot be written", err.Error())
		return nil, diags
	}
	req := &api.SMTChainWrite{
		CatalogPrecondition: api.SMTCatalogPrecondition{ContractVersion: contractVersion, Digest: digest},
		ManagedOverrides:    w.ManagedOverrides,
	}
	if w.ChainConfigured {
		req.Transforms = &w.Transforms
		req.Predicates = &w.Predicates
	}
	if !expectedRevision.IsNull() && !expectedRevision.IsUnknown() {
		rev := expectedRevision.ValueString()
		req.ExpectedRevision = &rev
	}
	return req, diags
}

// AppliedState is the state after the server accepted the write for w and
// answered with read.
func (s *Surface) AppliedState(ctx context.Context, w *Write, read *api.SMTChainRead) (GraphState, diag.Diagnostics) {
	var diags diag.Diagnostics
	if !w.ChainConfigured {
		return s.overrideStateFrom(read), diags
	}
	predicateKeys, d := predicateKeysForAppliedGraph(w, read)
	diags.Append(d...)
	if diags.HasError() {
		return GraphState{}, diags
	}
	state, d := s.stateGraphFrom(ctx, read, keysOf(w.correlated), predicateKeys, w.versions, w.predicateVersions)
	diags.Append(d...)
	if w.ManagedOverrides == nil {
		state.ManagedOverrides = w.plannedOverrides
	}
	return state, diags
}

func predicateKeysForAppliedGraph(w *Write, read *api.SMTChainRead) ([]string, diag.Diagnostics) {
	var diags diag.Diagnostics
	keysByID := map[string]string{}
	idsByKey := map[string]string{}
	for _, predicate := range w.predicateCorrelated {
		if predicate.Matched {
			keysByID[predicate.ID] = predicate.Key
			idsByKey[predicate.Key] = predicate.ID
		}
	}
	for i, transform := range read.Desired.Transforms {
		if !transform.HasPredicate() || i >= len(w.Transforms) || w.Transforms[i].Predicate == nil {
			continue
		}
		var returned api.SMTPredicateAttachment
		if err := json.Unmarshal(transform.Predicate, &returned); err != nil || returned.ID == "" {
			continue
		}
		planned := w.Transforms[i].Predicate
		key := planned.RequestKey
		if key == "" {
			key = keysByID[planned.ID]
		}
		if key == "" {
			continue
		}
		if prior, exists := keysByID[returned.ID]; exists && prior != key {
			diags.AddAttributeError(path.Root(AttrPredicates), "Predicate identity cannot be represented", fmt.Sprintf("the server associated predicate id %q with both provider keys %q and %q", returned.ID, prior, key))
			continue
		}
		if prior, exists := idsByKey[key]; exists && prior != returned.ID {
			diags.AddAttributeError(path.Root(AttrPredicates), "Predicate identity cannot be represented", fmt.Sprintf("the server associated provider key %q with both predicate ids %q and %q", key, prior, returned.ID))
			continue
		}
		keysByID[returned.ID] = key
		idsByKey[key] = returned.ID
	}
	keys := make([]string, 0, len(read.Desired.Predicates))
	for i, predicate := range read.Desired.Predicates {
		key, exists := keysByID[predicate.ID]
		if !exists {
			diags.AddAttributeError(path.Root(AttrPredicates).AtListIndex(i), "Predicate identity cannot be represented", fmt.Sprintf("the server returned predicate id %q without a matching transform attachment", predicate.ID))
			continue
		}
		keys = append(keys, key)
	}
	return keys, diags
}

// WriteError maps a refused chain write to diagnostics: a stale revision is
// named as such, and every validation issue is attached to the attribute it
// locates. Issue pointers are runtime data paths into the request body, so
// /transforms/{i} is the planned entry at that position and the rest of the
// path continues into its typed config child.
func (s *Surface) WriteError(err error, expectedRevision types.String, w *Write) diag.Diagnostics {
	var diags diag.Diagnostics
	root := writeRoot(w)
	if api.IsSMTRevisionConflict(err) {
		diags.AddAttributeError(root, "Transform chain changed outside Terraform",
			fmt.Sprintf("The chain revision %q this plan was built from is no longer current. Run terraform plan again to refresh it and review the drift before applying: %s", expectedRevision.ValueString(), err))
		return diags
	}
	if api.IsSMTConnectorDeleting(err) {
		diags.AddAttributeError(root, "Connector is being deleted",
			fmt.Sprintf("The connector's deletion is pending, so its transform chain can no longer be written. Run terraform plan again once the deletion completes: %s", err))
		return diags
	}
	if api.IsSMTRevisionRequired(err) {
		diags.AddAttributeError(root, "Transform chain already exists outside Terraform",
			fmt.Sprintf("The connector already has a transform chain this resource does not manage. Import the resource to adopt it (its instances become %s entries keyed by their server ids), then apply: %s", AttrChain, err))
		return diags
	}
	if api.IsSMTAdoptionRequired(err) {
		diags.AddAttributeError(root, "Legacy transforms require explicit adoption",
			fmt.Sprintf("The destination gained authored legacy transforms after Terraform audited it. Review and adopt them through the SMT chain adoption API or UI, then refresh Terraform state: %s", err))
		return diags
	}
	env, ok := api.SMTErrorOf(err)
	if !ok || len(env.Issues) == 0 {
		diags.AddAttributeError(root, "Error writing transform chain", err.Error())
		return diags
	}
	for _, issue := range env.Issues {
		diags.AddAttributeError(s.issuePath(issue.Pointer, w), "Transform chain rejected: "+env.Message, fmt.Sprintf("%s: %s (%s)", issue.Pointer, issue.Message, issue.Code))
	}
	return diags
}

func writeRoot(w *Write) path.Path {
	if !w.ChainConfigured && w.ManagedOverrides != nil {
		return path.Root(AttrManagedOverrides)
	}
	return path.Root(AttrChain)
}

// issuePath resolves a runtime pointer to the closest attribute path. A
// secret operation pointer lands on the smt_secrets entry of that instance's
// key, since operations are regrouped per instance on the wire.
func (s *Surface) issuePath(pointer string, w *Write) path.Path {
	tokens := strings.Split(strings.TrimPrefix(pointer, "/"), "/")
	if len(tokens) > 0 && tokens[0] == "managed_overrides" {
		return path.Root(AttrManagedOverrides)
	}
	if len(tokens) >= 2 && tokens[0] == "predicates" {
		i, err := strconv.Atoi(tokens[1])
		if err != nil || i < 0 || i >= len(w.predicateCorrelated) {
			return path.Root(AttrPredicates)
		}
		entry := path.Root(AttrPredicates).AtListIndex(i)
		if len(tokens) < 3 {
			return entry
		}
		switch tokens[2] {
		case "config":
			p := entry.AtName(w.predicateCorrelated[i].Type)
			for _, token := range tokens[3:] {
				if n, err := strconv.Atoi(token); err == nil {
					p = p.AtListIndex(n)
				} else {
					p = p.AtName(UnescapeToken(token))
				}
			}
			return p
		case "secret_operations":
			return path.Root(AttrPredicateSecrets)
		default:
			return entry.AtName(tokens[2])
		}
	}
	if len(tokens) < 2 || tokens[0] != "transforms" {
		return path.Root(AttrChain)
	}
	i, err := strconv.Atoi(tokens[1])
	if err != nil || i < 0 || i >= len(w.correlated) {
		return path.Root(AttrChain)
	}
	entry := path.Root(AttrChain).AtListIndex(i)
	if len(tokens) < 3 {
		return entry
	}
	switch tokens[2] {
	case "config":
		p := entry.AtName(w.correlated[i].Type)
		for _, tok := range tokens[3:] {
			if n, err := strconv.Atoi(tok); err == nil {
				p = p.AtListIndex(n)
			} else {
				p = p.AtName(UnescapeToken(tok))
			}
		}
		return p
	case "secret_operations":
		return path.Root(AttrSecrets)
	default:
		return entry.AtName(tokens[2])
	}
}

// Refresh reads the chain under the keys state already holds. A null chain
// in state stays null: the connector is on the released surface and the
// chain must not be populated behind its back. An empty chain is the import
// marker and is populated under keys derived from the server ids.
func (s *Surface) Refresh(ctx context.Context, client api.SMTChainAPI, kind, connectorID string, prior types.List) (types.List, types.String, diag.Diagnostics) {
	state, diags := s.RefreshGraph(ctx, client, kind, connectorID, prior, types.ListNull(s.Chain.PredicateEntryType), types.MapNull(types.BoolType))
	return state.Transforms, state.Revision, diags
}

func (s *Surface) RefreshGraph(ctx context.Context, client api.SMTChainAPI, kind, connectorID string, prior, priorPredicates types.List, priorOverrides types.Map) (GraphState, diag.Diagnostics) {
	var diags diag.Diagnostics
	if prior.IsNull() && priorOverrides.IsNull() {
		return GraphState{Transforms: prior, Predicates: priorPredicates, ManagedOverrides: priorOverrides, Revision: types.StringNull()}, nil
	}
	read, err := client.GetSMTChain(ctx, kind, connectorID)
	if api.IsSMTChainAbsent(err) {
		// The connector has no chain yet: an empty managed chain with no
		// revision, so the next write creates it.
		read, err = &api.SMTChainRead{}, nil
	}
	if err != nil {
		at := path.Root(AttrChain)
		if prior.IsNull() {
			at = path.Root(AttrManagedOverrides)
		}
		diags.AddAttributeError(at, "Error reading transform chain", err.Error())
		return GraphState{Transforms: prior, Predicates: priorPredicates, ManagedOverrides: priorOverrides, Revision: types.StringNull()}, diags
	}
	if prior.IsNull() {
		return s.overrideStateFrom(read), diags
	}
	priorInstances := PriorFromList(prior)
	versions := map[string]int64{}
	for _, p := range priorInstances {
		versions[p.Key] = p.SecretVersion
	}
	predicatePrior := PriorFromList(priorPredicates)
	predicateVersions := map[string]int64{}
	for _, p := range predicatePrior {
		predicateVersions[p.Key] = p.SecretVersion
	}
	state, d := s.stateGraphFrom(ctx, read, keysForReads(priorInstances, read.Desired.Transforms), keysForPredicateReads(predicatePrior, read.Desired.Predicates), versions, predicateVersions)
	diags.Append(d...)
	if priorOverrides.IsNull() {
		state.ManagedOverrides = priorOverrides
	}
	return state, diags
}

// stateFrom turns the desired snapshot into state. An instance the schema cannot represent, an opaque one or one carrying a selector, is refused rather than silently dropped: writing the chain back without it would delete it.
func (s *Surface) stateFrom(ctx context.Context, read *api.SMTChainRead, keys []string, versions map[string]int64) (types.List, types.String, diag.Diagnostics) {
	state, diags := s.stateGraphFrom(ctx, read, keys, nil, versions, nil)
	return state.Transforms, state.Revision, diags
}

func (s *Surface) stateGraphFrom(ctx context.Context, read *api.SMTChainRead, keys, predicateKeys []string, versions, predicateVersions map[string]int64) (GraphState, diag.Diagnostics) {
	var diags diag.Diagnostics
	for i, inst := range read.Desired.Transforms {
		at := path.Root(AttrChain).AtListIndex(i)
		if inst.Kind != "" && inst.Kind != "typed" && inst.Kind != "opaque" {
			diags.AddAttributeError(at, "Transform kind cannot be represented",
				"This provider version cannot safely represent the returned transform kind; use a compatible provider before managing this chain.")
			continue
		}
		if inst.Kind == "opaque" {
			diags.AddAttributeError(at, "Transform instance cannot be represented",
				fmt.Sprintf("instance %s (%s) is opaque: %s. This provider version cannot manage it; remove it from the chain elsewhere before managing the chain here.", inst.ID, inst.Alias, inst.Reason))
		}
		if inst.HasSelector() {
			diags.AddAttributeError(at, "Transform selector cannot be represented",
				fmt.Sprintf("instance %s (%s) carries a selector. This provider version has no selector attribute and would drop the rule on the next write; remove the selector elsewhere before managing the chain here.", inst.ID, inst.Alias))
		}
	}
	if diags.HasError() {
		transformType := attr.Type(types.DynamicType)
		predicateType := attr.Type(types.DynamicType)
		if s.Chain != nil {
			transformType = s.Chain.EntryType
			predicateType = s.Chain.PredicateEntryType
		}
		return GraphState{
			Transforms: types.ListNull(transformType),
			Predicates: types.ListNull(predicateType),
			Revision:   types.StringNull(),
		}, diags
	}
	if len(predicateKeys) != len(read.Desired.Predicates) {
		if len(read.Desired.Predicates) > 0 {
			diags.AddAttributeError(path.Root(AttrPredicates), "Predicate identity cannot be represented", fmt.Sprintf("%d provider keys for %d returned predicates", len(predicateKeys), len(read.Desired.Predicates)))
		}
	}
	predicateKeyByID := map[string]string{}
	predicateTypeByID := map[string]string{}
	predicateSpecsByID := map[string]*TypeSpec{}
	for i, predicate := range read.Desired.Predicates {
		at := path.Root(AttrPredicates).AtListIndex(i)
		if predicate.ID == "" {
			diags.AddAttributeError(at.AtName(attrID), "Predicate identity cannot be represented", "the server returned a predicate without an id")
			continue
		}
		if _, exists := predicateTypeByID[predicate.ID]; exists {
			diags.AddAttributeError(at.AtName(attrID), "Predicate identity cannot be represented", fmt.Sprintf("the server returned duplicate predicate id %q", predicate.ID))
			continue
		}
		spec, pinDiags := s.Chain.pinnedChild("predicate", CatalogPin{Role: "predicate", TypeID: predicate.Type, SchemaVersion: predicate.SchemaVersion, CodecVersion: predicate.CodecVersion})
		diags.Append(pinDiags...)
		if pinDiags.HasError() {
			continue
		}
		predicateTypeByID[predicate.ID] = predicate.Type
		predicateSpecsByID[predicate.ID] = spec
		if i < len(predicateKeys) {
			predicateKeyByID[predicate.ID] = predicateKeys[i]
		}
	}
	for i, inst := range read.Desired.Transforms {
		at := path.Root(AttrChain).AtListIndex(i)
		if inst.HasPredicate() {
			var attachment api.SMTPredicateAttachment
			if err := json.Unmarshal(inst.Predicate, &attachment); err != nil || attachment.ID == "" {
				diags.AddAttributeError(at.AtName(attrPredicate), "Transform predicate cannot be represented", fmt.Sprintf("instance %s has an invalid predicate attachment", inst.ID))
			} else if _, ok := predicateKeyByID[attachment.ID]; !ok {
				diags.AddAttributeError(at.AtName(attrPredicate), "Transform predicate cannot be represented", fmt.Sprintf("instance %s references predicate id %q outside the returned predicate pool", inst.ID, attachment.ID))
			} else {
				validatePredicateAttachmentSpec(&diags, at.AtName(attrPredicate), inst.Type, predicateSpecsByID[attachment.ID], attachment.Negate)
			}
		}
		_, pinDiags := s.Chain.pinnedChild("transform", CatalogPin{Role: "transform", TypeID: inst.Type, SchemaVersion: inst.SchemaVersion, CodecVersion: inst.CodecVersion})
		diags.Append(pinDiags...)
	}
	if diags.HasError() {
		return GraphState{}, diags
	}
	predicates, d := s.Chain.PredicatesValue(ctx, predicateKeys, read.Desired.Predicates, predicateVersions)
	diags.Append(d...)
	list, d := s.Chain.chainValue(ctx, keys, read.Desired.Transforms, versions, predicateKeyByID)
	diags.Append(d...)
	revision := types.StringNull()
	if read.Desired.ChainRevision != "" {
		revision = types.StringValue(read.Desired.ChainRevision)
	}
	managedOverrides := managedOverridesValue(read.Desired.ManagedOverrides)
	return GraphState{Transforms: list, Predicates: predicates, ManagedOverrides: managedOverrides, Revision: revision}, diags
}

func (s *Surface) overrideStateFrom(read *api.SMTChainRead) GraphState {
	revision := types.StringNull()
	if read.Desired.ChainRevision != "" {
		revision = types.StringValue(read.Desired.ChainRevision)
	}
	return GraphState{
		Transforms:       types.ListNull(s.Chain.EntryType),
		Predicates:       types.ListNull(s.Chain.PredicateEntryType),
		ManagedOverrides: managedOverridesValue(read.Desired.ManagedOverrides),
		Revision:         revision,
	}
}

func managedOverridesValue(values map[string]bool) types.Map {
	overrides := make(map[string]attr.Value, len(values))
	for key, value := range values {
		overrides[key] = types.BoolValue(value)
	}
	return types.MapValueMust(types.BoolType, overrides)
}

// ImportMarker is the empty, non-null chain ImportState writes so the Read
// that follows adopts the server instances under keys derived from their ids.
func (s *Surface) ImportMarker() types.List {
	return types.ListValueMust(s.Chain.EntryType, nil)
}

func (s *Surface) PredicateImportMarker() types.List {
	return types.ListValueMust(s.Chain.PredicateEntryType, nil)
}

// secretInputs reads the write-only entries from the configuration, the
// only place the values exist: plan and state hold them nulled.
func secretInputs(ctx context.Context, cfg tfsdk.Config) ([]SecretEntry, diag.Diagnostics) {
	return secretInputsAt(ctx, cfg, AttrSecrets)
}

func secretInputsAt(ctx context.Context, cfg tfsdk.Config, attribute string) ([]SecretEntry, diag.Diagnostics) {
	var configured types.List
	diags := cfg.GetAttribute(ctx, path.Root(attribute), &configured)
	return SecretEntriesFromList(configured), diags
}

func requireNewPredicateSecrets(diags *diag.Diagnostics, at path.Path, child *ConfigChild, key string, ops []SecretOp) {
	if child == nil {
		return
	}
	for _, required := range child.RequiredSecretTemplates {
		present := false
		for _, op := range ops {
			if op.Key == key && !op.Clear && (op.Pointer == required || strings.Contains(required, "*") && pointerMatchesTemplate(op.Pointer, required)) {
				present = true
				break
			}
		}
		if !present {
			diags.AddAttributeError(at, "New predicate needs its own secret", fmt.Sprintf("predicate %q is a new identity and must replace required secret %s through %s; it cannot inherit configured presence from another predicate", key, required, AttrPredicateSecrets))
		}
	}
}

func pointerMatchesTemplate(pointer, template string) bool {
	p, t := strings.Split(strings.TrimPrefix(pointer, "/"), "/"), strings.Split(strings.TrimPrefix(template, "/"), "/")
	if len(p) != len(t) {
		return false
	}
	for i := range p {
		if t[i] != "*" && p[i] != t[i] {
			return false
		}
	}
	return true
}

// secretVersions is the last rotation version applied per key, as the chain
// entries in prior state record it. A new key starts from zero because it is
// a new instance.
func secretVersions(correlated []Correlated) map[string]int64 {
	out := make(map[string]int64, len(correlated))
	for _, c := range correlated {
		if c.Matched {
			out[c.Key] = c.SecretVersion
		}
	}
	return out
}

// keysForReads pairs server instances with the keys state already holds,
// by id. An instance state has never seen gets a key derived from its id.
func keysForReads(prior []PriorInstance, reads []api.SMTInstanceRead) []string {
	byID := map[string]string{}
	for _, p := range prior {
		byID[p.ID] = p.Key
	}
	keys := make([]string, 0, len(reads))
	for _, rd := range reads {
		if k, ok := byID[rd.ID]; ok {
			keys = append(keys, k)
			continue
		}
		keys = append(keys, rd.ID)
	}
	return keys
}

func keysForPredicateReads(prior []PriorInstance, reads []api.SMTPredicateRead) []string {
	byID := map[string]string{}
	for _, p := range prior {
		byID[p.ID] = p.Key
	}
	keys := make([]string, 0, len(reads))
	for _, read := range reads {
		if key, ok := byID[read.ID]; ok {
			keys = append(keys, key)
		} else {
			keys = append(keys, read.ID)
		}
	}
	return keys
}

func keysOf(correlated []Correlated) []string {
	keys := make([]string, 0, len(correlated))
	for _, c := range correlated {
		keys = append(keys, c.Key)
	}
	return keys
}

func chainPins(correlated []Correlated, role string, catalog Catalog) map[string]CatalogPin {
	out := make(map[string]CatalogPin, len(correlated))
	for _, c := range correlated {
		if c.Matched {
			out[c.Key] = CatalogPin{Role: role, TypeID: c.Type, SchemaVersion: c.SchemaVersion, CodecVersion: c.CodecVersion}
			continue
		}
		if spec, ok := catalog.Current(role, c.Type); ok {
			out[c.Key] = spec.Pin()
		}
	}
	return out
}

func valuesByKey(list types.List) map[string]map[string]attr.Value {
	out := map[string]map[string]attr.Value{}
	if list.IsNull() || list.IsUnknown() {
		return out
	}
	for _, value := range list.Elements() {
		object := value.(types.Object).Attributes()
		out[str(object[attrKey])] = object
	}
	return out
}
