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
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/streamkap-com/terraform-provider-streamkap/internal/api"
	"github.com/streamkap-com/terraform-provider-streamkap/internal/helper"
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

// Ensure BaseConnectorResource satisfies framework interfaces.
var (
	_ resource.Resource                = &BaseConnectorResource{}
	_ resource.ResourceWithConfigure   = &BaseConnectorResource{}
	_ resource.ResourceWithImportState = &BaseConnectorResource{}
)

// BaseConnectorResource is a generic resource implementation for connectors.
// It implements the standard Terraform Resource interface and uses ConnectorConfig
// for connector-specific behavior.
type BaseConnectorResource struct {
	client api.StreamkapAPI
	config ConnectorConfig
}

// NewBaseConnectorResource creates a new BaseConnectorResource with the given config.
func NewBaseConnectorResource(config ConnectorConfig) resource.Resource {
	return &BaseConnectorResource{
		config: config,
	}
}

// Metadata returns the resource type name.
func (r *BaseConnectorResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_" + r.config.GetResourceName()
}

// Schema returns the schema for this resource.
func (r *BaseConnectorResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	baseSchema := r.config.GetSchema()

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

	tflog.Debug(ctx, fmt.Sprintf("Creating %s %s with config: %+v", r.config.GetConnectorType(), r.config.GetConnectorCode(), configMap))

	// Call the appropriate API based on connector type
	var id string
	var connectorName string
	var connectorCode string
	var responseConfig map[string]any

	switch r.config.GetConnectorType() {
	case ConnectorTypeSource:
		source, err := r.client.CreateSource(ctx, api.Source{
			Name:      name,
			Connector: r.config.GetConnectorCode(),
			Config:    configMap,
		})
		if err != nil {
			resp.Diagnostics.AddError(
				fmt.Sprintf("Error creating %s source", r.config.GetConnectorCode()),
				fmt.Sprintf("Unable to create source: %s", err),
			)
			return
		}
		id = source.ID
		connectorName = source.Name
		connectorCode = source.Connector
		responseConfig = source.Config

	case ConnectorTypeDestination:
		destination, err := r.client.CreateDestination(ctx, api.Destination{
			Name:      name,
			Connector: r.config.GetConnectorCode(),
			Config:    configMap,
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
		responseConfig = destination.Config
	}

	// Update model with response data
	r.setStringField(model, "ID", id)
	r.setStringField(model, "Name", connectorName)
	r.setStringField(model, "Connector", connectorCode)
	r.configMapToModel(ctx, responseConfig, model)

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, model)...)
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
			"The resource ID is missing from state.",
		)
		return
	}

	tflog.Debug(ctx, fmt.Sprintf("Reading %s %s with ID: %s", r.config.GetConnectorType(), r.config.GetConnectorCode(), id))

	// Call the appropriate API based on connector type
	var connectorName string
	var connectorCode string
	var responseConfig map[string]any

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
		responseConfig = source.Config

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
		responseConfig = destination.Config
	}

	// Update model with response data
	r.setStringField(model, "Name", connectorName)
	r.setStringField(model, "Connector", connectorCode)
	r.configMapToModel(ctx, responseConfig, model)

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, model)...)
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

	tflog.Debug(ctx, fmt.Sprintf("Updating %s %s with ID: %s, config: %+v", r.config.GetConnectorType(), r.config.GetConnectorCode(), id, configMap))

	// Call the appropriate API based on connector type
	var connectorName string
	var connectorCode string
	var responseConfig map[string]any

	switch r.config.GetConnectorType() {
	case ConnectorTypeSource:
		source, err := r.client.UpdateSource(ctx, id, api.Source{
			Name:      name,
			Connector: r.config.GetConnectorCode(),
			Config:    configMap,
		})
		if err != nil {
			resp.Diagnostics.AddError(
				fmt.Sprintf("Error updating %s source", r.config.GetConnectorCode()),
				fmt.Sprintf("Unable to update source: %s", err),
			)
			return
		}
		connectorName = source.Name
		connectorCode = source.Connector
		responseConfig = source.Config

	case ConnectorTypeDestination:
		destination, err := r.client.UpdateDestination(ctx, id, api.Destination{
			Name:      name,
			Connector: r.config.GetConnectorCode(),
			Config:    configMap,
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
		responseConfig = destination.Config
	}

	// Update model with response data
	r.setStringField(model, "Name", connectorName)
	r.setStringField(model, "Connector", connectorCode)
	r.configMapToModel(ctx, responseConfig, model)

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, model)...)
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

