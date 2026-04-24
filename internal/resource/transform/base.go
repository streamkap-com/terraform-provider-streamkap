// Package transform provides a generic base resource for Streamkap transforms.
// It implements the Terraform Resource interface and delegates transform-specific
// behavior to a TransformConfig interface.
package transform

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/streamkap-com/terraform-provider-streamkap/internal/api"
	"github.com/streamkap-com/terraform-provider-streamkap/internal/constants"
	"github.com/streamkap-com/terraform-provider-streamkap/internal/helper"
	"github.com/streamkap-com/terraform-provider-streamkap/internal/resource/shared"
)

// TransformConfig is the interface that each transform must implement to provide
// its schema, field mappings, and metadata to the base resource.
type TransformConfig interface {
	// GetSchema returns the Terraform schema for this transform.
	GetSchema() schema.Schema

	// GetFieldMappings returns a map from Terraform attribute names to API field names.
	// Example: "transforms_language" -> "transforms.language"
	GetFieldMappings() map[string]string

	// GetTransformType returns the transform type code (e.g., "map_filter", "enrich").
	GetTransformType() string

	// GetResourceName returns the full resource name (e.g., "transform_map_filter").
	GetResourceName() string

	// NewModelInstance creates a new instance of the transform's model struct.
	// This is needed for reflection-based operations.
	NewModelInstance() any

	// SupportsPreviewDeploy returns whether this transform supports preview deployment.
	// Flink-based transforms support preview; KC-based transforms (topic_router) do not.
	SupportsPreviewDeploy() bool
}

// Ensure BaseTransformResource satisfies framework interfaces.
var (
	_ resource.Resource                = &BaseTransformResource{}
	_ resource.ResourceWithConfigure   = &BaseTransformResource{}
	_ resource.ResourceWithImportState = &BaseTransformResource{}
)

// BaseTransformResource is a generic resource implementation for transforms.
// It implements the standard Terraform Resource interface and uses TransformConfig
// for transform-specific behavior.
type BaseTransformResource struct {
	client api.StreamkapAPI
	config TransformConfig
}

// NewBaseTransformResource creates a new BaseTransformResource with the given config.
func NewBaseTransformResource(config TransformConfig) resource.Resource {
	return &BaseTransformResource{
		config: config,
	}
}

// Metadata returns the resource type name.
func (r *BaseTransformResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_" + r.config.GetResourceName()
}

