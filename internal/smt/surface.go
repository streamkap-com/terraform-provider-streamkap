package smt

import (
	"context"
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
	planned := PlannedFromList(chain)
	_, d := Correlate(nil, planned)
	diags.Append(d...)

	var secrets types.List
	diags.Append(cfg.GetAttribute(ctx, path.Root(AttrSecrets), &secrets)...)
	if secrets.IsNull() || secrets.IsUnknown() || chain.IsUnknown() {
		return diags
	}
	keys := map[string]bool{}
	for _, p := range planned {
		keys[p.Key] = true
	}
	for i, e := range SecretEntriesFromList(secrets) {
		if e.Key != "" && !keys[e.Key] {
			diags.AddAttributeError(path.Root(AttrSecrets).AtListIndex(i).AtName(attrKey),
				"Secret entry for unknown instance", fmt.Sprintf("instance key %q is not in %s", e.Key, AttrChain))
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
	diags.Append(cfg.GetAttribute(ctx, path.Root(AttrChain), &planned)...)
	if !state.Raw.IsNull() {
		diags.Append(state.GetAttribute(ctx, path.Root(AttrChain), &prior)...)
		diags.Append(state.GetAttribute(ctx, path.Root(AttrRevision), &priorRevision)...)
	} else {
		priorRevision = types.StringNull()
	}
	if diags.HasError() || planned.IsUnknown() {
		return diags
	}
	if planned.IsNull() {
		// The chain is not managed here. The revision follows the chain: it
		// is only ever known for a managed chain.
		diags.Append(plan.SetAttribute(ctx, path.Root(AttrRevision), types.StringNull())...)
		return diags
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
	}

	entries, d := secretInputs(ctx, cfg)
	diags.Append(d...)
	if diags.HasError() {
		return diags
	}
	ops, next, d := PlanSecretOps(secretVersions(correlated), entries, chainTypes(correlated), s.Chain.Catalog)
	diags.Append(d...)
	if diags.HasError() {
		return diags
	}
	// The rotation version an apply will record is known now: the prior one
	// unless this plan rotates the key, and 0 for a key never rotated.
	for i, c := range correlated {
		at := path.Root(AttrChain).AtListIndex(i)
		diags.Append(plan.SetAttribute(ctx, at.AtName(attrSecretVersion), types.Int64Value(next[c.Key]))...)
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
		addr, _ := ResolvePointer(s.Chain.Catalog[typeID], op.Pointer)
		if !RowExists(s.Chain.Catalog[typeID], addr, wire.(map[string]any)) {
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
	if !diags.HasError() && len(ops) == 0 && !prior.IsNull() && pinned.Equal(prior) {
		diags.Append(plan.SetAttribute(ctx, path.Root(AttrRevision), priorRevision)...)
	}
	return diags
}

// Write is what one apply sends for the chain.
type Write struct {
	// Configured is false when the chain is null in the plan: nothing is
	// written and the chain stays unmanaged.
	Configured bool
	// Unchanged is true when the planned chain equals the chain in state and
	// no secret rotates, so nothing is sent and the revision stays.
	Unchanged  bool
	Transforms []api.SMTInstanceWrite
	planned    types.List
	correlated []Correlated
	versions   map[string]int64
}

// PlanWrite projects the planned chain with its secret operations. prior is
// the chain in state, null on create.
func (s *Surface) PlanWrite(ctx context.Context, plan tfsdk.Plan, prior types.List, cfg tfsdk.Config) (*Write, diag.Diagnostics) {
	var diags diag.Diagnostics
	var planned types.List
	diags.Append(plan.GetAttribute(ctx, path.Root(AttrChain), &planned)...)
	if diags.HasError() {
		return nil, diags
	}
	if planned.IsNull() {
		return &Write{}, nil
	}
	w := &Write{Configured: true, planned: planned}
	var d diag.Diagnostics
	w.correlated, d = Correlate(PriorFromList(prior), PlannedFromList(planned))
	diags.Append(d...)
	entries, d := secretInputs(ctx, cfg)
	diags.Append(d...)
	if diags.HasError() {
		return nil, diags
	}
	ops, versions, d := PlanSecretOps(secretVersions(w.correlated), entries, chainTypes(w.correlated), s.Chain.Catalog)
	diags.Append(d...)
	w.versions = versions
	w.Transforms, d = s.Chain.Project(planned, w.correlated, ops)
	diags.Append(d...)
	if diags.HasError() {
		return nil, diags
	}
	w.Unchanged = !prior.IsNull() && planned.Equal(prior) && len(ops) == 0
	return w, nil
}

// Apply sends the chain replacement, secret operations included, and
// returns the state values. expectedRevision is null on create. An unchanged
// chain with no rotation sends nothing. On error the returned values are
// meaningless and the caller keeps what state already held.
func (s *Surface) Apply(ctx context.Context, client api.SMTChainAPI, kind, connectorID string, w *Write, expectedRevision types.String) (types.List, types.String, diag.Diagnostics) {
	var diags diag.Diagnostics
	if !w.Configured {
		return types.ListNull(s.Chain.EntryType), types.StringNull(), nil
	}
	if w.Unchanged {
		return w.planned, expectedRevision, nil
	}
	req := api.SMTChainWrite{Transforms: w.Transforms}
	if !expectedRevision.IsNull() && !expectedRevision.IsUnknown() {
		rev := expectedRevision.ValueString()
		req.ExpectedRevision = &rev
	}
	read, err := client.PutSMTChain(ctx, kind, connectorID, req)
	if err != nil {
		diags.Append(s.writeError(err, expectedRevision, w)...)
		return types.List{}, types.String{}, diags
	}
	list, revision, d := s.stateFrom(ctx, read, keysOf(w.correlated), w.versions)
	diags.Append(d...)
	return list, revision, diags
}

// writeError maps a refused chain write to diagnostics: a stale revision is
// named as such, and every validation issue is attached to the attribute it
// locates. Issue pointers are runtime data paths into the request body, so
// /transforms/{i} is the planned entry at that position and the rest of the
// path continues into its typed config child.
func (s *Surface) writeError(err error, expectedRevision types.String, w *Write) diag.Diagnostics {
	var diags diag.Diagnostics
	if api.IsSMTRevisionConflict(err) {
		diags.AddAttributeError(path.Root(AttrChain), "Transform chain changed outside Terraform",
			fmt.Sprintf("The chain revision %q this plan was built from is no longer current. Run terraform plan again to refresh it and review the drift before applying: %s", expectedRevision.ValueString(), err))
		return diags
	}
	if api.IsSMTRevisionRequired(err) {
		diags.AddAttributeError(path.Root(AttrChain), "Transform chain already exists outside Terraform",
			fmt.Sprintf("The connector already has a transform chain this resource does not manage. Import the resource to adopt it (its instances become %s entries keyed by their server ids), then apply: %s", AttrChain, err))
		return diags
	}
	env, ok := api.SMTErrorOf(err)
	if !ok || len(env.Issues) == 0 {
		diags.AddAttributeError(path.Root(AttrChain), "Error writing transform chain", err.Error())
		return diags
	}
	for _, issue := range env.Issues {
		diags.AddAttributeError(s.issuePath(issue.Pointer, w), "Transform chain rejected: "+env.Message, fmt.Sprintf("%s: %s (%s)", issue.Pointer, issue.Message, issue.Code))
	}
	return diags
}

// issuePath resolves a runtime pointer to the closest attribute path. A
// secret operation pointer lands on the smt_secrets entry of that instance's
// key, since operations are regrouped per instance on the wire.
func (s *Surface) issuePath(pointer string, w *Write) path.Path {
	tokens := strings.Split(strings.TrimPrefix(pointer, "/"), "/")
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
	var diags diag.Diagnostics
	if prior.IsNull() {
		return prior, types.StringNull(), nil
	}
	read, err := client.GetSMTChain(ctx, kind, connectorID)
	if api.IsSMTChainAbsent(err) {
		// The connector has no chain yet: an empty managed chain with no
		// revision, so the next write creates it.
		read, err = &api.SMTChainRead{}, nil
	}
	if err != nil {
		diags.AddAttributeError(path.Root(AttrChain), "Error reading transform chain", err.Error())
		return prior, types.StringNull(), diags
	}
	priorInstances := PriorFromList(prior)
	versions := map[string]int64{}
	for _, p := range priorInstances {
		versions[p.Key] = p.SecretVersion
	}
	list, revision, d := s.stateFrom(ctx, read, keysForReads(priorInstances, read.Desired.Transforms), versions)
	diags.Append(d...)
	return list, revision, diags
}

// stateFrom turns the desired snapshot into state. An instance the schema
// cannot represent, an opaque one or one carrying a predicate, is refused
// rather than silently dropped: writing the chain back without it would
// delete it.
func (s *Surface) stateFrom(ctx context.Context, read *api.SMTChainRead, keys []string, versions map[string]int64) (types.List, types.String, diag.Diagnostics) {
	var diags diag.Diagnostics
	for i, inst := range read.Desired.Transforms {
		at := path.Root(AttrChain).AtListIndex(i)
		if inst.Kind == "opaque" {
			diags.AddAttributeError(at, "Transform instance cannot be represented",
				fmt.Sprintf("instance %s (%s) is opaque: %s. This provider version cannot manage it; remove it from the chain elsewhere before managing the chain here.", inst.ID, inst.Alias, inst.Reason))
		}
		if inst.HasPredicate() {
			diags.AddAttributeError(at, "Transform predicate cannot be represented",
				fmt.Sprintf("instance %s (%s) carries a predicate. This provider version has no predicate attribute and would drop it on the next write; remove the predicate elsewhere before managing the chain here.", inst.ID, inst.Alias))
		}
	}
	if diags.HasError() {
		return types.List{}, types.String{}, diags
	}
	list, d := s.Chain.ChainValue(ctx, keys, read.Desired.Transforms, versions)
	diags.Append(d...)
	revision := types.StringNull()
	if read.Desired.ChainRevision != "" {
		revision = types.StringValue(read.Desired.ChainRevision)
	}
	return list, revision, diags
}

// ImportMarker is the empty, non-null chain ImportState writes so the Read
// that follows adopts the server instances under keys derived from their ids.
func (s *Surface) ImportMarker() types.List {
	return types.ListValueMust(s.Chain.EntryType, nil)
}

// secretInputs reads the write-only entries from the configuration, the
// only place the values exist: plan and state hold them nulled.
func secretInputs(ctx context.Context, cfg tfsdk.Config) ([]SecretEntry, diag.Diagnostics) {
	var configured types.List
	diags := cfg.GetAttribute(ctx, path.Root(AttrSecrets), &configured)
	return SecretEntriesFromList(configured), diags
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

func keysOf(correlated []Correlated) []string {
	keys := make([]string, 0, len(correlated))
	for _, c := range correlated {
		keys = append(keys, c.Key)
	}
	return keys
}

func chainTypes(correlated []Correlated) map[string]string {
	out := make(map[string]string, len(correlated))
	for _, c := range correlated {
		out[c.Key] = c.Type
	}
	return out
}