// ImportState imports an existing resource by ID.
func (r *BaseConnectorResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// modelToConfigMap converts a model struct to a config map using the field mappings.
// It uses reflection to read values from the model and maps them to API field names.
func (r *BaseConnectorResource) modelToConfigMap(ctx context.Context, model any) (map[string]any, error) {
	configMap := make(map[string]any)
	mappings := r.config.GetFieldMappings()

	// Check if the config specifies fields that should be JSON-stringified
	jsonStringFields := r.getJSONStringFields()

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
			// Check if this field should be serialized as a JSON string
			if r.isJSONStringField(tfAttr, jsonStringFields) {
				// If apiValue is a map, serialize it to JSON string
				if mapVal, isMap := apiValue.(map[string]map[string]any); isMap {
					jsonBytes, err := json.Marshal(mapVal)
					if err != nil {
						tflog.Warn(ctx, fmt.Sprintf("Failed to serialize %s to JSON: %s", tfAttr, err))
					} else {
						configMap[apiField] = string(jsonBytes)
						continue
					}
				}
			}
			configMap[apiField] = apiValue
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
// It uses reflection to set values on the model.
func (r *BaseConnectorResource) configMapToModel(ctx context.Context, cfg map[string]any, model any) {
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
func (r *BaseConnectorResource) extractTerraformValue(ctx context.Context, fieldValue reflect.Value) any {
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

	case jsontypes.Normalized:
		if v.IsNull() || v.IsUnknown() {
			return nil
		}
		return v.ValueString()

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
		// Check if it's a map type
		if fieldValue.Kind() == reflect.Map {
			mapType := fieldValue.Type()
			if mapType.Key().Kind() == reflect.String {
				if fieldValue.IsNil() || fieldValue.Len() == 0 {
					return nil
				}

				// Check if it's map[string]types.String
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
					if len(result) == 0 {
						return nil
					}
					return result
				}

				// Check if it's a map[string]struct (nested map type)
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
					if len(result) == 0 {
						return nil
					}
					return result
				}
			}
		}
		tflog.Warn(ctx, fmt.Sprintf("Unknown Terraform type: %T", v))
		return nil
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

// setTerraformValue sets a Terraform types value from a config map value.
func (r *BaseConnectorResource) setTerraformValue(ctx context.Context, cfg map[string]any, apiField string, fieldValue reflect.Value) {
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

	case reflect.TypeOf(jsontypes.Normalized{}):
		// JSON type - stored as string in API
		if val, ok := cfg[apiField]; ok && val != nil {
			if strVal, ok := val.(string); ok {
				fieldValue.Set(reflect.ValueOf(jsontypes.NewNormalizedValue(strVal)))
			} else {
				fieldValue.Set(reflect.ValueOf(jsontypes.NewNormalizedNull()))
			}
		} else {
			fieldValue.Set(reflect.ValueOf(jsontypes.NewNormalizedNull()))
		}

	case reflect.TypeOf(types.List{}):
		tfVal := helper.GetTfCfgListString(ctx, cfg, apiField)
		fieldValue.Set(reflect.ValueOf(tfVal))

	default:
		// Check if it's a map type
		if fieldType.Kind() == reflect.Map && fieldType.Key().Kind() == reflect.String {
			// Check if it's map[string]types.String
			if fieldType.Elem() == reflect.TypeOf(types.String{}) {
				tfVal := helper.GetTfCfgMapString(ctx, cfg, apiField)
				if tfVal != nil {
					fieldValue.Set(reflect.ValueOf(tfVal))
				} else {
					fieldValue.Set(reflect.Zero(fieldType))
				}
				return
			}

			// Check if it's a map[string]struct (nested map type)
			if fieldType.Elem().Kind() == reflect.Struct {
				r.setNestedMapValue(ctx, cfg, apiField, fieldValue)
				return
			}
		}
		tflog.Warn(ctx, fmt.Sprintf("Unknown field type for '%s': %s", apiField, fieldType))
	}
}

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

	if len(apiMap) == 0 {
		fieldValue.Set(reflect.Zero(fieldValue.Type()))
		return
	}

	// Create a new map of the correct type
	elemType := fieldValue.Type().Elem()
	newMap := reflect.MakeMap(fieldValue.Type())

	for key, itemValue := range apiMap {
		// Each item should be a map[string]any representing the struct fields
		itemMap, ok := itemValue.(map[string]any)
		if !ok {
			tflog.Debug(ctx, fmt.Sprintf("Expected map[string]any for nested item '%s', got %T", key, itemValue))
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

// getStringField gets a string value from a model field using reflection.
func (r *BaseConnectorResource) getStringField(model any, fieldName string) string {
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
func (r *BaseConnectorResource) setStringField(model any, fieldName string, value string) {
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
