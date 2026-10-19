// Package connector provides a generic base resource implementation for Streamkap
// source and destination connectors in the Terraform provider.
//
// # Architecture
//
// This package implements the base layer of the three-layer architecture:
//
//  1. Generated Schemas (internal/generated/) - Auto-generated from backend config
//  2. Thin Wrappers (internal/resource/source/, destination/) - ~55 LOC each
//  3. Shared Base Resource (this package) - Generic CRUD with reflection
//
// The BaseConnectorResource implements the Terraform Plugin Framework interfaces
// and uses reflection-based marshaling to convert between Terraform state and
// API payloads, eliminating the need for per-connector CRUD implementations.
//
// # Key Components
//
// ConnectorConfig interface: Each connector wrapper must implement this interface
// to provide its schema, field mappings, connector type, and model factory.
//
// BaseConnectorResource: The generic resource implementation that handles Create,
// Read, Update, Delete, and ImportState operations for all connectors.
//
// # Reflection-Based Marshaling
//
// The modelToConfigMap and configMapToModel functions use Go reflection to:
//   - Convert Terraform model structs to API request maps (for Create/Update)
//   - Convert API response maps back to Terraform model structs (for Read)
//
// This approach eliminates ~400 LOC of manual type conversion per connector
// while maintaining type safety through compile-time interface checks.
//
// # Usage
//
// Connector wrappers create a BaseConnectorResource via NewBaseConnectorResource:
//
//	func NewSourcePostgreSQL() resource.Resource {
//	    return connector.NewBaseConnectorResource(&Config{})
//	}
//
// Where Config implements the ConnectorConfig interface with the connector-specific
// schema and field mappings.
package connector

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/defaults"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/streamkap-com/terraform-provider-streamkap/internal/api"
	"github.com/streamkap-com/terraform-provider-streamkap/internal/generated"
	"github.com/streamkap-com/terraform-provider-streamkap/internal/helper"
	"github.com/streamkap-com/terraform-provider-streamkap/internal/resource/shared"
	"github.com/streamkap-com/terraform-provider-streamkap/internal/smt"
)

// ConnectorType represents the type of connector (source or destination).
type ConnectorType string

const (
	// ConnectorTypeSource indicates a source connector.
	ConnectorTypeSource ConnectorType = "source"
	// ConnectorTypeDestination indicates a destination connector.
	ConnectorTypeDestination ConnectorType = "destination"
)

// ConnectorConfig is the interface that each connector must implement to provide
// its schema, field mappings, and metadata to the base resource.
type ConnectorConfig interface {
	// GetSchema returns the Terraform schema for this connector.
	GetSchema() schema.Schema

	// GetFieldMappings returns a map from Terraform attribute names to API field names.
	// Example: "database_hostname" -> "database.hostname.user.defined"
	GetFieldMappings() map[string]string

	// GetConnectorType returns whether this is a "source" or "destination".
	GetConnectorType() ConnectorType

	// GetConnectorCode returns the connector code (e.g., "postgresql", "snowflake").
	GetConnectorCode() string

	// GetResourceName returns the full resource name (e.g., "source_postgresql").
	GetResourceName() string

	// NewModelInstance creates a new instance of the connector's model struct.
	// This is needed for reflection-based operations.
	NewModelInstance() any
}

// ConnectorConfigWithJSONStringFields is an optional interface that connectors
// can implement to specify fields that should be serialized as JSON strings
// when sent to the API, rather than as nested objects.
// This is needed for fields where the backend expects a JSON string (textarea control)
// but Terraform uses a nested map for better UX.
type ConnectorConfigWithJSONStringFields interface {
	// GetJSONStringFields returns a list of Terraform attribute names that should
	// be serialized as JSON strings when sent to the API.
	// Example: []string{"topics_config_map"} means the nested map should be
	// JSON-stringified before sending to the API.
	GetJSONStringFields() []string
}

// smtChainModel is a generated model that carries the chain fields. tfgen
// emits SMTChainHost on every source and destination model the backend chain
// API can manage; the base resource builds that host's chain from the pinned
// catalog, adds the attributes and drives the chain lifecycle.
type smtChainModel interface {
	SMTChainHost() string
}

// ConnectorConfigWithSMTChain overrides the chain schema a model's host would
// get, for a model that already carries the chain fields. Tests use it to
// drive the lifecycle with the contract corpus instead of the pinned catalog.
type ConnectorConfigWithSMTChain interface {
	GetSMTChain() *smt.ChainSchema
}

// Ensure BaseConnectorResource satisfies framework interfaces.
var (
	_ resource.Resource                   = &BaseConnectorResource{}
	_ resource.ResourceWithConfigure      = &BaseConnectorResource{}
	_ resource.ResourceWithImportState    = &BaseConnectorResource{}
	_ resource.ResourceWithModifyPlan     = &BaseConnectorResource{}
	_ resource.ResourceWithValidateConfig = &BaseConnectorResource{}
)

// BaseConnectorResource is a generic resource implementation for connectors.
// It implements the standard Terraform Resource interface and uses ConnectorConfig
// for connector-specific behavior.
type BaseConnectorResource struct {
	client api.StreamkapAPI
	config ConnectorConfig
	// smt is nil unless the config opts into the nested chain.
	smt *smt.Surface
}

// NewBaseConnectorResource creates a new BaseConnectorResource with the given config.
func NewBaseConnectorResource(config ConnectorConfig) resource.Resource {
	r := &BaseConnectorResource{
		config: config,
	}
	if c, ok := config.(ConnectorConfigWithSMTChain); ok {
		r.smt = smt.NewSurface(c.GetSMTChain(), config.GetFieldMappings())
	} else if m, ok := config.NewModelInstance().(smtChainModel); ok {
		r.smt = smt.NewSurface(generated.SMTChain(m.SMTChainHost()), config.GetFieldMappings())
	}
	return r
}

// Metadata returns the resource type name.
func (r *BaseConnectorResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_" + r.config.GetResourceName()
}

// Schema returns the schema for this resource.
func (r *BaseConnectorResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	baseSchema := r.config.GetSchema()
	// Read refreshes connector_status before planning. Preserve that observed
	// value through the plan instead of replacing it with an unknown value.
	status := baseSchema.Attributes["connector_status"].(schema.StringAttribute)
	status.PlanModifiers = append(status.PlanModifiers, stringplanmodifier.UseStateForUnknown())
	baseSchema.Attributes["connector_status"] = status

	// Deprecated computed aliases retain an absent prior value instead of
	// planning it unknown. Alias defaults must wait until ModifyPlan; applying
	// them earlier marks computed fields unknown even when the alias resolves
	// to unchanged state.
	mappings := r.config.GetFieldMappings()
	for alias, attribute := range baseSchema.Attributes {
		if attribute.GetDeprecationMessage() == "" {
			continue
		}
		switch a := attribute.(type) {
		case schema.StringAttribute:
			if a.Computed {
				a.PlanModifiers = append(a.PlanModifiers, stringplanmodifier.UseStateForUnknown())
				baseSchema.Attributes[alias] = a
			}
		case schema.BoolAttribute:
			if a.Computed {
				a.PlanModifiers = append(a.PlanModifiers, boolplanmodifier.UseStateForUnknown())
				baseSchema.Attributes[alias] = a
			}
		case schema.Int64Attribute:
			if a.Computed {
				a.PlanModifiers = append(a.PlanModifiers, int64planmodifier.UseStateForUnknown())
				baseSchema.Attributes[alias] = a
			}
		}
		for canonical, apiField := range mappings {
			if canonical == alias || apiField != mappings[alias] {
				continue
			}
			switch a := baseSchema.Attributes[canonical].(type) {
			case schema.StringAttribute:
				if a.Default != nil {
					a.Default = nil
					a.PlanModifiers = append(a.PlanModifiers, stringplanmodifier.UseStateForUnknown())
					baseSchema.Attributes[canonical] = a
				}
			case schema.BoolAttribute:
				if a.Default != nil {
					a.Default = nil
					a.PlanModifiers = append(a.PlanModifiers, boolplanmodifier.UseStateForUnknown())
					baseSchema.Attributes[canonical] = a
				}
			case schema.Int64Attribute:
				if a.Default != nil {
					a.Default = nil
					a.PlanModifiers = append(a.PlanModifiers, int64planmodifier.UseStateForUnknown())
					baseSchema.Attributes[canonical] = a
				}
			}
		}
	}

	if r.smt != nil {
		r.smt.AddAttributes(baseSchema.Attributes)
		if r.config.GetConnectorType() == ConnectorTypeSource {
			overrides := baseSchema.Attributes[smt.AttrManagedOverrides].(schema.MapAttribute)
			overrides.Description = "Managed transform slots are destination-scoped, so a source chain accepts no override: leave this unset or set an empty map."
			overrides.MarkdownDescription = overrides.Description
			baseSchema.Attributes[smt.AttrManagedOverrides] = overrides
		}
	}

	// Add timeouts block to the schema
	baseSchema.Blocks = map[string]schema.Block{
		"timeouts": timeouts.Block(ctx, timeouts.Opts{
			Create: true,
			Read:   true,
			Update: true,
			Delete: true,
		}),
	}

	resp.Schema = baseSchema
}