// Schema returns the schema for this resource.
func (r *BaseTransformResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	baseSchema := r.config.GetSchema()

	// Add implementation_json attribute for managing transform implementation code
	baseSchema.Attributes["implementation_json"] = schema.StringAttribute{
		CustomType:          jsontypes.NormalizedType{},
		Optional:            true,
		Computed:            true,
		Description:         "Transform implementation as JSON. Structure varies by transform type (map_filter, enrich, sql_join, rollup, etc.).",
		MarkdownDescription: "Transform implementation as JSON. Structure varies by transform type (`map_filter`, `enrich`, `sql_join`, `rollup`, etc.).\n\n**Note:** If not specified, the implementation is managed outside Terraform (e.g., via Streamkap UI).",
		PlanModifiers: []planmodifier.String{
			stringplanmodifier.UseStateForUnknown(),
		},
	}

	// Add deploy attribute for auto-deploying transforms to Flink
	baseSchema.Attributes["deploy"] = schema.BoolAttribute{
		Description:         "Whether to automatically deploy the transform after creation or update. When true, the transform will be deployed to Flink after apply. Defaults to false.",
		MarkdownDescription: "Whether to automatically deploy the transform after creation or update. When `true`, the transform will be deployed to Flink after apply. Defaults to `false`.",
		Optional:            true,
		Computed:            true,
		Default:             booldefault.StaticBool(false),
	}

	// Add replay_window attribute for deployment replay configuration. The
	// backend never returns this field on Read (it's a deployment-time action
	// param), so we mark it Optional+Computed+UseStateForUnknown to keep the
	// plan stable — this matches the generator-wide rule from issue #80 and
	// prevents "planned value was X, but now null" on refresh.
	baseSchema.Attributes["replay_window"] = schema.StringAttribute{
		Description:         "Replay window for deployment. Specifies how much historical data to reprocess on deploy. Valid values: \"7d\", \"3d\", \"24h\", \"10m\", \"0\" (continue from last position). Only used when deploy is true.",
		MarkdownDescription: "Replay window for deployment. Specifies how much historical data to reprocess on deploy.\n\nValid values: `7d`, `3d`, `24h`, `10m`, `0` (continue from last position). Only used when `deploy` is `true`.",
		Optional:            true,
		Computed:            true,
		PlanModifiers: []planmodifier.String{
			stringplanmodifier.UseStateForUnknown(),
		},
		Validators: []validator.String{
			stringvalidator.OneOf("7d", "3d", "24h", "10m", "0"),
		},
	}

	// Add connector_status attribute for reading the current Flink job status
	baseSchema.Attributes["connector_status"] = schema.StringAttribute{
		Description:         "Current deployment status of the transform. Read-only. Values: RUNNING, INITIALIZING, DEPLOYING, STOPPED, FAILED, UNKNOWN.",
		MarkdownDescription: "Current deployment status of the transform. **Read-only**.\n\nValues: `RUNNING`, `INITIALIZING`, `DEPLOYING`, `STOPPED`, `FAILED`, `UNKNOWN`.",
		Computed:            true,
		PlanModifiers: []planmodifier.String{
			stringplanmodifier.UseStateForUnknown(),
		},
	}

	// Add timeouts block to the schema
	baseSchema.Blocks = map[string]schema.Block{
		"timeouts": timeouts.Block(ctx, timeouts.Opts{
			Create: true,
			Update: true,
			Delete: true,
		}),
	}

	resp.Schema = baseSchema
}

// Configure sets the API client for this resource.
func (r *BaseTransformResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// Create creates a new transform resource.
func (r *BaseTransformResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
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

	// Create a new model instance for this transform
	model := r.config.NewModelInstance()

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, model)...)
	if resp.Diagnostics.HasError() {
		return
	}

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
			fmt.Sprintf("Error creating %s transform", r.config.GetTransformType()),
			fmt.Sprintf("Unable to marshal configuration: %s", err),
		)
		return
	}

	// Add the name to the config map as transforms expect it there
	configMap["transforms.name"] = name

	tflog.Debug(ctx, fmt.Sprintf("Creating %s transform with config: %+v", r.config.GetTransformType(), configMap))

	// Call the Transform API
	transform, err := r.client.CreateTransform(ctx, api.CreateTransformRequest{
		Transform: r.config.GetTransformType(),
		Config:    configMap,
	})
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Error creating %s transform", r.config.GetTransformType()),
			fmt.Sprintf("Unable to create transform: %s", err),
		)
		return
	}

	// Update model with response data
	r.setStringField(model, "ID", transform.ID)
	r.setStringField(model, "Name", transform.Name)
	r.setStringField(model, "TransformType", transform.TransformType)
	r.setStringField(model, "ConnectorStatus", constants.JobStatusUnknown)
	r.configMapToModel(ctx, transform.Config, model)

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Handle implementation_json if provided in the plan
	var implementationJSON jsontypes.Normalized
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("implementation_json"), &implementationJSON)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if !implementationJSON.IsNull() && !implementationJSON.IsUnknown() {
		// User provided implementation - update it via the implementation_details API
		var implMap map[string]any
		if err := json.Unmarshal([]byte(implementationJSON.ValueString()), &implMap); err != nil {
			resp.Diagnostics.AddError(
				"Invalid Implementation JSON",
				fmt.Sprintf("Unable to parse implementation_json: %s", err),
			)
			return
		}

		_, err := r.client.UpdateTransformImplementationDetails(ctx, transform.ID, api.TransformImplementationDetails{
			TransformID:    transform.ID,
			Implementation: implMap,
		})
		if err != nil {
			resp.Diagnostics.AddError(
				fmt.Sprintf("Error setting %s transform implementation", r.config.GetTransformType()),
				fmt.Sprintf("Unable to update implementation: %s", err),
			)
			return
		}

		// Set the user's implementation_json in state
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("implementation_json"), implementationJSON)...)
	} else {
		// No implementation provided - read from API response and set as computed
		if len(transform.Implementation) > 0 {
			implJSON, err := marshalImplementation(transform.Implementation)
			if err != nil {
				tflog.Warn(ctx, fmt.Sprintf("Failed to marshal implementation to JSON: %s", err))
				resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("implementation_json"), jsontypes.NewNormalizedNull())...)
			} else {
				resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("implementation_json"), jsontypes.NewNormalizedValue(implJSON))...)
			}
		} else {
			resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("implementation_json"), jsontypes.NewNormalizedNull())...)
		}
	}

	// Deploy if requested and read connector_status
	r.deployFromPlan(ctx, transform.ID, req.Plan, &resp.Diagnostics, &resp.State)
}

