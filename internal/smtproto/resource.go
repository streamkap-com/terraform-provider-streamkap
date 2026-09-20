package smtproto

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	"github.com/streamkap-com/terraform-provider-streamkap/internal/resource/source"
)

// Backend is the minimal server surface the prototype needs. The public HTTP
// shape is not frozen, so this is an interface the tests fake in memory; it
// carries the DTOs the wire projection produces and nothing else.
type Backend interface {
	Create(ctx context.Context, chain []InstanceWrite) (string, []InstanceRead, error)
	Read(ctx context.Context, id string) ([]InstanceRead, bool, error)
	Update(ctx context.Context, id string, chain []InstanceWrite) ([]InstanceRead, error)
	Delete(ctx context.Context, id string) error
	ApplySecrets(ctx context.Context, id string, ops []ResolvedSecretOp) error
}

// ResolvedSecretOp is a SecretOp addressed by the server instance id its
// provider key correlated to.
type ResolvedSecretOp struct {
	InstanceID string
	Pointer    string
	Clear      bool
	Value      string
}

// Resource is the released PostgreSQL source schema plus the nested SMT
// surface. The flat attributes pass through plan and state untouched: the
// released marshaling owns their wire path, this prototype only has to prove
// that the two surfaces coexist.
type Resource struct {
	backend Backend
	chain   *ChainSchema
	schema  schema.Schema
	flatSMT []string
}

var (
	_ resource.Resource                   = &Resource{}
	_ resource.ResourceWithImportState    = &Resource{}
	_ resource.ResourceWithModifyPlan     = &Resource{}
	_ resource.ResourceWithValidateConfig = &Resource{}
)

// NewResource builds the prototype over a catalog. The released schema is
// taken from the registered PostgreSQL resource, not copied.
func NewResource(backend Backend, cat Catalog) (*Resource, error) {
	chain, err := BuildChain(cat)
	if err != nil {
		return nil, err
	}
	released := &resource.SchemaResponse{}
	source.NewPostgreSQLResource().Schema(context.Background(), resource.SchemaRequest{}, released)
	if released.Diagnostics.HasError() {
		return nil, fmt.Errorf("NewResource: released schema: %v", released.Diagnostics)
	}
	s := released.Schema
	s.Attributes[AttrChain] = chain.Attribute
	s.Attributes[AttrSecrets] = SecretsAttribute()
	return &Resource{
		backend: backend,
		chain:   chain,
		schema:  s,
		flatSMT: FlatSMTAttributes((&source.PostgreSQLConfig{}).GetFieldMappings()),
	}, nil
}

// Schema returns the combined schema.
func (r *Resource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = r.schema
}

// Metadata names the prototype resource.
func (r *Resource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_smtproto_source_postgresql"
}

// ValidateConfig rejects a configuration that drives the same backend surface
// from both the released flat attributes and the chain, and a secret entry
// whose key is not in the chain. Both run before any plan or request.
func (r *Resource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var chain types.List
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root(AttrChain), &chain)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !chain.IsNull() {
		for _, name := range r.flatSMT {
			var v attr.Value
			resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root(name), &v)...)
			if v != nil && !v.IsNull() {
				resp.Diagnostics.AddAttributeError(path.Root(name),
					"Flat and nested transform attributes cannot be mixed",
					fmt.Sprintf("%s addresses the transform chain through the released flat attributes while %s is also configured. Move it into a %s entry or remove %s.", name, AttrChain, AttrChain, AttrChain))
			}
		}
	}
	planned := PlannedFromList(chain)
	_, d := Correlate(nil, planned)
	resp.Diagnostics.Append(d...)

	var secrets types.List
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root(AttrSecrets), &secrets)...)
	if secrets.IsNull() || secrets.IsUnknown() || chain.IsUnknown() {
		return
	}
	keys := map[string]bool{}
	for _, p := range planned {
		keys[p.Key] = true
	}
	for i, e := range SecretEntriesFromList(secrets) {
		if e.Key != "" && !keys[e.Key] {
			resp.Diagnostics.AddAttributeError(path.Root(AttrSecrets).AtListIndex(i).AtName(attrKey),
				"Secret entry for unknown instance", fmt.Sprintf("instance key %q is not in %s", e.Key, AttrChain))
		}
	}
}

