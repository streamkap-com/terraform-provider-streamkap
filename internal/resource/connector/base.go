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
	"errors"
	"fmt"
	"maps"
	"net/http"
	"reflect"
	"slices"
	"strings"

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
	"github.com/hashicorp/terraform-plugin-go/tftypes"
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

// ConnectorConfigValidator checks the planned model at plan time. prior is the
// prior state model, nil on Create.
type ConnectorConfigValidator interface {
	ValidateConfiguration(model any, prior any) diag.Diagnostics
}

// ConnectorConfigRequestFilter lets a connector adjust the config it sends.
// state is nil on Create and the prior state model on Update.
type ConnectorConfigRequestFilter interface {
	FilterRequestConfig(configMap map[string]any, plan any, state any)
}

// ConnectorConfigPlanAdjuster names attributes whose planned value must be
// null or unknown, given the configuration and the prior state (nil on Create).
type ConnectorConfigPlanAdjuster interface {
	PlanAdjustments(config any, state any) (unset []string, unknown []string)
}

// ConnectorConfigResponseDecoder rewrites a response config into the shapes the
// model reads, before it is mapped (API sources store lists as CSV strings).
type ConnectorConfigResponseDecoder interface {
	DecodeResponseConfig(config map[string]any)
}

// ConnectorConfigKeepsConfiguredForm names the attributes whose value the
// backend normalizes before echoing it (a date stored as a UTC datetime, an ID
// with its dashes removed, an empty list as unset). For those, the configured
// spelling is kept in state while the backend still echoes the normalized form
// recorded at apply time; any other echo is real drift and replaces it. Every
// other attribute takes its echo, so a value the backend dropped or rewrote
// shows as a change.
type ConnectorConfigKeepsConfiguredForm interface {
	ConfiguredFormFields() []string
}