// Read reads the transform resource.
func (r *BaseTransformResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError(
			"Unconfigured Resource",
			"Expected configured API client. Please report this issue to the provider developers.",
		)
		return
	}

	// Create a new model instance for this transform
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
			"The resource ID is missing from state.",
		)
		return
	}

	tflog.Debug(ctx, fmt.Sprintf("Reading %s transform with ID: %s", r.config.GetTransformType(), id))

	// Call the Transform API
	transform, err := r.client.GetTransform(ctx, id)
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Error reading %s transform", r.config.GetTransformType()),
			fmt.Sprintf("Unable to read transform: %s", err),
		)
		return
	}
	if transform == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	// Update model with response data
	r.setStringField(model, "Name", transform.Name)
	r.setStringField(model, "TransformType", transform.TransformType)
	r.configMapToModel(ctx, transform.Config, model)

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Read implementation_json from API and set in state.
	// The API may add server-side defaults (e.g., sourceIdleTimeoutMs, stateTTLMs for rollup).
	// If the user's config is a subset of the API response, preserve the user's value to avoid
	// spurious diffs from these defaults.
	var currentImplJSON jsontypes.Normalized
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("implementation_json"), &currentImplJSON)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if len(transform.Implementation) > 0 {
		if !currentImplJSON.IsNull() && !currentImplJSON.IsUnknown() {
			// User has set implementation_json - check if API just added defaults
			if isImplementationSubset(currentImplJSON.ValueString(), transform.Implementation) {
				// State is a subset of API response (API added server-side defaults) - preserve user's value
				resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("implementation_json"), currentImplJSON)...)
			} else {
				// Real drift detected - update state with API value
				implJSON, err := marshalImplementation(transform.Implementation)
				if err != nil {
					tflog.Warn(ctx, fmt.Sprintf("Failed to marshal implementation to JSON: %s", err))
					resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("implementation_json"), jsontypes.NewNormalizedNull())...)
				} else {
					resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("implementation_json"), jsontypes.NewNormalizedValue(implJSON))...)
				}
			}
		} else {
			// No user implementation in state - set from API
			implJSON, err := marshalImplementation(transform.Implementation)
			if err != nil {
				tflog.Warn(ctx, fmt.Sprintf("Failed to marshal implementation to JSON: %s", err))
				resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("implementation_json"), jsontypes.NewNormalizedNull())...)
			} else {
				resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("implementation_json"), jsontypes.NewNormalizedValue(implJSON))...)
			}
		}
	} else {
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("implementation_json"), jsontypes.NewNormalizedNull())...)
	}

	// Read connector_status.
	//
	// Three incoming cases via req.State.connector_status:
	//   - null           → import path (no prior state). Must call the API or
	//                      the imported resource will show UNKNOWN even when
	//                      the transform is RUNNING, causing a spurious diff
	//                      on the first post-import plan.
	//   - "UNKNOWN"      → never deployed or deploy skipped. Skip the API
	//                      call (cheap refresh for the common case).
	//   - known non-null → refresh from the API; on API error, preserve the
	//                      prior value rather than silently demoting to
	//                      UNKNOWN, so a transient 5xx or network blip does
	//                      not rewrite a confirmed RUNNING transform in
	//                      state (users rely on this field for automation
	//                      gates).
	//
	// The backend's /job_status endpoint rewrites HTTPException(404) into 400
	// through a broad except-clause (see app/api/transforms_api.py::
	// get_jobs_status), so we cannot rely on the HTTP status to tell
	// "transform truly not deployed" apart from "transient failure".
	// We fall back to matching the detail string: "Job not found" /
	// "Transform not found" are authoritative "not deployed" signals and
	// should demote prior state to UNKNOWN. Any other error is treated as
	// transient — preserve the prior value so a 5xx or network blip does
	// not silently rewrite a confirmed RUNNING transform in state.
	var currentStatus types.String
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("connector_status"), &currentStatus)...)

	priorKnown := !currentStatus.IsNull() && currentStatus.ValueString() != constants.JobStatusUnknown
	shouldRefresh := currentStatus.IsNull() || priorKnown

	if !shouldRefresh {
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("connector_status"), types.StringValue(constants.JobStatusUnknown))...)
		return
	}

	status, err := r.client.GetTransformJobStatus(ctx, id)
	switch {
	case err == nil && status != nil:
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("connector_status"), types.StringValue(status.Status))...)
	case err != nil && isJobNotDeployedError(err):
		// Authoritative "no deployment exists" — demote prior state so the
		// Terraform plan surfaces the drift on the next apply.
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("connector_status"), types.StringValue(constants.JobStatusUnknown))...)
	case priorKnown:
		tflog.Warn(ctx, fmt.Sprintf("Failed to refresh job status for transform %s; preserving prior value %q: %v", id, currentStatus.ValueString(), err))
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("connector_status"), currentStatus)...)
	default:
		// Import path (currentStatus was null) and the API call failed —
		// fall back to UNKNOWN so the attribute is populated.
		if err != nil {
			tflog.Warn(ctx, fmt.Sprintf("Failed to read job status on import for transform %s; defaulting to UNKNOWN: %v", id, err))
		}
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("connector_status"), types.StringValue(constants.JobStatusUnknown))...)
	}
}