// ValidateConfig rejects a configuration that mixes the released flat SMT
// attributes with the nested chain. Connectors without the chain have
// nothing to validate here.
func (r *BaseConnectorResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	if r.smt == nil {
		return
	}
	resp.Diagnostics.Append(r.smt.ValidateConfig(ctx, req.Config)...)
	if r.config.GetConnectorType() == ConnectorTypeSource {
		var overrides types.Map
		resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root(smt.AttrManagedOverrides), &overrides)...)
		// The backend refuses every explicit override on a source chain
		// (managed slots are destination-scoped), so stop before any request.
		if !overrides.IsNull() && !overrides.IsUnknown() && len(overrides.Elements()) > 0 {
			resp.Diagnostics.AddAttributeError(path.Root(smt.AttrManagedOverrides), "Managed overrides apply to destinations only",
				fmt.Sprintf("Platform-managed transform slots are destination-scoped, so a source chain accepts no override. Remove the entries or set %s = {}.", smt.AttrManagedOverrides))
		}
	}
}

// ModifyPlan aligns aliases before marshaling: schema defaults can otherwise
// give two attributes different planned values for the same API field. With
// the chain enabled it then correlates chain entries to prior state by key.
func (r *BaseConnectorResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.Plan.Raw.IsNull() {
		return
	}
	defer func() {
		if r.smt != nil && !resp.Diagnostics.HasError() {
			resp.Diagnostics.Append(r.smt.ModifyPlan(ctx, req.Config, req.State, &resp.Plan)...)
		}
	}()
	model := r.config.NewModelInstance()
	resp.Diagnostics.Append(req.Config.Get(ctx, model)...)
	if resp.Diagnostics.HasError() {
		return
	}
	mappings := r.config.GetFieldMappings()
	names := make([]string, 0, len(mappings))
	for name := range mappings {
		names = append(names, name)
	}
	configured := shared.CaptureFields(model, names)
	s := r.config.GetSchema()
	for alias, attribute := range s.Attributes {
		if attribute.GetDeprecationMessage() == "" {
			continue
		}
		for canonical, apiField := range mappings {
			if canonical == alias || apiField != mappings[alias] {
				continue
			}
			oldValue, newValue := configured[alias], configured[canonical]
			value := oldValue
			if value.IsNull() {
				value = newValue
			}
			if value.IsNull() {
				p := path.Root(canonical)
				switch a := s.Attributes[canonical].(type) {
				case schema.StringAttribute:
					if a.Default != nil {
						var d defaults.StringResponse
						a.Default.DefaultString(ctx, defaults.StringRequest{Path: p}, &d)
						resp.Diagnostics.Append(d.Diagnostics...)
						value = d.PlanValue
					}
				case schema.BoolAttribute:
					if a.Default != nil {
						var d defaults.BoolResponse
						a.Default.DefaultBool(ctx, defaults.BoolRequest{Path: p}, &d)
						resp.Diagnostics.Append(d.Diagnostics...)
						value = d.PlanValue
					}
				case schema.Int64Attribute:
					if a.Default != nil {
						var d defaults.Int64Response
						a.Default.DefaultInt64(ctx, defaults.Int64Request{Path: p}, &d)
						resp.Diagnostics.Append(d.Diagnostics...)
						value = d.PlanValue
					}
				}
			}
			if value.IsNull() || resp.Diagnostics.HasError() {
				continue
			}
			if newValue.IsNull() && s.Attributes[canonical].IsComputed() {
				resp.Diagnostics.Append(resp.Plan.SetAttribute(ctx, path.Root(canonical), value)...)
			}
			if oldValue.IsNull() && attribute.IsComputed() {
				resp.Diagnostics.Append(resp.Plan.SetAttribute(ctx, path.Root(alias), value)...)
			}
		}
	}
}

// Config exposes the connector's ConnectorConfig. Provider-level tests use it to
// cross-check field mappings against the model and schema, which would otherwise
// require reaching into unexported state.
func (r *BaseConnectorResource) Config() ConnectorConfig {
	return r.config
}

// Configure sets the API client for this resource.
func (r *BaseConnectorResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(api.StreamkapAPI)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected api.StreamkapAPI, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}
	r.client = client
}