// configuredFormKey is the private-state key holding, per attribute, the
// normalized echo recorded when the configured spelling was kept.
const configuredFormKey = "configured_form_echo"

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
	// A config may cache its schema; the rewrites below must not reach it.
	baseSchema.Attributes = maps.Clone(baseSchema.Attributes)
	// Read refreshes connector_status before planning. Preserve that observed
	// value through the plan instead of replacing it with an unknown value.
	status := baseSchema.Attributes["connector_status"].(schema.StringAttribute)
	status.PlanModifiers = append(slices.Clip(status.PlanModifiers), stringplanmodifier.UseStateForUnknown())
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
	validator, validates := r.config.(ConnectorConfigValidator)
	adjuster, adjusts := r.config.(ConnectorConfigPlanAdjuster)
	var prior any
	if (validates || adjusts) && !req.State.Raw.IsNull() {
		prior = r.config.NewModelInstance()
		resp.Diagnostics.Append(req.State.Get(ctx, prior)...)
		if resp.Diagnostics.HasError() {
			return
		}
	}
	if validates {
		resp.Diagnostics.Append(validator.ValidateConfiguration(model, prior)...)
		if resp.Diagnostics.HasError() {
			return
		}
	}
	s := r.config.GetSchema()
	if adjusts {
		plan := func(name string, value attr.Value, err error) {
			if err != nil {
				resp.Diagnostics.AddError("Error planning "+r.config.GetResourceName(), fmt.Sprintf("Cannot plan %s: %s.", name, err))
				return
			}
			resp.Diagnostics.Append(resp.Plan.SetAttribute(ctx, path.Root(name), value)...)
		}
		unset, unknown := adjuster.PlanAdjustments(model, prior)
		for _, name := range unset {
			value, err := nullValue(ctx, s.Attributes[name])
			plan(name, value, err)
		}
		for _, name := range unknown {
			value, err := unknownValue(ctx, s.Attributes[name])
			plan(name, value, err)
		}
		if resp.Diagnostics.HasError() {
			return
		}
	}
	mappings := r.config.GetFieldMappings()
	names := make([]string, 0, len(mappings))
	for name := range mappings {
		names = append(names, name)
	}
	configured := shared.CaptureFields(model, names)
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
	s := r.config.GetSchema()
	secrets := shared.SensitiveStringAttrNames(s)
	plannedSecrets := shared.CaptureFields(model, secrets)
	ctx = withSentSecrets(ctx, plannedSecrets)
	plannedDefaults := shared.CaptureFields(model, shared.DefaultedAttrNames(s))
	plannedForms := shared.CaptureFields(model, r.configuredFormAttrNames())

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

	if filter, ok := r.config.(ConnectorConfigRequestFilter); ok {
		filter.FilterRequestConfig(configMap, model, nil)
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
	var responseClusterID string
	var responseConfig map[string]any

	kcClusterId := r.getStringField(model, "KcClusterId")
	tags := r.getStringSliceField(ctx, model, "Tags")
	var responseTags []string
	var receipt *api.SMTChainReceipt
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
			r.addConnectorWriteError(&resp.Diagnostics, "creating", "create", err, chain, embedded != nil, priorRevision)
			return
		}
		receipt = source.SMTChainReceipt
		chainSent = embedded != nil
		id = source.ID
		connectorName = source.Name
		connectorCode = source.Connector
		connectorStatus = source.ConnectorStatus
		responseClusterID = source.KcClusterId
		responseConfig = source.Config
		responseTags = source.Tags
		addConnectionWarning(ctx, &resp.Diagnostics, source)

	case ConnectorTypeDestination:
		embedded, ok := r.embeddedChain(chain, priorRevision, &resp.Diagnostics)
		if !ok {
			return
		}
		destination, err := r.client.CreateDestination(ctx, api.Destination{
			Name:        name,
			Connector:   r.config.GetConnectorCode(),
			Config:      configMap,
			KcClusterId: kcClusterId,
			Tags:        tags,
			SMTChain:    embedded,
		})
		if err != nil {
			r.addConnectorWriteError(&resp.Diagnostics, "creating", "create", err, chain, embedded != nil, priorRevision)
			return
		}
		receipt = destination.SMTChainReceipt
		chainSent = embedded != nil
		id = destination.ID
		connectorName = destination.Name
		connectorCode = destination.Connector
		connectorStatus = destination.ConnectorStatus
		responseClusterID = destination.KcClusterId
		responseConfig = destination.Config
		responseTags = destination.Tags
	}

	// Update model with response data
	r.setStringField(model, "ID", id)
	r.setStringField(model, "Name", connectorName)
	r.setStringField(model, "Connector", connectorCode)
	r.setStringField(model, "ConnectorStatus", connectorStatus)
	// A cluster the server selects is planned unknown; the response names it.
	if cluster := shared.CaptureFields(model, []string{"kc_cluster_id"})["kc_cluster_id"]; cluster != nil && cluster.IsUnknown() {
		r.setStringField(model, "KcClusterId", responseClusterID)
	}
	r.setStringSliceField(model, "Tags", normalizeTagsResponse(tags, responseTags))
	r.configMapToModel(ctx, responseConfig, model)
	shared.PreserveKnownFields(model, plannedSecrets)
	shared.FillNullFields(model, plannedDefaults)
	shared.PinFields(model, plannedFlat)
	if r.keepsConfiguredForm() {
		resp.Diagnostics.Append(resp.Private.SetKey(ctx, configuredFormKey, keepConfiguredForms(model, plannedForms))...)
	}

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, model)...)
	// The connector saved its chain in the same request; only the read-back remains.
	r.recordEmbeddedChain(ctx, id, chain, chainSent, receipt, priorChain, priorPredicates, priorOverrides, priorRevision, &resp.State, &resp.Diagnostics)
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
	s := r.config.GetSchema()
	secrets := shared.SensitiveStringAttrNames(s)
	priorSecrets := shared.CaptureFields(model, secrets)
	// Same for defaulted attributes the backend drops when their gating
	// condition is unmet: nulling them in state would plan the Default back
	// every run.
	priorDefaults := shared.CaptureFields(model, shared.DefaultedAttrNames(s))
	priorForms := shared.CaptureFields(model, r.configuredFormAttrNames())

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
	if r.keepsConfiguredForm() {
		recorded, diags := req.Private.GetKey(ctx, configuredFormKey)
		resp.Diagnostics.Append(diags...)
		kept, err := restoreConfiguredForms(model, priorForms, recorded)
		if err != nil {
			resp.Diagnostics.AddError(fmt.Sprintf("Error reading %s %s", r.config.GetConnectorType(), r.config.GetConnectorCode()), err.Error())
			return
		}
		resp.Diagnostics.Append(resp.Private.SetKey(ctx, configuredFormKey, kept)...)
	}

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
	s := r.config.GetSchema()
	secrets := shared.SensitiveStringAttrNames(s)
	plannedSecrets := shared.CaptureFields(model, secrets)
	ctx = withSentSecrets(ctx, plannedSecrets)
	plannedDefaults := shared.CaptureFields(model, shared.DefaultedAttrNames(s))
	plannedForms := shared.CaptureFields(model, r.configuredFormAttrNames())

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

	if filter, ok := r.config.(ConnectorConfigRequestFilter); ok {
		prior := r.config.NewModelInstance()
		resp.Diagnostics.Append(req.State.Get(ctx, prior)...)
		if resp.Diagnostics.HasError() {
			return
		}
		filter.FilterRequestConfig(configMap, model, prior)
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
	expectedRevision, ok := r.expectedRevision(ctx, id, chain, priorRevision, &resp.Diagnostics)
	if !ok {
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
	var receipt *api.SMTChainReceipt
	var chainSent bool

	switch r.config.GetConnectorType() {
	case ConnectorTypeSource:
		embedded, ok := r.embeddedChain(chain, expectedRevision, &resp.Diagnostics)
		if !ok {
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
			r.addConnectorWriteError(&resp.Diagnostics, "updating", "update", err, chain, embedded != nil, expectedRevision)
			return
		}
		receipt = source.SMTChainReceipt
		chainSent = embedded != nil
		connectorName = source.Name
		connectorCode = source.Connector
		connectorStatus = source.ConnectorStatus
		responseConfig = source.Config
		responseTags = source.Tags
		addConnectionWarning(ctx, &resp.Diagnostics, source)

	case ConnectorTypeDestination:
		embedded, ok := r.embeddedChain(chain, expectedRevision, &resp.Diagnostics)
		if !ok {
			return
		}
		destination, err := r.client.UpdateDestination(ctx, id, api.Destination{
			Name:        name,
			Connector:   r.config.GetConnectorCode(),
			Config:      configMap,
			KcClusterId: kcClusterId,
			Tags:        tags,
			SMTChain:    embedded,
		})
		if err != nil {
			r.addConnectorWriteError(&resp.Diagnostics, "updating", "update", err, chain, embedded != nil, expectedRevision)
			return
		}
		receipt = destination.SMTChainReceipt
		chainSent = embedded != nil
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
	if r.keepsConfiguredForm() {
		resp.Diagnostics.Append(resp.Private.SetKey(ctx, configuredFormKey, keepConfiguredForms(model, plannedForms))...)
	}

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
	r.recordEmbeddedChain(ctx, id, chain, chainSent, receipt, priorChain, priorPredicates, priorOverrides, priorRevision, &resp.State, &resp.Diagnostics)
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

// chainImportSuffix, appended to an import id, adopts the connector's
// transform chain.
const chainImportSuffix = "/" + smt.AttrChain

// ImportState imports an existing resource by ID, leaving the chain unmanaged
// as the released provider did, so a configuration without SMT attributes
// plans no change. An id ending in chainImportSuffix also writes the empty
// chain that marks the surface as managed, so the Read that follows adopts the
// server instances under keys derived from their ids.
func (r *BaseConnectorResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	id, adoptChain := strings.CutSuffix(req.ID, chainImportSuffix)
	if !adoptChain {
		resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
		return
	}
	if r.smt == nil {
		resp.Diagnostics.AddError("Transform chain import not supported",
			fmt.Sprintf("This resource has no %s to adopt. Import it by its id alone.", smt.AttrChain))
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), id)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root(smt.AttrChain), r.smt.ImportMarker())...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root(smt.AttrPredicates), r.smt.PredicateImportMarker())...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root(smt.AttrManagedOverrides), types.MapValueMust(types.BoolType, map[string]attr.Value{}))...)
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