// isJobNotDeployedError returns true when the backend's /job_status response
// means "no deployment exists" as opposed to a transient failure. The backend
// rewrites its 404 into a 400 via a broad except-clause, so we match on the
// detail string the handler emits for both transform-level and job-level
// misses (see app/api/transforms_api.py::get_jobs_status).
func isJobNotDeployedError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "Job not found") || strings.Contains(msg, "Transform not found")
}

// Update updates the transform resource.
func (r *BaseTransformResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
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

	// Create a new model instance for this transform
	model := r.config.NewModelInstance()

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, model)...)
	if resp.Diagnostics.HasError() {
		return
	}

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

	// Fetch existing transform to get implementation data
	// This is required because the backend overwrites implementation on update
	existingTransform, err := r.client.GetTransform(ctx, id)
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Error reading %s transform for update", r.config.GetTransformType()),
			fmt.Sprintf("Unable to read existing transform: %s", err),
		)
		return
	}
	if existingTransform == nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Error reading %s transform for update", r.config.GetTransformType()),
			fmt.Sprintf("Transform %s does not exist", id),
		)
		return
	}

	// Convert model to config map using field mappings
	configMap, err := r.modelToConfigMap(ctx, model)
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Error updating %s transform", r.config.GetTransformType()),
			fmt.Sprintf("Unable to marshal configuration: %s", err),
		)
		return
	}

	// Add the name to the config map as transforms expect it there
	configMap["transforms.name"] = name

	tflog.Debug(ctx, fmt.Sprintf("Updating %s transform with ID: %s, config: %+v", r.config.GetTransformType(), id, configMap))

	// Call the Transform API with existing implementation to prevent it from being overwritten
	transform, err := r.client.UpdateTransform(ctx, id, api.UpdateTransformRequest{
		Transform:      r.config.GetTransformType(),
		Config:         configMap,
		Implementation: existingTransform.Implementation,
	})
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Error updating %s transform", r.config.GetTransformType()),
			fmt.Sprintf("Unable to update transform: %s", err),
		)
		return
	}

	// Update model with response data
	r.setStringField(model, "Name", transform.Name)
	r.setStringField(model, "TransformType", transform.TransformType)
	// req.Plan.Get() above populated model.ConnectorStatus with the plan's
	// Unknown placeholder. Overwrite to a Known value before saving state —
	// deployFromPlan will refine it below if the transform is deployed. Without
	// this, state.connector_status ends Unknown, tripping Terraform's
	// "All values must be known after apply" consistency check (issue #71).
	r.setStringField(model, "ConnectorStatus", constants.JobStatusUnknown)
	r.configMapToModel(ctx, transform.Config, model)

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Handle implementation_json update
	var planImplementationJSON jsontypes.Normalized
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("implementation_json"), &planImplementationJSON)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var stateImplementationJSON jsontypes.Normalized
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("implementation_json"), &stateImplementationJSON)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Check if implementation_json has changed or is being set
	if !planImplementationJSON.IsNull() && !planImplementationJSON.IsUnknown() {
		// Check if it's different from state
		implementationChanged := stateImplementationJSON.IsNull() ||
			stateImplementationJSON.IsUnknown() ||
			planImplementationJSON.ValueString() != stateImplementationJSON.ValueString()

		if implementationChanged {
			// Parse the new implementation JSON
			var implMap map[string]any
			if err := json.Unmarshal([]byte(planImplementationJSON.ValueString()), &implMap); err != nil {
				resp.Diagnostics.AddError(
					"Invalid Implementation JSON",
					fmt.Sprintf("Unable to parse implementation_json: %s", err),
				)
				return
			}

			// Update implementation via the implementation_details API
			_, err := r.client.UpdateTransformImplementationDetails(ctx, id, api.TransformImplementationDetails{
				TransformID:    id,
				Implementation: implMap,
			})
			if err != nil {
				resp.Diagnostics.AddError(
					fmt.Sprintf("Error updating %s transform implementation", r.config.GetTransformType()),
					fmt.Sprintf("Unable to update implementation: %s", err),
				)
				return
			}
		}

		// Set the planned implementation_json in state
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("implementation_json"), planImplementationJSON)...)
	} else {
		// No implementation in plan - read from API response
		if len(transform.Implementation) > 0 {
			implJSON, err := marshalImplementation(transform.Implementation)
			if err != nil {
				tflog.Warn(ctx, fmt.Sprintf("Failed to marshal implementation to JSON: %s", err))
				resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("implementation_json"), jsontypes.NewNormalizedNull())...)
			} else {
				resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("implementation_json"), jsontypes.NewNormalizedValue(implJSON))...)
			}
		} else {
			resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("implementation_json"), jsontypes.NewNormalizedNull())...)
		}
	}

	// Deploy if requested and read connector_status
	r.deployFromPlan(ctx, id, req.Plan, &resp.Diagnostics, &resp.State)
}