// ModifyPlan correlates the planned chain to prior state by key, pins the
// matched server identities into the plan, and rejects a type change under
// an existing key and any invalid rotation before a request can be sent.
func (r *Resource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.Plan.Raw.IsNull() {
		return
	}
	var planned, prior types.List
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root(AttrChain), &planned)...)
	if !req.State.Raw.IsNull() {
		resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root(AttrChain), &prior)...)
	}
	if resp.Diagnostics.HasError() || planned.IsUnknown() {
		return
	}
	correlated, d := Correlate(PriorFromList(prior), PlannedFromList(planned))
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}
	for i, c := range correlated {
		if !c.Matched {
			continue
		}
		at := path.Root(AttrChain).AtListIndex(i)
		resp.Diagnostics.Append(resp.Plan.SetAttribute(ctx, at.AtName(attrID), types.StringValue(c.ID))...)
		resp.Diagnostics.Append(resp.Plan.SetAttribute(ctx, at.AtName(attrAlias), types.StringValue(c.Alias))...)
		resp.Diagnostics.Append(resp.Plan.SetAttribute(ctx, at.AtName(attrSchemaVersion), types.Int64Value(c.SchemaVersion))...)
	}

	entries, priorVersions, d := r.secretInputs(ctx, req.Config, &req.State)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}
	ops, d := PlanSecretOps(priorVersions, entries, chainTypes(correlated), r.chain.Catalog)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
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
		addr, _ := ResolvePointer(r.chain.Catalog[typeID], op.Pointer)
		if !RowExists(r.chain.Catalog[typeID], addr, wire.(map[string]any)) {
			resp.Diagnostics.AddAttributeError(path.Root(AttrSecrets), "Secret pointer names a missing row",
				fmt.Sprintf("%s does not exist in the planned config of %q", op.Pointer, op.Key))
		}
	}
}

// Create sends the chain, then the secret operations, then records state.
func (r *Resource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var planned types.List
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root(AttrChain), &planned)...)
	if resp.Diagnostics.HasError() {
		return
	}
	correlated, d := Correlate(nil, PlannedFromList(planned))
	resp.Diagnostics.Append(d...)
	writes, d := r.chain.Project(planned, correlated)
	resp.Diagnostics.Append(d...)
	entries, _, d := r.secretInputs(ctx, req.Config, nil)
	resp.Diagnostics.Append(d...)
	ops, d := PlanSecretOps(nil, entries, chainTypes(correlated), r.chain.Catalog)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, reads, err := r.backend.Create(ctx, writes)
	if err != nil {
		resp.Diagnostics.AddError("Error creating connector", err.Error())
		return
	}
	if err := r.applySecrets(ctx, id, ops, correlated, reads); err != nil {
		resp.Diagnostics.AddError("Error applying secrets", err.Error())
		return
	}

	resp.State.Raw = knownOrNull(req.Plan.Raw)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), types.StringValue(id))...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("connector"), types.StringValue("postgresql"))...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("connector_status"), types.StringValue("Active"))...)
	resp.Diagnostics.Append(r.writeChain(ctx, &resp.State, planned, keysOf(correlated), reads)...)
}

// Read refreshes the chain under the keys state already holds. A null chain
// in state stays null: the connector is on the released surface and the
// chain must not be populated behind its back.
func (r *Resource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var id types.String
	var prior types.List
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("id"), &id)...)
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root(AttrChain), &prior)...)
	if resp.Diagnostics.HasError() {
		return
	}
	reads, found, err := r.backend.Read(ctx, id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading connector", err.Error())
		return
	}
	if !found {
		resp.State.RemoveResource(ctx)
		return
	}
	resp.State.Raw = req.State.Raw
	if prior.IsNull() {
		return
	}
	keys := keysForReads(PriorFromList(prior), reads)
	resp.Diagnostics.Append(r.writeChain(ctx, &resp.State, prior, keys, reads)...)
}