// Create creates a new connector resource.
func (r *BaseConnectorResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError(
			"Unconfigured Resource",
			"Expected configured API client. Please report this issue to the provider developers.",
		)
		return
	}

	// Get timeout from config
	var timeoutsValue timeouts.Value
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("timeouts"), &timeoutsValue)...)
	if resp.Diagnostics.HasError() {
		return
	}

	createTimeout, diags := timeoutsValue.Create(ctx, helper.DefaultCreateTimeout)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(ctx, createTimeout)
	defer cancel()

	// Create a new model instance for this connector
	model := r.config.NewModelInstance()

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Capture user-supplied secrets before the API echo can overwrite them, and
	// the defaulted attributes the backend may drop (see shared.DefaultedAttrNames).
	plannedSecrets := shared.CaptureFields(model, shared.SensitiveStringAttrNames(r.config.GetSchema()))
	plannedDefaults := shared.CaptureFields(model, shared.DefaultedAttrNames(r.config.GetSchema()))

	// Get name from model
	name := r.getStringField(model, "Name")
	if name == "" {
		resp.Diagnostics.AddError(
			"Missing Required Field",
			"The 'name' field is required but was not provided.",
		)
		return
	}

	// Convert model to config map using field mappings
	configMap, err := r.modelToConfigMap(ctx, model)
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Error creating %s %s", r.config.GetConnectorType(), r.config.GetConnectorCode()),
			fmt.Sprintf("Unable to marshal configuration: %s", err),
		)
		return
	}

	// Nothing precedes a create: no chain and no revision.
	var priorChain, priorPredicates types.List
	priorOverrides := types.MapNull(types.BoolType)
	priorRevision := types.StringNull()
	if r.smt != nil {
		priorChain = types.ListNull(r.smt.Chain.EntryType)
		priorPredicates = types.ListNull(r.smt.Chain.PredicateEntryType)
	}
	chain, plannedFlat, ok := r.planChain(ctx, req.Plan, priorChain, priorPredicates, priorOverrides, req.Config, configMap, &resp.Diagnostics)
	if !ok {
		return
	}

	// Never log configMap: it holds decrypted credentials keyed by API field name
	// and bypasses the redaction applied to the request body in internal/api.
	tflog.Debug(ctx, fmt.Sprintf("Creating %s %s", r.config.GetConnectorType(), r.config.GetConnectorCode()))

	// Call the appropriate API based on connector type
	var id string
	var connectorName string
	var connectorCode string
	var connectorStatus string
	var responseConfig map[string]any

	kcClusterId := r.getStringField(model, "KcClusterId")
	tags := r.getStringSliceField(ctx, model, "Tags")
	var responseTags []string
	var receipt *api.SourceSMTChainReceipt
	var chainSent bool

	switch r.config.GetConnectorType() {
	case ConnectorTypeSource:
		embedded, ok := r.embeddedChain(chain, priorRevision, &resp.Diagnostics)
		if !ok {
			return
		}
		source, err := r.client.CreateSource(ctx, api.Source{
			Name:        name,
			Connector:   r.config.GetConnectorCode(),
			Config:      configMap,
			KcClusterId: kcClusterId,
			Tags:        tags,
			SMTChain:    embedded,
		})
		if err != nil {
			r.addSourceWriteError(&resp.Diagnostics, "creating", "create", err, chain, priorRevision)
			return
		}
		receipt = source.SMTChainReceipt
		chainSent = embedded != nil
		id = source.ID
		connectorName = source.Name
		connectorCode = source.Connector
		connectorStatus = source.ConnectorStatus
		responseConfig = source.Config
		responseTags = source.Tags

	case ConnectorTypeDestination:
		destination, err := r.client.CreateDestination(ctx, api.Destination{
			Name:        name,
			Connector:   r.config.GetConnectorCode(),
			Config:      configMap,
			KcClusterId: kcClusterId,
			Tags:        tags,
		})
		if err != nil {
			resp.Diagnostics.AddError(
				fmt.Sprintf("Error creating %s destination", r.config.GetConnectorCode()),
				fmt.Sprintf("Unable to create destination: %s", err),
			)
			return
		}
		id = destination.ID
		connectorName = destination.Name
		connectorCode = destination.Connector
		connectorStatus = destination.ConnectorStatus
		responseConfig = destination.Config
		responseTags = destination.Tags
	}

	// Update model with response data
	r.setStringField(model, "ID", id)
	r.setStringField(model, "Name", connectorName)
	r.setStringField(model, "Connector", connectorCode)
	r.setStringField(model, "ConnectorStatus", connectorStatus)
	r.setStringSliceField(model, "Tags", normalizeTagsResponse(tags, responseTags))
	r.configMapToModel(ctx, responseConfig, model)
	shared.PreserveKnownFields(model, plannedSecrets)
	shared.FillNullFields(model, plannedDefaults)
	shared.PinFields(model, plannedFlat)

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, model)...)
	// A source saved its chain in the same request; only the read-back remains.
	if r.config.GetConnectorType() == ConnectorTypeSource {
		r.recordEmbeddedChain(ctx, id, chain, chainSent, receipt, priorChain, priorPredicates, priorOverrides, priorRevision, &resp.State, &resp.Diagnostics)
		return
	}
	// The connector exists from here on. A failed chain write still records
	// it, so Terraform taints the resource instead of losing the record and
	// colliding on the name at the next apply.
	r.applyChain(ctx, id, chain, priorChain, priorPredicates, priorOverrides, priorRevision, &resp.State, &resp.Diagnostics)
}

// applyInitialChainUpdate creates a destination's first chain before any generic connector update can overwrite legacy SMT state; the backend repeats the adoption classification inside the tokenless PUT, and mixed ordinary connector edits are refused because the two authorities cannot be made atomic by the provider.
func (r *BaseConnectorResource) applyInitialChainUpdate(ctx context.Context, id string, plannedModel any, priorState tfsdk.State, write *smt.Write, priorRevision types.String, plannedConfig map[string]any, state *tfsdk.State, diags *diag.Diagnostics) bool {
	if r.config.GetConnectorType() != ConnectorTypeDestination || r.smt == nil || write == nil || !write.Configured || write.Unchanged {
		return false
	}
	if priorRevision.IsUnknown() {
		diags.AddAttributeError(path.Root(smt.AttrChain), "Transform chain revision is unknown", "The initial transform chain was not written because Terraform could not determine whether a chain already exists. Refresh the resource and plan again.")
		return true
	}
	if !priorRevision.IsNull() {
		return false
	}
	priorModel := r.config.NewModelInstance()
	diags.Append(priorState.Get(ctx, priorModel)...)
	if diags.HasError() {
		return true
	}
	priorConfig, err := r.modelToConfigMap(ctx, priorModel)
	if err != nil {
		diags.AddAttributeError(path.Root(smt.AttrChain), "Unable to compare connector settings", fmt.Sprintf("The initial transform chain was not written because the prior connector configuration could not be marshalled: %s", err))
		return true
	}
	r.smt.StripFlat(priorConfig)
	if !reflect.DeepEqual(priorConfig, plannedConfig) || r.getStringField(priorModel, "Name") != r.getStringField(plannedModel, "Name") || r.getStringField(priorModel, "KcClusterId") != r.getStringField(plannedModel, "KcClusterId") || !reflect.DeepEqual(r.getStringSliceField(ctx, priorModel, "Tags"), r.getStringSliceField(ctx, plannedModel, "Tags")) {
		diags.AddAttributeError(path.Root(smt.AttrChain), "Initial transform chain must be applied separately", "The connector settings and its first transform chain cannot be changed safely in one update. Apply the transform chain while the connector settings are unchanged, then apply the connector changes.")
		return true
	}
	client, ok := r.chainClient(diags)
	if !ok {
		return true
	}
	audit, err := client.GetSMTChainAdoption(ctx, r.chainKind(), id)
	if err != nil {
		diags.AddAttributeError(path.Root(smt.AttrChain), "Unable to inspect legacy transforms", fmt.Sprintf("The initial transform chain was not written because the destination could not be audited: %s", err))
		return true
	}
	switch audit.Status {
	case api.SMTAdoptionAvailable:
		diags.AddAttributeError(path.Root(smt.AttrChain), "Legacy transforms require explicit adoption", "This destination has legacy transforms that Terraform must not replace implicitly. Review and adopt them through the SMT chain adoption API or UI, then refresh Terraform state.")
		return true
	case api.SMTAdoptionUnavailable:
		message := audit.Message
		if message == "" {
			message = "The destination did not provide a safe adoption classification."
		}
		diags.AddAttributeError(path.Root(smt.AttrChain), "Transform chain cannot be created", message)
		return true
	case api.SMTAdoptionManagedOnly:
		// The tokenless PUT immediately below repeats this classification under
		// the backend's captured connector authority.
	default:
		diags.AddAttributeError(path.Root(smt.AttrChain), "Invalid transform adoption response", fmt.Sprintf("The destination returned unknown adoption status %q; no connector or chain change was sent.", audit.Status))
		return true
	}
	applied, applyDiags := r.smt.ApplyGraph(ctx, client, r.chainKind(), id, write, priorRevision)
	diags.Append(applyDiags...)
	if diags.HasError() {
		return true
	}
	r.setStringField(plannedModel, "ConnectorStatus", r.getStringField(priorModel, "ConnectorStatus"))
	diags.Append(state.Set(ctx, plannedModel)...)
	diags.Append(state.SetAttribute(ctx, path.Root(smt.AttrChain), applied.Transforms)...)
	diags.Append(state.SetAttribute(ctx, path.Root(smt.AttrPredicates), applied.Predicates)...)
	diags.Append(state.SetAttribute(ctx, path.Root(smt.AttrManagedOverrides), applied.ManagedOverrides)...)
	diags.Append(state.SetAttribute(ctx, path.Root(smt.AttrRevision), applied.Revision)...)
	return true
}