// Delete deletes the transform resource.
func (r *BaseTransformResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError(
			"Unconfigured Resource",
			"Expected configured API client. Please report this issue to the provider developers.",
		)
		return
	}

	// Get timeout from state (not config, since config may not be available during delete)
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

	// Create a new model instance for this transform
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

	tflog.Debug(ctx, fmt.Sprintf("Deleting %s transform with ID: %s", r.config.GetTransformType(), id))

	// Call the Transform API
	err := r.client.DeleteTransform(ctx, id)
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Error deleting %s transform", r.config.GetTransformType()),
			fmt.Sprintf("Unable to delete transform: %s", err),
		)
		return
	}
}

// ImportState imports an existing resource by ID.
func (r *BaseTransformResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// deployFromPlan reads deploy/replay_window from the plan and handles deployment + status.
// Used by both Create and Update to avoid duplication.
func (r *BaseTransformResource) deployFromPlan(ctx context.Context, transformID string, plan tfsdk.Plan, diagnostics *diag.Diagnostics, state *tfsdk.State) {
	// Set initial connector_status so state is complete even if deploy fails
	diagnostics.Append(state.SetAttribute(ctx, path.Root("connector_status"), types.StringValue(constants.JobStatusUnknown))...)
	if diagnostics.HasError() {
		return
	}

	var deploy types.Bool
	diagnostics.Append(plan.GetAttribute(ctx, path.Root("deploy"), &deploy)...)
	if diagnostics.HasError() {
		return
	}

	if !deploy.IsNull() && deploy.ValueBool() {
		var replayWindow types.String
		diagnostics.Append(plan.GetAttribute(ctx, path.Root("replay_window"), &replayWindow)...)
		if diagnostics.HasError() {
			return
		}

		replayWindowValue := ""
		if !replayWindow.IsNull() {
			replayWindowValue = replayWindow.ValueString()
		}

		// Deploy preview first for Flink-based transforms (matches frontend flow)
		if r.config.SupportsPreviewDeploy() {
			err := r.client.DeployTransformPreview(ctx, transformID, constants.DeployVersionLatest, replayWindowValue)
			if err != nil {
				diagnostics.AddError("Error deploying transform preview",
					fmt.Sprintf("Transform saved successfully but preview deployment failed: %s", err))
				return
			}
		}

		// Deploy live
		err := r.client.DeployTransformLive(ctx, transformID, constants.DeployVersionLatest)
		if err != nil {
			diagnostics.AddError("Error deploying transform live",
				fmt.Sprintf("Transform saved successfully but live deployment failed: %s", err))
			return
		}

		// Poll for RUNNING status — returns final status to avoid redundant API call
		finalStatus, err := r.waitForDeployment(ctx, transformID)
		if err != nil {
			diagnostics.AddWarning("Transform deployed but not yet active",
				fmt.Sprintf("Deployment initiated but status check timed out: %s", err))
		}
		if finalStatus != nil {
			diagnostics.Append(state.SetAttribute(ctx, path.Root("connector_status"), types.StringValue(finalStatus.Status))...)
			return
		}
	}

	// Read connector_status (for non-deploy path or if polling returned no status)
	status, err := r.client.GetTransformJobStatus(ctx, transformID)
	if err == nil && status != nil {
		diagnostics.Append(state.SetAttribute(ctx, path.Root("connector_status"), types.StringValue(status.Status))...)
	}
}

// waitForDeployment polls the transform job status until it reaches a terminal state.
// Returns the final status to avoid redundant API calls.
// Relies on the parent context's deadline for timeout enforcement.
func (r *BaseTransformResource) waitForDeployment(ctx context.Context, transformID string) (*api.TransformJobStatus, error) {
	tflog.Info(ctx, fmt.Sprintf("Waiting for transform deployment to complete (transform_id: %s)", transformID))
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("deployment timed out: %w", ctx.Err())
		case <-ticker.C:
			status, err := r.client.GetTransformJobStatus(ctx, transformID)
			if err != nil {
				tflog.Debug(ctx, fmt.Sprintf("Transient error polling job status: %s", err))
				continue
			}
			if status == nil {
				continue
			}
			tflog.Debug(ctx, fmt.Sprintf("Transform %s deployment status: %s", transformID, status.Status))
			switch status.Status {
			case constants.JobStatusRunning:
				return status, nil
			case constants.JobStatusFailed:
				return status, fmt.Errorf("transform deployment failed (Flink status: %s)", constants.JobStatusFailed)
			case constants.JobStatusCanceled, constants.JobStatusStopped:
				return status, fmt.Errorf("transform deployment stopped (status: %s)", status.Status)
			// INITIALIZING, DEPLOYING, CREATED, RESTARTING — keep polling
			}
		}
	}
}