// embeddedChain is the chain write a connector create or update carries,
// which the backend saves atomically with the connector body, so a refused
// chain leaves the connector settings unsaved too. It is nil when nothing is
// to be written, which omits the key and so keeps the stored chain. ok is
// false when the write cannot be built.
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

// expectedRevision is the revision a write to the existing connector is
// compared against; see smt.Surface.ExpectedRevision. ok is false when the
// write must not be sent.
func (r *BaseConnectorResource) expectedRevision(ctx context.Context, id string, write *smt.Write, priorRevision types.String, diags *diag.Diagnostics) (types.String, bool) {
	if r.smt == nil || write == nil {
		return priorRevision, true
	}
	client, ok := r.chainClient(diags)
	if !ok {
		return priorRevision, false
	}
	revision, d := r.smt.ExpectedRevision(ctx, client, r.chainKind(), id, write, priorRevision)
	diags.Append(d...)
	return revision, !d.HasError()
}

// addConnectorWriteError reports a refused connector create or update. A
// structured SMT envelope is located on the chain attributes; anything else
// is reported against the connector, since the same request carried the
// connector settings. The connector routes answer a body carrying a chain
// that Kafka Connect refuses with a plain 400, which saved nothing.
func (r *BaseConnectorResource) addConnectorWriteError(diags *diag.Diagnostics, verbing, verb string, err error, write *smt.Write, chainSent bool, expectedRevision types.String) {
	if summary, detail, ok := smt.ViewInvalid(err); ok {
		diags.AddError(summary, detail)
		return
	}
	if _, structured := api.SMTErrorOf(err); structured && r.smt != nil && write != nil && write.Configured {
		diags.Append(r.smt.WriteError(err, expectedRevision, write)...)
		return
	}
	kind := r.config.GetConnectorType()
	summary := fmt.Sprintf("Error %s %s %s", verbing, r.config.GetConnectorCode(), kind)
	var apiErr *api.APIError
	if chainSent && errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusBadRequest {
		diags.AddError(summary, fmt.Sprintf("The %s and its transform chain were refused, so nothing was saved: %s", kind, err))
		return
	}
	r.addSaveError(diags, summary, fmt.Sprintf("Unable to %s %s", verb, kind), err)
}