// Read reads the connector resource.
func (r *BaseConnectorResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError(
			"Unconfigured Resource",
			"Expected configured API client. Please report this issue to the provider developers.",
		)
		return
	}

	// Get timeout from state (Read runs without a config on refresh)
	var timeoutsValue timeouts.Value
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("timeouts"), &timeoutsValue)...)
	if resp.Diagnostics.HasError() {
		return
	}

	readTimeout, diags := timeoutsValue.Read(ctx, helper.DefaultReadTimeout)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(ctx, readTimeout)
	defer cancel()

	// Create a new model instance for this connector
	model := r.config.NewModelInstance()

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Snapshot the secrets already in state. The API response can null them out
	// (see shared.FillNullFields); prior state is the only source we have
	// on refresh, since Read gets no plan.
	priorSecrets := shared.CaptureFields(model, shared.SensitiveStringAttrNames(r.config.GetSchema()))
	// Same for defaulted attributes the backend drops when their gating
	// condition is unmet: nulling them in state would plan the Default back
	// every run.
	priorDefaults := shared.CaptureFields(model, shared.DefaultedAttrNames(r.config.GetSchema()))

	// Get ID from model
	id := r.getStringField(model, "ID")
	if id == "" {
		resp.Diagnostics.AddError(
			"Missing Resource ID",
			"The resource ID is missing from state.",
		)
		return
	}

	tflog.Debug(ctx, fmt.Sprintf("Reading %s %s with ID: %s", r.config.GetConnectorType(), r.config.GetConnectorCode(), id))

	// Call the appropriate API based on connector type
	var connectorName string
	var connectorCode string
	var connectorStatus string
	var kcClusterId string
	var responseConfig map[string]any
	var responseTags []string

	switch r.config.GetConnectorType() {
	case ConnectorTypeSource:
		source, err := r.client.GetSource(ctx, id)
		if err != nil {
			resp.Diagnostics.AddError(
				fmt.Sprintf("Error reading %s source", r.config.GetConnectorCode()),
				fmt.Sprintf("Unable to read source: %s", err),
			)
			return
		}
		if source == nil {
			resp.State.RemoveResource(ctx)
			return
		}
		connectorName = source.Name
		connectorCode = source.Connector
		connectorStatus = source.ConnectorStatus
		kcClusterId = source.KcClusterId
		responseConfig = source.Config
		responseTags = source.Tags

	case ConnectorTypeDestination:
		destination, err := r.client.GetDestination(ctx, id)
		if err != nil {
			resp.Diagnostics.AddError(
				fmt.Sprintf("Error reading %s destination", r.config.GetConnectorCode()),
				fmt.Sprintf("Unable to read destination: %s", err),
			)
			return
		}
		if destination == nil {
			resp.State.RemoveResource(ctx)
			return
		}
		connectorName = destination.Name
		connectorCode = destination.Connector
		connectorStatus = destination.ConnectorStatus
		kcClusterId = destination.KcClusterId
		responseConfig = destination.Config
		responseTags = destination.Tags
	}

	// Update model with response data
	priorTags := r.getStringSliceField(ctx, model, "Tags")
	r.setStringField(model, "Name", connectorName)
	r.setStringField(model, "Connector", connectorCode)
	r.setStringField(model, "ConnectorStatus", connectorStatus)
	r.setStringField(model, "KcClusterId", kcClusterId)
	r.setStringSliceField(model, "Tags", normalizeTagsResponse(priorTags, responseTags))
	r.configMapToModel(ctx, responseConfig, model)
	shared.FillNullFields(model, priorSecrets)
	shared.FillNullFields(model, priorDefaults)

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, model)...)
	r.refreshChain(ctx, id, req.State, &resp.State, &resp.Diagnostics)
}

// Update updates the connector resource.
func (r *BaseConnectorResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError(
			"Unconfigured Resource",
			"Expected configured API client. Please report this issue to the provider developers.",
		)
		return
	}

	// Get timeout from config
	var timeoutsValue timeouts.Value
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("timeouts"), &timeoutsValue)...)
	if resp.Diagnostics.HasError() {
		return
	}

	updateTimeout, diags := timeoutsValue.Update(ctx, helper.DefaultUpdateTimeout)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(ctx, updateTimeout)
	defer cancel()

	// Create a new model instance for this connector
	model := r.config.NewModelInstance()

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Capture user-supplied secrets before the API echo can overwrite them, and
	// the defaulted attributes the backend may drop (see shared.DefaultedAttrNames).
	plannedSecrets := shared.CaptureFields(model, shared.SensitiveStringAttrNames(r.config.GetSchema()))
	plannedDefaults := shared.CaptureFields(model, shared.DefaultedAttrNames(r.config.GetSchema()))

	// Get ID and name from model
	id := r.getStringField(model, "ID")
	if id == "" {
		resp.Diagnostics.AddError(
			"Missing Resource ID",
			"Cannot update resource: ID is missing from state.",
		)
		return
	}
	name := r.getStringField(model, "Name")

	// Convert model to config map using field mappings
	configMap, err := r.modelToConfigMap(ctx, model)
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Error updating %s %s", r.config.GetConnectorType(), r.config.GetConnectorCode()),
			fmt.Sprintf("Unable to marshal configuration: %s", err),
		)
		return
	}

	var priorChain, priorPredicates types.List
	var priorOverrides types.Map
	var priorRevision types.String
	if r.smt != nil {
		resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root(smt.AttrChain), &priorChain)...)
		resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root(smt.AttrPredicates), &priorPredicates)...)
		resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root(smt.AttrManagedOverrides), &priorOverrides)...)
		resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root(smt.AttrRevision), &priorRevision)...)
		if resp.Diagnostics.HasError() {
			return
		}
	}
	chain, plannedFlat, ok := r.planChain(ctx, req.Plan, priorChain, priorPredicates, priorOverrides, req.Config, configMap, &resp.Diagnostics)
	if !ok {
		return
	}
	if r.applyInitialChainUpdate(ctx, id, model, req.State, chain, priorRevision, configMap, &resp.State, &resp.Diagnostics) {
		return
	}

	tflog.Debug(ctx, fmt.Sprintf("Updating %s %s with ID: %s", r.config.GetConnectorType(), r.config.GetConnectorCode(), id))

	// Call the appropriate API based on connector type
	var connectorName string
	var connectorCode string
	var connectorStatus string
	var responseConfig map[string]any

	kcClusterId := r.getStringField(model, "KcClusterId")
	tags := r.getStringSliceField(ctx, model, "Tags")
	var responseTags []string
	var receipt *api.SourceSMTChainReceipt
	var chainSent bool

	switch r.config.GetConnectorType() {
	case ConnectorTypeSource:
		embedded, ok := r.embeddedChain(chain, priorRevision, &resp.Diagnostics)
		if !ok || !r.requireNoUnmanagedSourceChain(ctx, id, embedded, &resp.Diagnostics) {
			return
		}
		source, err := r.client.UpdateSource(ctx, id, api.Source{
			Name:        name,
			Connector:   r.config.GetConnectorCode(),
			Config:      configMap,
			KcClusterId: kcClusterId,
			Tags:        tags,
			SMTChain:    embedded,
		})
		if err != nil {
			r.addSourceWriteError(&resp.Diagnostics, "updating", "update", err, chain, priorRevision)
			return
		}
		receipt = source.SMTChainReceipt
		chainSent = embedded != nil
		connectorName = source.Name
		connectorCode = source.Connector
		connectorStatus = source.ConnectorStatus
		responseConfig = source.Config
		responseTags = source.Tags

	case ConnectorTypeDestination:
		destination, err := r.client.UpdateDestination(ctx, id, api.Destination{
			Name:        name,
			Connector:   r.config.GetConnectorCode(),
			Config:      configMap,
			KcClusterId: kcClusterId,
			Tags:        tags,
		})
		if err != nil {
			resp.Diagnostics.AddError(
				fmt.Sprintf("Error updating %s destination", r.config.GetConnectorCode()),
				fmt.Sprintf("Unable to update destination: %s", err),
			)
			return
		}
		connectorName = destination.Name
		connectorCode = destination.Connector
		connectorStatus = destination.ConnectorStatus
		responseConfig = destination.Config
		responseTags = destination.Tags
	}

	// Update model with response data.
	r.setStringField(model, "Name", connectorName)
	r.setStringField(model, "Connector", connectorCode)
	r.setStringSliceField(model, "Tags", normalizeTagsResponse(tags, responseTags))
	r.configMapToModel(ctx, responseConfig, model)
	shared.PreserveKnownFields(model, plannedSecrets)
	shared.FillNullFields(model, plannedDefaults)
	shared.PinFields(model, plannedFlat)

	// connector_status handling on Update.
	//
	// The PUT response (sent with wait=false) reports the transient
	// desired_state "Pending Update", while for an in-place update the plan
	// carries the prior KNOWN status ("Active"/"Unassigned") — Terraform copies
	// the prior state value for a computed attribute that isn't being changed.
	// Writing the transient response value would make the applied value differ
	// from the planned value, which Terraform rejects with "Provider produced
	// inconsistent result after apply: .connector_status was Active but now
	// Pending Update". So when the plan value is known we keep it (the model
	// already holds it from req.Plan.Get) and let Read refresh the real status
	// once the reconciler settles the connector back to Active.
	//
	// If the plan value is unknown/null (a computed value with no prior state),
	// we must write a concrete value or the framework rejects the unknown left
	// in state — use the response value, which is always known.
	var plannedStatus types.String
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("connector_status"), &plannedStatus)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if plannedStatus.IsUnknown() || plannedStatus.IsNull() {
		r.setStringField(model, "ConnectorStatus", connectorStatus)
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, model)...)
	if r.config.GetConnectorType() == ConnectorTypeSource {
		r.recordEmbeddedChain(ctx, id, chain, chainSent, receipt, priorChain, priorPredicates, priorOverrides, priorRevision, &resp.State, &resp.Diagnostics)
		return
	}
	r.applyChain(ctx, id, chain, priorChain, priorPredicates, priorOverrides, priorRevision, &resp.State, &resp.Diagnostics)
}