// stripNullValues recursively removes null values from a map.
// This is needed because the API returns extra null fields in implementation JSON
// that weren't in the original request, causing spurious diffs.
func stripNullValues(m map[string]any) map[string]any {
	result := make(map[string]any, len(m))
	for k, v := range m {
		if v == nil {
			continue
		}
		if nested, ok := v.(map[string]any); ok {
			stripped := stripNullValues(nested)
			if len(stripped) > 0 {
				result[k] = stripped
			}
		} else {
			result[k] = v
		}
	}
	return result
}

// marshalImplementation converts an implementation map to a JSON string,
// stripping null values to prevent spurious diffs from API-added defaults.
func marshalImplementation(impl map[string]any) (string, error) {
	stripped := stripNullValues(impl)
	implBytes, err := json.Marshal(stripped)
	if err != nil {
		return "", err
	}
	return string(implBytes), nil
}

// isImplementationSubset checks if all fields in the state's implementation JSON exist
// with the same values in the API response. Returns true if the API response is a superset
// of the state (i.e., the API only added server-side defaults like sourceIdleTimeoutMs).
// This prevents spurious diffs when the backend adds default fields to user-provided implementations.
func isImplementationSubset(stateJSON string, apiImpl map[string]any) bool {
	var stateMap map[string]any
	if err := json.Unmarshal([]byte(stateJSON), &stateMap); err != nil {
		return false
	}
	return jsonMapIsSubset(stateMap, apiImpl)
}

