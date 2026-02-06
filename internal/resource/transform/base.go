// Package transform provides a generic base resource for Streamkap transforms.
// It implements the Terraform Resource interface and delegates transform-specific
// behavior to a TransformConfig interface.
package transform

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/streamkap-com/terraform-provider-streamkap/internal/api"
	"github.com/streamkap-com/terraform-provider-streamkap/internal/helper"
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

	// Read implementation_json from API and set in state
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

// modelToConfigMap converts a model struct to a config map using the field mappings.
// It uses reflection to read values from the model and maps them to API field names.
func (r *BaseTransformResource) modelToConfigMap(ctx context.Context, model any) (map[string]any, error) {
	configMap := make(map[string]any)
	mappings := r.config.GetFieldMappings()

	// Get the reflect value of the model (need to dereference if it's a pointer)
	v := reflect.ValueOf(model)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	t := v.Type()

	// Build a map from tfsdk tag to field index for quick lookup
	tfsdkToField := make(map[string]int)
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		tag := field.Tag.Get("tfsdk")
		if tag != "" && tag != "-" {
			tfsdkToField[tag] = i
		}
	}

	// Iterate over field mappings and extract values
	for tfAttr, apiField := range mappings {
		fieldIdx, ok := tfsdkToField[tfAttr]
		if !ok {
			tflog.Warn(ctx, fmt.Sprintf("Field mapping for '%s' not found in model", tfAttr))
			continue
		}

		fieldValue := v.Field(fieldIdx)
		apiValue := r.extractTerraformValue(ctx, fieldValue)

		// Only include non-nil values in the config map
		if apiValue != nil {
			configMap[apiField] = apiValue
		}
	}

	return configMap, nil
}

// configMapToModel updates a model struct from a config map using the field mappings.
// It uses reflection to set values on the model.
func (r *BaseTransformResource) configMapToModel(ctx context.Context, cfg map[string]any, model any) {
	mappings := r.config.GetFieldMappings()

	// Get the reflect value of the model (need to dereference if it's a pointer)
	v := reflect.ValueOf(model)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	t := v.Type()

	// Build a map from tfsdk tag to field index for quick lookup
	tfsdkToField := make(map[string]int)
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		tag := field.Tag.Get("tfsdk")
		if tag != "" && tag != "-" {
			tfsdkToField[tag] = i
		}
	}

	// Iterate over field mappings and set values
	for tfAttr, apiField := range mappings {
		fieldIdx, ok := tfsdkToField[tfAttr]
		if !ok {
			continue
		}

		fieldValue := v.Field(fieldIdx)
		if !fieldValue.CanSet() {
			continue
		}

		// Get the Terraform value based on the field type
		r.setTerraformValue(ctx, cfg, apiField, fieldValue)
	}
}

// extractTerraformValue extracts the underlying value from a Terraform types value.
func (r *BaseTransformResource) extractTerraformValue(ctx context.Context, fieldValue reflect.Value) any {
	// Handle different Terraform types
	switch v := fieldValue.Interface().(type) {
	case types.String:
		if v.IsNull() || v.IsUnknown() {
			return nil
		}
		return v.ValueString()

	case types.Int64:
		if v.IsNull() || v.IsUnknown() {
			return nil
		}
		return v.ValueInt64()

	case types.Bool:
		if v.IsNull() || v.IsUnknown() {
			return nil
		}
		return v.ValueBool()

	case types.Float64:
		if v.IsNull() || v.IsUnknown() {
			return nil
		}
		return v.ValueFloat64()

	case types.List:
		if v.IsNull() || v.IsUnknown() {
			return nil
		}
		// Convert list to slice of strings
		var result []string
		for _, elem := range v.Elements() {
			if strVal, ok := elem.(types.String); ok {
				result = append(result, strVal.ValueString())
			}
		}
		return result

	default:
		tflog.Warn(ctx, fmt.Sprintf("Unknown Terraform type: %T", v))
		return nil
	}
}

// setTerraformValue sets a Terraform types value from a config map value.
func (r *BaseTransformResource) setTerraformValue(ctx context.Context, cfg map[string]any, apiField string, fieldValue reflect.Value) {
	// Get the field type to determine which helper to use
	fieldType := fieldValue.Type()

	switch fieldType {
	case reflect.TypeOf(types.String{}):
		tfVal := helper.GetTfCfgString(cfg, apiField)
		fieldValue.Set(reflect.ValueOf(tfVal))

	case reflect.TypeOf(types.Int64{}):
		tfVal := helper.GetTfCfgInt64(cfg, apiField)
		fieldValue.Set(reflect.ValueOf(tfVal))

	case reflect.TypeOf(types.Bool{}):
		tfVal := helper.GetTfCfgBool(cfg, apiField)
		fieldValue.Set(reflect.ValueOf(tfVal))

	case reflect.TypeOf(types.Float64{}):
		// Float64 helper not in current helper.go, handle inline
		if val, ok := cfg[apiField]; ok && val != nil {
			if floatVal, ok := val.(float64); ok {
				fieldValue.Set(reflect.ValueOf(types.Float64Value(floatVal)))
			} else {
				fieldValue.Set(reflect.ValueOf(types.Float64Null()))
			}
		} else {
			fieldValue.Set(reflect.ValueOf(types.Float64Null()))
		}

	case reflect.TypeOf(types.List{}):
		tfVal := helper.GetTfCfgListString(ctx, cfg, apiField)
		fieldValue.Set(reflect.ValueOf(tfVal))

	default:
		tflog.Warn(ctx, fmt.Sprintf("Unknown field type for '%s': %s", apiField, fieldType))
	}
}

// getStringField gets a string value from a model field using reflection.
func (r *BaseTransformResource) getStringField(model any, fieldName string) string {
	v := reflect.ValueOf(model)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	field := v.FieldByName(fieldName)
	if !field.IsValid() {
		return ""
	}

	if strVal, ok := field.Interface().(types.String); ok {
		return strVal.ValueString()
	}

	return ""
}

// setStringField sets a types.String value on a model field using reflection.
func (r *BaseTransformResource) setStringField(model any, fieldName string, value string) {
	v := reflect.ValueOf(model)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	field := v.FieldByName(fieldName)
	if !field.IsValid() || !field.CanSet() {
		return
	}

	field.Set(reflect.ValueOf(types.StringValue(value)))
}