// recordEmbeddedChain records the chain of a connector whose create or update
// succeeded. When the request carried a chain, the chain is read back. The
// response names the revision it accepted and the read must be that revision;
// otherwise the read could belong to a later edit and its instances would be
// keyed by the wrong plan entries. A failed read keeps the prior chain and
// revision so state holds no unknown; the next refresh reads the real chain.
func (r *BaseConnectorResource) recordEmbeddedChain(ctx context.Context, id string, write *smt.Write, sent bool, receipt *api.SMTChainReceipt, priorChain, priorPredicates types.List, priorOverrides types.Map, priorRevision types.String, state *tfsdk.State, diags *diag.Diagnostics) {
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

func (r *BaseConnectorResource) readEmbeddedChain(ctx context.Context, id string, write *smt.Write, receipt *api.SMTChainReceipt, diags *diag.Diagnostics) (smt.GraphState, bool) {
	kind := r.config.GetConnectorType()
	if receipt == nil || receipt.ChainRevision == "" {
		diags.AddAttributeError(path.Root(smt.AttrChain), "Transform chain outcome unconfirmed",
			fmt.Sprintf("The %s was saved with its transform chain, but the response did not identify the accepted chain revision. Run terraform plan to refresh the chain before applying again.", kind))
		return smt.GraphState{}, false
	}
	client, ok := r.chainClient(diags)
	if !ok {
		return smt.GraphState{}, false
	}
	read, err := client.GetSMTChain(ctx, r.chainKind(), id)
	if summary, detail, ok := smt.ViewInvalid(err); ok {
		diags.AddAttributeError(path.Root(smt.AttrChain), summary, detail)
		return smt.GraphState{}, false
	}
	if err != nil {
		diags.AddAttributeError(path.Root(smt.AttrChain), "Unable to read the saved transform chain",
			fmt.Sprintf("The %s and its transform chain were saved, but reading the chain back failed. Run terraform plan to refresh it: %s", kind, err))
		return smt.GraphState{}, false
	}
	if read.Desired.ChainRevision != receipt.ChainRevision {
		diags.AddAttributeError(path.Root(smt.AttrChain), "Transform chain changed outside Terraform",
			fmt.Sprintf("The %s saved transform chain revision %q, but the chain now reads as revision %q. Run terraform plan to refresh it and review the drift before applying.", kind, receipt.ChainRevision, read.Desired.ChainRevision))
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
	var priorRevision types.String
	diags.Append(priorState.GetAttribute(ctx, path.Root(smt.AttrChain), &prior)...)
	diags.Append(priorState.GetAttribute(ctx, path.Root(smt.AttrPredicates), &priorPredicates)...)
	diags.Append(priorState.GetAttribute(ctx, path.Root(smt.AttrManagedOverrides), &priorOverrides)...)
	diags.Append(priorState.GetAttribute(ctx, path.Root(smt.AttrRevision), &priorRevision)...)
	if diags.HasError() {
		return
	}
	// Only ImportState leaves the empty chain marker without a revision.
	importing := !prior.IsNull() && len(prior.Elements()) == 0 && priorRevision.IsNull()
	if prior.IsNull() && priorOverrides.IsNull() {
		diags.Append(state.SetAttribute(ctx, path.Root(smt.AttrRevision), types.StringNull())...)
		return
	}
	client, ok := r.chainClient(diags)
	if !ok {
		return
	}
	graph, d := r.smt.RefreshGraph(ctx, client, r.chainKind(), id, prior, priorPredicates, priorOverrides, importing)
	diags.Append(d...)
	if diags.HasError() {
		return
	}
	// A chain the import could not adopt leaves the surface unmanaged, so the
	// flat attributes keep what the API returned.
	if !graph.Transforms.IsNull() || !graph.ManagedOverrides.IsNull() {
		for _, name := range r.smt.FlatSMT {
			var v attr.Value
			diags.Append(priorState.GetAttribute(ctx, path.Root(name), &v)...)
			diags.Append(state.SetAttribute(ctx, path.Root(name), v)...)
		}
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
	if decoder, ok := r.config.(ConnectorConfigResponseDecoder); ok {
		decoder.DecodeResponseConfig(cfg)
	}
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

func (r *BaseConnectorResource) keepsConfiguredForm() bool {
	_, ok := r.config.(ConnectorConfigKeepsConfiguredForm)
	return ok
}

// configuredFormAttrNames lists the attributes whose configured spelling is
// kept over a normalized echo.
func (r *BaseConnectorResource) configuredFormAttrNames() []string {
	if keeps, ok := r.config.(ConnectorConfigKeepsConfiguredForm); ok {
		return keeps.ConfiguredFormFields()
	}
	return nil
}

// keepConfiguredForms restores each planned value the backend echoed in a
// different, normalized form (null included: an empty value it drops), and
// returns those echoes as private state.
func keepConfiguredForms(model any, planned map[string]attr.Value) []byte {
	echoed := shared.CaptureFields(model, slices.Collect(maps.Keys(planned)))
	recorded := map[string]string{}
	kept := map[string]attr.Value{}
	for name, want := range planned {
		got := echoed[name]
		if want == nil || got == nil || want.IsNull() || want.IsUnknown() || got.IsUnknown() || want.Equal(got) {
			continue
		}
		recorded[name] = got.String()
		kept[name] = want
	}
	shared.PreserveKnownFields(model, kept)
	return encodeConfiguredForms(recorded)
}

// restoreConfiguredForms keeps the prior configured spelling while the API
// still echoes the normalized value recorded for it, and forgets a record the
// API no longer matches.
func restoreConfiguredForms(model any, prior map[string]attr.Value, private []byte) ([]byte, error) {
	recorded := map[string]string{}
	if len(private) > 0 {
		if err := json.Unmarshal(private, &recorded); err != nil {
			return nil, fmt.Errorf("restoreConfiguredForms: private state %q is not a JSON object: %w", configuredFormKey, err)
		}
	}
	echoed := shared.CaptureFields(model, slices.Collect(maps.Keys(recorded)))
	kept := map[string]attr.Value{}
	for name, echo := range recorded {
		if got := echoed[name]; got == nil || got.String() != echo || prior[name] == nil {
			delete(recorded, name)
			continue
		}
		kept[name] = prior[name]
	}
	shared.PreserveKnownFields(model, kept)
	return encodeConfiguredForms(recorded), nil
}

// encodeConfiguredForms returns nil for no records, which removes the key.
func encodeConfiguredForms(recorded map[string]string) []byte {
	if len(recorded) == 0 {
		return nil
	}
	data, _ := json.Marshal(recorded)
	return data
}

// addSaveError puts each message of a refused save on the attribute it names,
// when the backend names a mapped field for every one; otherwise it reports the
// whole error. A field a deprecated alias shares goes to the canonical
// attribute, and each message keeps the full error for its request ID.
func (r *BaseConnectorResource) addSaveError(diags *diag.Diagnostics, summary, prefix string, err error) {
	var apiErr *api.APIError
	if errors.As(err, &apiErr) && len(apiErr.Fields) > 0 {
		mappings := r.config.GetFieldMappings()
		attributes := r.config.GetSchema().Attributes
		deprecated := func(name string) bool {
			return attributes[name] != nil && attributes[name].GetDeprecationMessage() != ""
		}
		owners := map[string]string{}
		for _, name := range slices.Sorted(maps.Keys(mappings)) {
			if owner, taken := owners[mappings[name]]; !taken || (deprecated(owner) && !deprecated(name)) {
				owners[mappings[name]] = name
			}
		}
		var attributed diag.Diagnostics
		for _, field := range apiErr.Fields {
			name, ok := owners[field.Field]
			if !ok {
				attributed = nil
				break
			}
			attributed.AddAttributeError(path.Root(name), summary, fmt.Sprintf("%s\n\n%s: %s", field.Message, prefix, err))
		}
		if attributed != nil {
			diags.Append(attributed...)
			return
		}
	}
	diags.AddError(summary, fmt.Sprintf("%s: %s", prefix, err))
}

func addConnectionWarning(ctx context.Context, diags *diag.Diagnostics, source *api.Source) {
	if source.ConnectionWarning != "" {
		diags.AddWarning("Source saved with a connection warning", api.MaskSentSecrets(ctx, source.ConnectionWarning))
	}
}

// unknownValue is the typed unknown for a top-level attribute.
func unknownValue(ctx context.Context, attribute schema.Attribute) (attr.Value, error) {
	return typedValue(ctx, attribute, tftypes.UnknownValue)
}

// nullValue is the typed null for a top-level attribute.
func nullValue(ctx context.Context, attribute schema.Attribute) (attr.Value, error) {
	return typedValue(ctx, attribute, nil)
}

func typedValue(ctx context.Context, attribute schema.Attribute, raw any) (attr.Value, error) {
	if attribute == nil {
		return nil, errors.New("the attribute is not in the schema")
	}
	t := attribute.GetType()
	return t.ValueFromTerraform(ctx, tftypes.NewValue(t.TerraformType(ctx), raw))
}

// withSentSecrets names the plan's Sensitive values on ctx, so an error that
// quotes one back masks it whatever its API field is called.
func withSentSecrets(ctx context.Context, secrets map[string]attr.Value) context.Context {
	var values []string
	for _, value := range secrets {
		if s, ok := value.(types.String); ok && !s.IsNull() && !s.IsUnknown() {
			values = append(values, s.ValueString())
		}
	}
	return api.WithSentSecrets(ctx, values...)
}