// jsonMapIsSubset recursively checks that every key in subset exists in superset
// with an equivalent value. Extra keys in superset are allowed.
func jsonMapIsSubset(subset, superset map[string]any) bool {
	for k, subVal := range subset {
		superVal, exists := superset[k]
		if !exists {
			return false
		}
		if !jsonValuesEqual(subVal, superVal) {
			return false
		}
	}
	return true
}

// jsonValuesEqual compares two JSON-deserialized values for deep equality.
// It handles maps, slices, and scalar types.
func jsonValuesEqual(a, b any) bool {
	// Handle nil
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}

	// Handle maps recursively with subset logic
	aMap, aIsMap := a.(map[string]any)
	bMap, bIsMap := b.(map[string]any)
	if aIsMap && bIsMap {
		// For nested maps, require exact match (not subset)
		if len(aMap) != len(bMap) {
			return false
		}
		for k, av := range aMap {
			bv, exists := bMap[k]
			if !exists || !jsonValuesEqual(av, bv) {
				return false
			}
		}
		return true
	}

	// Handle slices
	aSlice, aIsSlice := a.([]any)
	bSlice, bIsSlice := b.([]any)
	if aIsSlice && bIsSlice {
		if len(aSlice) != len(bSlice) {
			return false
		}
		for i := range aSlice {
			if !jsonValuesEqual(aSlice[i], bSlice[i]) {
				return false
			}
		}
		return true
	}

	// Scalar comparison via JSON marshaling to normalize numeric types
	aJSON, errA := json.Marshal(a)
	bJSON, errB := json.Marshal(b)
	if errA != nil || errB != nil {
		return false
	}
	return string(aJSON) == string(bJSON)
}

// modelToConfigMap converts a model struct to a config map using the field mappings.
// It uses reflection to read values from the model and maps them to API field names.
func (r *BaseTransformResource) modelToConfigMap(ctx context.Context, model any) (map[string]any, error) {
	return shared.ModelToConfigMap(ctx, model, r.config.GetFieldMappings(), nil)
}

func (r *BaseTransformResource) configMapToModel(ctx context.Context, cfg map[string]any, model any) {
	shared.ConfigMapToModel(ctx, cfg, model, r.config.GetFieldMappings(), nil)
}

func (r *BaseTransformResource) getStringField(model any, fieldName string) string {
	return shared.GetStringField(model, fieldName)
}

func (r *BaseTransformResource) setStringField(model any, fieldName string, value string) {
	shared.SetStringField(model, fieldName, value)
}