// Delete deletes the connector resource.
func (r *BaseConnectorResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError(
			"Unconfigured Resource",
			"Expected configured API client. Please report this issue to the provider developers.",
		)
		return
	}

	// Get timeout from config
	var timeoutsValue timeouts.Value
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("timeouts"), &timeoutsValue)...)
	if resp.Diagnostics.HasError() {
		return
	}

	deleteTimeout, diags := timeoutsValue.Delete(ctx, helper.DefaultDeleteTimeout)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(ctx, deleteTimeout)
	defer cancel()

	// Create a new model instance for this connector
	model := r.config.NewModelInstance()

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get ID from model
	id := r.getStringField(model, "ID")
	if id == "" {
		resp.Diagnostics.AddError(
			"Missing Resource ID",
			"Cannot delete resource: ID is missing from state.",
		)
		return
	}

	tflog.Debug(ctx, fmt.Sprintf("Deleting %s %s with ID: %s", r.config.GetConnectorType(), r.config.GetConnectorCode(), id))

	// Call the appropriate API based on connector type
	switch r.config.GetConnectorType() {
	case ConnectorTypeSource:
		err := r.client.DeleteSource(ctx, id)
		if err != nil {
			resp.Diagnostics.AddError(
				fmt.Sprintf("Error deleting %s source", r.config.GetConnectorCode()),
				fmt.Sprintf("Unable to delete source: %s", err),
			)
			return
		}

	case ConnectorTypeDestination:
		err := r.client.DeleteDestination(ctx, id)
		if err != nil {
			resp.Diagnostics.AddError(
				fmt.Sprintf("Error deleting %s destination", r.config.GetConnectorCode()),
				fmt.Sprintf("Unable to delete destination: %s", err),
			)
			return
		}
	}
}

// ImportState imports an existing resource by ID. With the chain enabled it
// also writes the empty chain that marks the surface as managed, so the Read
// that follows adopts the server instances under keys derived from their ids.
func (r *BaseConnectorResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
	if r.smt != nil {
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root(smt.AttrChain), r.smt.ImportMarker())...)
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root(smt.AttrPredicates), r.smt.PredicateImportMarker())...)
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root(smt.AttrManagedOverrides), types.MapValueMust(types.BoolType, map[string]attr.Value{}))...)
	}
}

// chainKind is the API path segment for this connector type.
func (r *BaseConnectorResource) chainKind() string {
	return string(r.config.GetConnectorType()) + "s"
}

// chainClient is the API client's chain surface, or an error diagnostic when
// the configured client cannot drive the chain.
func (r *BaseConnectorResource) chainClient(diags *diag.Diagnostics) (api.SMTChainAPI, bool) {
	client, ok := r.client.(api.SMTChainAPI)
	if !ok {
		diags.AddError("Unconfigured Resource", fmt.Sprintf("Expected an API client that manages transform chains, got %T. Please report this issue to the provider developers.", r.client))
	}
	return client, ok
}

// planChain projects the planned chain and secret operations before the
// connector request is sent, and drops every flat SMT key from the wire
// config when the chain owns the surface. It returns the planned values of
// the flat SMT attributes so the API echo cannot move them. Without the chain
// everything is a no-op.
func (r *BaseConnectorResource) planChain(ctx context.Context, plan tfsdk.Plan, prior, priorPredicates types.List, priorOverrides types.Map, config tfsdk.Config, configMap map[string]any, diags *diag.Diagnostics) (*smt.Write, map[string]attr.Value, bool) {
	if r.smt == nil {
		return nil, nil, true
	}
	write, d := r.smt.PlanWrite(ctx, plan, prior, config, priorPredicates, priorOverrides)
	diags.Append(d...)
	if diags.HasError() {
		return nil, nil, false
	}
	if !write.ChainConfigured {
		return write, nil, true
	}
	r.smt.StripFlat(configMap)
	model := r.config.NewModelInstance()
	diags.Append(plan.Get(ctx, model)...)
	if diags.HasError() {
		return nil, nil, false
	}
	return write, shared.CaptureFields(model, r.smt.FlatSMT), true
}