// Update correlates by key so a reorder or rename keeps every server id.
func (r *Resource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var planned, prior types.List
	var id types.String
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root(AttrChain), &planned)...)
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root(AttrChain), &prior)...)
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("id"), &id)...)
	if resp.Diagnostics.HasError() {
		return
	}
	correlated, d := Correlate(PriorFromList(prior), PlannedFromList(planned))
	resp.Diagnostics.Append(d...)
	writes, d := r.chain.Project(planned, correlated)
	resp.Diagnostics.Append(d...)
	entries, priorVersions, d := r.secretInputs(ctx, req.Config, &req.State)
	resp.Diagnostics.Append(d...)
	ops, d := PlanSecretOps(priorVersions, entries, chainTypes(correlated), r.chain.Catalog)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}

	var reads []InstanceRead
	var err error
	if planned.IsNull() {
		reads, _, err = r.backend.Read(ctx, id.ValueString())
	} else {
		reads, err = r.backend.Update(ctx, id.ValueString(), writes)
	}
	if err != nil {
		resp.Diagnostics.AddError("Error updating connector", err.Error())
		return
	}
	if err := r.applySecrets(ctx, id.ValueString(), ops, correlated, reads); err != nil {
		resp.Diagnostics.AddError("Error applying secrets", err.Error())
		return
	}

	resp.State.Raw = knownOrNull(req.Plan.Raw)
	resp.Diagnostics.Append(r.writeChain(ctx, &resp.State, planned, keysOf(correlated), reads)...)
}

// Delete removes the connector.
func (r *Resource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var id types.String
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("id"), &id)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.backend.Delete(ctx, id.ValueString()); err != nil {
		resp.Diagnostics.AddError("Error deleting connector", err.Error())
	}
}

// ImportState adopts the chain: an empty (not null) chain marks the surface
// as managed so the Read that follows populates it with keys derived from
// the server ids.
func (r *Resource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root(AttrChain), types.ListValueMust(r.chain.EntryType, nil))...)
}

// writeChain records the server's chain in state. The released flat
// transform attributes are left as planned: most of them carry a client-side
// Default, so they are present in every plan whether or not the user wrote
// them, and nulling them here would make the apply inconsistent.
func (r *Resource) writeChain(ctx context.Context, state *tfsdk.State, planned types.List, keys []string, reads []InstanceRead) diag.Diagnostics {
	var diags diag.Diagnostics
	if planned.IsNull() {
		return nil
	}
	list, d := r.chain.ChainValue(ctx, keys, reads)
	diags.Append(d...)
	diags.Append(state.SetAttribute(ctx, path.Root(AttrChain), list)...)
	return diags
}

// secretInputs reads the write-only entries from the configuration and the
// rotation versions from prior state, nil on create. Values are never read
// from plan or state, where the framework has already nulled them.
func (r *Resource) secretInputs(ctx context.Context, config tfsdk.Config, state *tfsdk.State) ([]SecretEntry, map[string]int64, diag.Diagnostics) {
	var diags diag.Diagnostics
	var configured, prior types.List
	diags.Append(config.GetAttribute(ctx, path.Root(AttrSecrets), &configured)...)
	if state != nil && !state.Raw.IsNull() {
		diags.Append(state.GetAttribute(ctx, path.Root(AttrSecrets), &prior)...)
	}
	versions := map[string]int64{}
	for _, e := range SecretEntriesFromList(prior) {
		versions[e.Key] = e.Version
	}
	return SecretEntriesFromList(configured), versions, diags
}

// applySecrets resolves each operation's provider key to the server id the
// write returned for it and sends the batch.
func (r *Resource) applySecrets(ctx context.Context, connectorID string, ops []SecretOp, correlated []Correlated, reads []InstanceRead) error {
	if len(ops) == 0 {
		return nil
	}
	if len(reads) != len(correlated) {
		return fmt.Errorf("applySecrets: %d instances returned for %d planned", len(reads), len(correlated))
	}
	ids := map[string]string{}
	for i, c := range correlated {
		ids[c.Key] = reads[i].ID
	}
	resolved := make([]ResolvedSecretOp, 0, len(ops))
	for _, op := range ops {
		resolved = append(resolved, ResolvedSecretOp{InstanceID: ids[op.Key], Pointer: op.Pointer, Clear: op.Clear, Value: op.Value})
	}
	return r.backend.ApplySecrets(ctx, connectorID, resolved)
}

// keysForReads pairs server instances with the keys state already holds,
// by id. An instance state has never seen gets a key derived from its id.
func keysForReads(prior []PriorInstance, reads []InstanceRead) []string {
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

// knownOrNull turns every unknown in a planned value into null, so the flat
// attributes the prototype does not manage still leave apply fully known.
func knownOrNull(v tftypes.Value) tftypes.Value {
	out, err := tftypes.Transform(v, func(_ *tftypes.AttributePath, v tftypes.Value) (tftypes.Value, error) {
		if !v.IsKnown() {
			return tftypes.NewValue(v.Type(), nil), nil
		}
		return v, nil
	})
	if err != nil {
		return v
	}
	return out
}