// applyChain writes the chain and the secret operations after the connector
// request succeeded, and records the result in state. A failed write keeps
// the prior chain and revision: the server chain did not move, and state
// must hold no unknown from the plan.
func (r *BaseConnectorResource) applyChain(ctx context.Context, id string, write *smt.Write, priorChain, priorPredicates types.List, priorOverrides types.Map, priorRevision types.String, state *tfsdk.State, diags *diag.Diagnostics) {
	if r.smt == nil || diags.HasError() {
		return
	}
	chain, predicates, overrides, revision := priorChain, priorPredicates, priorOverrides, priorRevision
	if client, ok := r.chainClient(diags); ok {
		applied, d := r.smt.ApplyGraph(ctx, client, r.chainKind(), id, write, priorRevision)
		diags.Append(d...)
		if !d.HasError() {
			chain, predicates, overrides, revision = applied.Transforms, applied.Predicates, applied.ManagedOverrides, applied.Revision
		}
	}
	diags.Append(state.SetAttribute(ctx, path.Root(smt.AttrChain), chain)...)
	diags.Append(state.SetAttribute(ctx, path.Root(smt.AttrPredicates), predicates)...)
	diags.Append(state.SetAttribute(ctx, path.Root(smt.AttrManagedOverrides), overrides)...)
	diags.Append(state.SetAttribute(ctx, path.Root(smt.AttrRevision), revision)...)
}

// embeddedChain is the chain write a source create or update carries: sources
// have no chain route, the backend saves the chain atomically with the source
// body. It is nil when nothing is to be written, which omits the key and so
// keeps the stored chain. ok is false when the write cannot be built.
func (r *BaseConnectorResource) embeddedChain(write *smt.Write, expectedRevision types.String, diags *diag.Diagnostics) (*api.SMTChainWrite, bool) {
	if r.smt == nil || write == nil {
		return nil, true
	}
	if _, unsent := r.smt.Unsent(write, expectedRevision); unsent {
		return nil, true
	}
	req, d := r.smt.WriteRequest(write, expectedRevision)
	diags.Append(d...)
	return req, !d.HasError()
}

// requireNoUnmanagedSourceChain refuses a first chain write to an existing
// source that already has a chain this resource does not manage. The source
// routes answer that case with a plain 400, so the import guidance the
// destination route's smt_expected_revision_required carries is given here,
// before the write. A chain created between this read and the write is still
// refused by the backend.
func (r *BaseConnectorResource) requireNoUnmanagedSourceChain(ctx context.Context, id string, embedded *api.SMTChainWrite, diags *diag.Diagnostics) bool {
	if embedded == nil || embedded.ExpectedRevision != nil {
		return true
	}
	client, ok := r.chainClient(diags)
	if !ok {
		return false
	}
	_, err := client.GetSMTChain(ctx, r.chainKind(), id)
	switch {
	case api.IsSMTChainAbsent(err):
		return true
	case err != nil:
		diags.AddAttributeError(path.Root(smt.AttrChain), "Unable to inspect the source transform chain",
			fmt.Sprintf("The first transform chain was not written because the source's existing chain could not be read: %s", err))
	default:
		diags.AddAttributeError(path.Root(smt.AttrChain), "Transform chain already exists outside Terraform",
			fmt.Sprintf("The source already has a transform chain this resource does not manage. Import the resource to adopt it (its instances become %s entries keyed by their server ids), then apply.", smt.AttrChain))
	}
	return false
}

// addSourceWriteError reports a refused source create or update. A structured
// SMT envelope is located on the chain attributes; anything else is reported
// against the source, since the same request carried the connector settings.
func (r *BaseConnectorResource) addSourceWriteError(diags *diag.Diagnostics, verbing, verb string, err error, write *smt.Write, expectedRevision types.String) {
	if _, structured := api.SMTErrorOf(err); structured && r.smt != nil && write != nil && write.Configured {
		diags.Append(r.smt.WriteError(err, expectedRevision, write)...)
		return
	}
	diags.AddError(
		fmt.Sprintf("Error %s %s source", verbing, r.config.GetConnectorCode()),
		fmt.Sprintf("Unable to %s source: %s", verb, err),
	)
}

// recordEmbeddedChain records the chain of a source whose create or update
// succeeded. When the request carried a chain, the chain is read back and must
// be the revision the response accepted; otherwise the read could belong to a
// later edit and its instances would be keyed by the wrong plan entries. A
// failed read keeps the prior chain and revision so state holds no unknown;
// the next refresh reads the real chain.
func (r *BaseConnectorResource) recordEmbeddedChain(ctx context.Context, id string, write *smt.Write, sent bool, receipt *api.SourceSMTChainReceipt, priorChain, priorPredicates types.List, priorOverrides types.Map, priorRevision types.String, state *tfsdk.State, diags *diag.Diagnostics) {
	if r.smt == nil || diags.HasError() {
		return
	}
	graph := smt.GraphState{Transforms: priorChain, Predicates: priorPredicates, ManagedOverrides: priorOverrides, Revision: priorRevision}
	if !sent {
		graph, _ = r.smt.Unsent(write, priorRevision)
	} else if applied, ok := r.readEmbeddedChain(ctx, id, write, receipt, diags); ok {
		graph = applied
	}
	diags.Append(state.SetAttribute(ctx, path.Root(smt.AttrChain), graph.Transforms)...)
	diags.Append(state.SetAttribute(ctx, path.Root(smt.AttrPredicates), graph.Predicates)...)
	diags.Append(state.SetAttribute(ctx, path.Root(smt.AttrManagedOverrides), graph.ManagedOverrides)...)
	diags.Append(state.SetAttribute(ctx, path.Root(smt.AttrRevision), graph.Revision)...)
}

func (r *BaseConnectorResource) readEmbeddedChain(ctx context.Context, id string, write *smt.Write, receipt *api.SourceSMTChainReceipt, diags *diag.Diagnostics) (smt.GraphState, bool) {
	if receipt == nil || receipt.ChainRevision == "" {
		diags.AddAttributeError(path.Root(smt.AttrChain), "Transform chain outcome unconfirmed",
			"The source was saved with its transform chain, but the response did not identify the accepted chain revision. Run terraform plan to refresh the chain before applying again.")
		return smt.GraphState{}, false
	}
	client, ok := r.chainClient(diags)
	if !ok {
		return smt.GraphState{}, false
	}
	read, err := client.GetSMTChain(ctx, r.chainKind(), id)
	if err != nil {
		diags.AddAttributeError(path.Root(smt.AttrChain), "Unable to read the saved transform chain",
			fmt.Sprintf("The source and its transform chain revision %q were saved, but reading the chain back failed. Run terraform plan to refresh it: %s", receipt.ChainRevision, err))
		return smt.GraphState{}, false
	}
	if read.Desired.ChainRevision != receipt.ChainRevision {
		diags.AddAttributeError(path.Root(smt.AttrChain), "Transform chain changed outside Terraform",
			fmt.Sprintf("The source saved transform chain revision %q, but the chain now reads as revision %q. Run terraform plan to refresh it and review the drift before applying.", receipt.ChainRevision, read.Desired.ChainRevision))
		return smt.GraphState{}, false
	}
	applied, d := r.smt.AppliedState(ctx, write, read)
	diags.Append(d...)
	return applied, !d.HasError()
}

// refreshChain reads the chain back under the keys prior state holds. The
// flat SMT attributes are pinned to prior state whenever the chain owns the
// surface, so whatever the backend projects into the legacy keys never
// becomes state.
func (r *BaseConnectorResource) refreshChain(ctx context.Context, id string, priorState tfsdk.State, state *tfsdk.State, diags *diag.Diagnostics) {
	if r.smt == nil || diags.HasError() {
		return
	}
	var prior, priorPredicates types.List
	var priorOverrides types.Map
	diags.Append(priorState.GetAttribute(ctx, path.Root(smt.AttrChain), &prior)...)
	diags.Append(priorState.GetAttribute(ctx, path.Root(smt.AttrPredicates), &priorPredicates)...)
	diags.Append(priorState.GetAttribute(ctx, path.Root(smt.AttrManagedOverrides), &priorOverrides)...)
	if diags.HasError() {
		return
	}
	if prior.IsNull() && priorOverrides.IsNull() {
		diags.Append(state.SetAttribute(ctx, path.Root(smt.AttrRevision), types.StringNull())...)
		return
	}
	client, ok := r.chainClient(diags)
	if !ok {
		return
	}
	graph, d := r.smt.RefreshGraph(ctx, client, r.chainKind(), id, prior, priorPredicates, priorOverrides)
	diags.Append(d...)
	if diags.HasError() {
		return
	}
	for _, name := range r.smt.FlatSMT {
		var v attr.Value
		diags.Append(priorState.GetAttribute(ctx, path.Root(name), &v)...)
		diags.Append(state.SetAttribute(ctx, path.Root(name), v)...)
	}
	diags.Append(state.SetAttribute(ctx, path.Root(smt.AttrChain), graph.Transforms)...)
	diags.Append(state.SetAttribute(ctx, path.Root(smt.AttrPredicates), graph.Predicates)...)
	diags.Append(state.SetAttribute(ctx, path.Root(smt.AttrManagedOverrides), graph.ManagedOverrides)...)
	diags.Append(state.SetAttribute(ctx, path.Root(smt.AttrRevision), graph.Revision)...)
}

// modelToConfigMap converts a model struct to a config map using the field mappings.
// It uses reflection to read values from the model and maps them to API field names.
func (r *BaseConnectorResource) modelToConfigMap(ctx context.Context, model any) (map[string]any, error) {
	configMap, err := shared.ModelToConfigMap(ctx, model, r.config.GetFieldMappings(), r.extractValueHook())
	if err != nil {
		return nil, err
	}

	// Post-process JSON string fields (double-encoding for nested map types)
	jsonStringFields := r.getJSONStringFields()
	if len(jsonStringFields) > 0 {
		mappings := r.config.GetFieldMappings()
		for tfAttr, apiField := range mappings {
			apiValue := configMap[apiField]
			if apiValue != nil && r.isJSONStringField(tfAttr, jsonStringFields) {
				if mapVal, isMap := apiValue.(map[string]map[string]any); isMap {
					stringifiedMap := make(map[string]string)
					for key, innerMap := range mapVal {
						innerBytes, jsonErr := json.Marshal(innerMap)
						if jsonErr != nil {
							tflog.Warn(ctx, fmt.Sprintf("Failed to serialize inner map for %s.%s to JSON: %s", tfAttr, key, jsonErr))
							continue
						}
						stringifiedMap[key] = string(innerBytes)
					}
					jsonBytes, jsonErr := json.Marshal(stringifiedMap)
					if jsonErr != nil {
						tflog.Warn(ctx, fmt.Sprintf("Failed to serialize %s to JSON: %s", tfAttr, jsonErr))
					} else {
						configMap[apiField] = string(jsonBytes)
					}
				}
			}
		}
	}

	return configMap, nil
}

// getJSONStringFields returns the list of fields that should be JSON-stringified.
func (r *BaseConnectorResource) getJSONStringFields() []string {
	if cfg, ok := r.config.(ConnectorConfigWithJSONStringFields); ok {
		return cfg.GetJSONStringFields()
	}
	return nil
}

// isJSONStringField checks if a field is in the JSON string fields list.
func (r *BaseConnectorResource) isJSONStringField(fieldName string, jsonStringFields []string) bool {
	for _, f := range jsonStringFields {
		if f == fieldName {
			return true
		}
	}
	return false
}

// configMapToModel updates a model struct from a config map using the field mappings.
func (r *BaseConnectorResource) configMapToModel(ctx context.Context, cfg map[string]any, model any) {
	shared.ConfigMapToModel(ctx, cfg, model, r.config.GetFieldMappings(), r.setValueHook())
}

// extractValueHook returns a hook for shared.ExtractTerraformValue that handles
// connector-specific types: jsontypes.Normalized, map[string]types.String,
// and map[string]struct (nested map types).
func (r *BaseConnectorResource) extractValueHook() shared.ExtractValueFunc {
	return func(ctx context.Context, fieldValue reflect.Value) (any, bool) {
		// Handle jsontypes.Normalized
		if v, ok := fieldValue.Interface().(jsontypes.Normalized); ok {
			if v.IsNull() || v.IsUnknown() {
				return nil, true
			}
			return v.ValueString(), true
		}

		// Handle map types
		if fieldValue.Kind() == reflect.Map {
			mapType := fieldValue.Type()
			if mapType.Key().Kind() == reflect.String {
				if fieldValue.IsNil() {
					return nil, true
				}

				// map[string]types.String
				if mapType.Elem() == reflect.TypeOf(types.String{}) {
					result := make(map[string]string)
					iter := fieldValue.MapRange()
					for iter.Next() {
						key := iter.Key().String()
						value := iter.Value().Interface().(types.String)
						if !value.IsNull() && !value.IsUnknown() {
							result[key] = value.ValueString()
						}
					}
					return result, true
				}

				// map[string]struct (nested map type)
				if mapType.Elem().Kind() == reflect.Struct {
					result := make(map[string]map[string]any)
					iter := fieldValue.MapRange()
					for iter.Next() {
						key := iter.Key().String()
						structValue := iter.Value()
						nestedMap := r.extractNestedStruct(ctx, structValue)
						if len(nestedMap) > 0 {
							result[key] = nestedMap
						}
					}
					return result, true
				}
			}
		}

		return nil, false // not handled
	}
}

// setValueHook returns a hook for shared.SetTerraformValue that handles
// connector-specific types: jsontypes.Normalized, map[string]types.String,
// and map[string]struct (nested map types).
func (r *BaseConnectorResource) setValueHook() shared.SetValueFunc {
	return func(ctx context.Context, cfg map[string]any, apiField string, fieldValue reflect.Value) bool {
		fieldType := fieldValue.Type()

		// Handle jsontypes.Normalized
		if fieldType == reflect.TypeOf(jsontypes.Normalized{}) {
			if val, ok := cfg[apiField]; ok && val != nil {
				if strVal, ok := val.(string); ok {
					fieldValue.Set(reflect.ValueOf(jsontypes.NewNormalizedValue(strVal)))
				} else {
					fieldValue.Set(reflect.ValueOf(jsontypes.NewNormalizedNull()))
				}
			} else {
				fieldValue.Set(reflect.ValueOf(jsontypes.NewNormalizedNull()))
			}
			return true
		}

		// Handle map types
		if fieldType.Kind() == reflect.Map && fieldType.Key().Kind() == reflect.String {
			// map[string]types.String
			if fieldType.Elem() == reflect.TypeOf(types.String{}) {
				tfVal := helper.GetTfCfgMapString(ctx, cfg, apiField)
				if tfVal != nil {
					fieldValue.Set(reflect.ValueOf(tfVal))
				} else {
					fieldValue.Set(reflect.Zero(fieldType))
				}
				return true
			}

			// map[string]struct (nested map type)
			if fieldType.Elem().Kind() == reflect.Struct {
				r.setNestedMapValue(ctx, cfg, apiField, fieldValue)
				return true
			}
		}

		return false // not handled
	}
}

// extractNestedStruct extracts a struct's fields into a map using tfsdk tags.
func (r *BaseConnectorResource) extractNestedStruct(ctx context.Context, structValue reflect.Value) map[string]any {
	result := make(map[string]any)
	t := structValue.Type()

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		tag := field.Tag.Get("tfsdk")
		if tag == "" || tag == "-" {
			continue
		}

		fieldVal := structValue.Field(i)
		// Extract the value based on its type
		switch v := fieldVal.Interface().(type) {
		case types.String:
			if !v.IsNull() && !v.IsUnknown() {
				result[tag] = v.ValueString()
			}
		case types.Int64:
			if !v.IsNull() && !v.IsUnknown() {
				result[tag] = v.ValueInt64()
			}
		case types.Bool:
			if !v.IsNull() && !v.IsUnknown() {
				result[tag] = v.ValueBool()
			}
		case types.Float64:
			if !v.IsNull() && !v.IsUnknown() {
				result[tag] = v.ValueFloat64()
			}
		default:
			tflog.Debug(ctx, fmt.Sprintf("Unsupported nested field type: %T for field %s", v, tag))
		}
	}

	return result
}

// Note: setTerraformValue is now handled by shared.SetTerraformValue + setValueHook().

// setNestedMapValue sets a map[string]struct field from a config map value.
func (r *BaseConnectorResource) setNestedMapValue(ctx context.Context, cfg map[string]any, apiField string, fieldValue reflect.Value) {
	// Get the value from the config
	val, ok := cfg[apiField]
	if !ok || val == nil {
		fieldValue.Set(reflect.Zero(fieldValue.Type()))
		return
	}

	// The API response could be a map[string]any OR a JSON string (for textarea control fields)
	var apiMap map[string]any

	switch v := val.(type) {
	case map[string]any:
		apiMap = v
	case string:
		// Try to parse as JSON string (backend stores some nested maps as JSON strings)
		if v == "" {
			fieldValue.Set(reflect.Zero(fieldValue.Type()))
			return
		}
		if err := json.Unmarshal([]byte(v), &apiMap); err != nil {
			tflog.Debug(ctx, fmt.Sprintf("Failed to parse JSON string for nested field '%s': %s", apiField, err))
			fieldValue.Set(reflect.Zero(fieldValue.Type()))
			return
		}
	default:
		tflog.Debug(ctx, fmt.Sprintf("Expected map[string]any or JSON string for nested field '%s', got %T", apiField, val))
		fieldValue.Set(reflect.Zero(fieldValue.Type()))
		return
	}

	if apiMap == nil {
		fieldValue.Set(reflect.Zero(fieldValue.Type()))
		return
	}

	// An explicit `{}` clears the mapping and has to round-trip as an empty map.
	// Collapsing it to null makes Terraform reject the apply with
	// "was cty.MapValEmpty(...), but now null".
	if len(apiMap) == 0 {
		fieldValue.Set(reflect.MakeMap(fieldValue.Type()))
		return
	}

	// Create a new map of the correct type
	elemType := fieldValue.Type().Elem()
	newMap := reflect.MakeMap(fieldValue.Type())

	for key, itemValue := range apiMap {
		// Each item could be a map[string]any or a JSON string (double-encoded format)
		// The API returns: {"topic":"{\"key\":\"value\"}"} (inner values are JSON strings)
		var itemMap map[string]any
		switch v := itemValue.(type) {
		case map[string]any:
			itemMap = v
		case string:
			// Try to parse as JSON string (double-encoded format)
			if v == "" {
				continue
			}
			if err := json.Unmarshal([]byte(v), &itemMap); err != nil {
				tflog.Debug(ctx, fmt.Sprintf("Failed to parse JSON string for nested item '%s': %s", key, err))
				continue
			}
		default:
			tflog.Debug(ctx, fmt.Sprintf("Expected map[string]any or JSON string for nested item '%s', got %T", key, itemValue))
			continue
		}

		// Create a new instance of the struct
		newStruct := reflect.New(elemType).Elem()
		r.setNestedStructFields(ctx, itemMap, newStruct)
		newMap.SetMapIndex(reflect.ValueOf(key), newStruct)
	}

	fieldValue.Set(newMap)
}

// setNestedStructFields sets fields on a struct from a map using tfsdk tags.
func (r *BaseConnectorResource) setNestedStructFields(ctx context.Context, itemMap map[string]any, structValue reflect.Value) {
	t := structValue.Type()

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		tag := field.Tag.Get("tfsdk")
		if tag == "" || tag == "-" {
			continue
		}

		fieldVal := structValue.Field(i)
		if !fieldVal.CanSet() {
			continue
		}

		// Get the value from the map
		val, ok := itemMap[tag]
		if !ok || val == nil {
			// Set null value based on type
			switch fieldVal.Type() {
			case reflect.TypeOf(types.String{}):
				fieldVal.Set(reflect.ValueOf(types.StringNull()))
			case reflect.TypeOf(types.Int64{}):
				fieldVal.Set(reflect.ValueOf(types.Int64Null()))
			case reflect.TypeOf(types.Bool{}):
				fieldVal.Set(reflect.ValueOf(types.BoolNull()))
			case reflect.TypeOf(types.Float64{}):
				fieldVal.Set(reflect.ValueOf(types.Float64Null()))
			}
			continue
		}

		// Set the value based on the field type
		switch fieldVal.Type() {
		case reflect.TypeOf(types.String{}):
			if strVal, ok := val.(string); ok {
				fieldVal.Set(reflect.ValueOf(types.StringValue(strVal)))
			} else {
				fieldVal.Set(reflect.ValueOf(types.StringNull()))
			}
		case reflect.TypeOf(types.Int64{}):
			switch v := val.(type) {
			case float64:
				fieldVal.Set(reflect.ValueOf(types.Int64Value(int64(v))))
			case int64:
				fieldVal.Set(reflect.ValueOf(types.Int64Value(v)))
			case int:
				fieldVal.Set(reflect.ValueOf(types.Int64Value(int64(v))))
			default:
				fieldVal.Set(reflect.ValueOf(types.Int64Null()))
			}
		case reflect.TypeOf(types.Bool{}):
			if boolVal, ok := val.(bool); ok {
				fieldVal.Set(reflect.ValueOf(types.BoolValue(boolVal)))
			} else {
				fieldVal.Set(reflect.ValueOf(types.BoolNull()))
			}
		case reflect.TypeOf(types.Float64{}):
			if floatVal, ok := val.(float64); ok {
				fieldVal.Set(reflect.ValueOf(types.Float64Value(floatVal)))
			} else {
				fieldVal.Set(reflect.ValueOf(types.Float64Null()))
			}
		default:
			tflog.Debug(ctx, fmt.Sprintf("Unsupported nested field type: %s for field %s", fieldVal.Type(), tag))
		}
	}
}

func (r *BaseConnectorResource) getStringField(model any, fieldName string) string {
	return shared.GetStringField(model, fieldName)
}

func (r *BaseConnectorResource) setStringField(model any, fieldName string, value string) {
	shared.SetStringField(model, fieldName, value)
}

func (r *BaseConnectorResource) getStringSliceField(ctx context.Context, model any, fieldName string) []string {
	return shared.GetStringSliceField(ctx, model, fieldName)
}

func (r *BaseConnectorResource) setStringSliceField(model any, fieldName string, values []string) {
	shared.SetStringSliceField(model, fieldName, values)
}

// normalizeTagsResponse reconciles user intent with the server response for
// the `tags` attribute. Backend uses `omitempty`, so "no tags" comes back as
// nil, not empty array. When the user explicitly sent `tags = []`, treat a
// nil server response as the equivalent empty slice so state ends up as an
// empty (but non-null) Set — otherwise Terraform sees plan=`[]`, state=Null
// and loops on diff every time.
func normalizeTagsResponse(userTags, serverTags []string) []string {
	if userTags != nil && serverTags == nil {
		return []string{}
	}
	return serverTags
}
