package source

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	res "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/streamkap-com/terraform-provider-streamkap/internal/api"
	"github.com/streamkap-com/terraform-provider-streamkap/internal/helper"
)

// Ensure provider defined types fully satisfy framework interfaces.
var (
	_ res.Resource                = &SourceMongoDBResource{}
	_ res.ResourceWithConfigure   = &SourceMongoDBResource{}
	_ res.ResourceWithImportState = &SourceMongoDBResource{}
)

func NewSourceMongoDBResource() res.Resource {
	return &SourceMongoDBResource{connector_code: "mongodb"}
}

// SourceMongoDBResource defines the resource implementation.
type SourceMongoDBResource struct {
	client         api.StreamkapAPI
	connector_code string
}

// SourceMongoDBResourceModel describes the resource data model.
type SourceMongoDBResourceModel struct {
	ID                                   types.String `tfsdk:"id"`
	Name                                 types.String `tfsdk:"name"`
	Connector                            types.String `tfsdk:"connector"`
	MongoDBConnectionString              types.String `tfsdk:"mongodb_connection_string"`
	ArrayEncoding                        types.String `tfsdk:"array_encoding"`
	NestedDocumentEncoding               types.String `tfsdk:"nested_document_encoding"`
	DatabaseIncludeList                  types.String `tfsdk:"database_include_list"`
	CollectionIncludeList                types.String `tfsdk:"collection_include_list"`
	SignalDataCollectionSchemaOrDatabase types.String `tfsdk:"signal_data_collection_schema_or_database"`
	SSHEnabled                           types.Bool   `tfsdk:"ssh_enabled"`
	SSHHost                              types.String `tfsdk:"ssh_host"`
	SSHPort                              types.String `tfsdk:"ssh_port"`
	SSHUser                              types.String `tfsdk:"ssh_user"`
	PredicatesIsTopicToEnrichPattern     types.String `tfsdk:"predicates_istopictoenrich_pattern"`
	InsertStaticKeyField1                   types.String `tfsdk:"insert_static_key_field_1"`
	InsertStaticKeyValue1                   types.String `tfsdk:"insert_static_key_value_1"`
	InsertStaticValueField1                 types.String `tfsdk:"insert_static_value_field_1"`
	InsertStaticValue1                      types.String `tfsdk:"insert_static_value_1"`
	InsertStaticKeyField2                   types.String `tfsdk:"insert_static_key_field_2"`
	InsertStaticKeyValue2                   types.String `tfsdk:"insert_static_key_value_2"`
	InsertStaticValueField2                 types.String `tfsdk:"insert_static_value_field_2"`
	InsertStaticValue2                      types.String `tfsdk:"insert_static_value_2"`
	PollIntervalMs                          types.Int64  `tfsdk:"poll_interval_ms"`
}

func (r *SourceMongoDBResource) Metadata(ctx context.Context, req res.MetadataRequest, resp *res.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_source_mongodb"
}

func (r *SourceMongoDBResource) Schema(ctx context.Context, req res.SchemaRequest, resp *res.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description:         "Source MongoDB resource",
		MarkdownDescription: "Source MongoDB resource",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				Description:         "Source MongoDB identifier",
				MarkdownDescription: "Source MongoDB identifier",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:            true,
				Description:         "Source name",
				MarkdownDescription: "Source name",
			},
			"connector": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"mongodb_connection_string": schema.StringAttribute{
				Required:            true,
				Sensitive:           true,
				Description:         "Mongodb Connection String. See Mongodb documentation for further details.",
				MarkdownDescription: "Mongodb Connection String. See Mongodb documentation for further details.",
			},
			"array_encoding": schema.StringAttribute{
				Computed:            true,
				Optional:            true,
				Default:             stringdefault.StaticString("array_string"),
				Description:         "How to encode arrays. 'Array' encodes them as Array objects but requires all values in the array to be of the same type. 'Array_String' encodes them as JSON Strings and should be used if arrays have mixed types",
				MarkdownDescription: "How to encode arrays. 'Array' encodes them as Array objects but requires all values in the array to be of the same type. 'Array_String' encodes them as JSON Strings and should be used if arrays have mixed types",
				Validators: []validator.String{
					stringvalidator.OneOf(
						"array",
						"array_string",
					),
				},
			},
			"nested_document_encoding": schema.StringAttribute{
				Computed:            true,
				Optional:            true,
				Default:             stringdefault.StaticString("document"),
				Description:         "How to encode nested documents. 'Document' encodes them as JSON Objects, 'String' encodes them as JSON Strings",
				MarkdownDescription: "How to encode nested documents. 'Document' encodes them as JSON Objects, 'String' encodes them as JSON Strings",
				Validators: []validator.String{
					stringvalidator.OneOf(
						"document",
						"string",
					),
				},
			},
			"database_include_list": schema.StringAttribute{
				Required:            true,
				Description:         "Source databases to sync.",
				MarkdownDescription: "Source databases to sync.",
			},
			"collection_include_list": schema.StringAttribute{
				Required:            true,
				Description:         "Source collections to sync.",
				MarkdownDescription: "Source collections to sync.",
			},
			"signal_data_collection_schema_or_database": schema.StringAttribute{
				Required:            true,
				Description:         "Streamkap will use a collection in this database to monitor incremental snapshotting. Follow the instructions in the documentation for creating this collection and specify which database to use here.",
				MarkdownDescription: "Streamkap will use a collection in this database to monitor incremental snapshotting. Follow the instructions in the documentation for creating this collection and specify which database to use here.",
			},
			"ssh_enabled": schema.BoolAttribute{
				Computed:            true,
				Optional:            true,
				Default:             booldefault.StaticBool(false),
				Description:         "Connect via SSH tunnel",
				MarkdownDescription: "Connect via SSH tunnel",
			},
			"ssh_host": schema.StringAttribute{
				Optional:            true,
				Description:         "Hostname of the SSH server, only required if `ssh_enabled` is true",
				MarkdownDescription: "Hostname of the SSH server, only required if `ssh_enabled` is true",
			},
			"ssh_port": schema.StringAttribute{
				Computed:            true,
				Optional:            true,
				Default:             stringdefault.StaticString("22"),
				Description:         "Port of the SSH server, only required if `ssh_enabled` is true",
				MarkdownDescription: "Port of the SSH server, only required if `ssh_enabled` is true",
			},
			"ssh_user": schema.StringAttribute{
				Computed:            true,
				Optional:            true,
				Default:             stringdefault.StaticString("streamkap"),
				Description:         "User for connecting to the SSH server, only required if `ssh_enabled` is true",
				MarkdownDescription: "User for connecting to the SSH server, only required if `ssh_enabled` is true",
			},
			"predicates_istopictoenrich_pattern": schema.StringAttribute{
				Computed:            true,
				Optional:            true,
				Default:             stringdefault.StaticString("$^"),
				Description:         "Regex pattern to match topics for enrichment",
				MarkdownDescription: "Regex pattern to match topics for enrichment",
			},
			"insert_static_key_field_1": schema.StringAttribute{
				Computed:            true,
				Optional:            true,
				Default:             stringdefault.StaticString(""),
				Description:         "The name of the static field to be added to the message key.",
				MarkdownDescription: "The name of the static field to be added to the message key.",
			},
			"insert_static_key_value_1": schema.StringAttribute{
				Computed:            true,
				Optional:            true,
				Default:             stringdefault.StaticString(""),
				Description:         "The value of the static field to be added to the message key.",
				MarkdownDescription: "The value of the static field to be added to the message key.",
			},
			"insert_static_value_field_1": schema.StringAttribute{
				Computed:            true,
				Optional:            true,
				Default:             stringdefault.StaticString(""),
				Description:         "The name of the static field to be added to the message value.",
				MarkdownDescription: "The name of the static field to be added to the message value.",
			},
			"insert_static_value_1": schema.StringAttribute{
				Computed:            true,
				Optional:            true,
				Default:             stringdefault.StaticString(""),
				Description:         "The value of the static field to be added to the message value.",
				MarkdownDescription: "The value of the static field to be added to the message value.",
			},
			"insert_static_key_field_2": schema.StringAttribute{
				Computed:            true,
				Optional:            true,
				Default:             stringdefault.StaticString(""),
				Description:         "The name of the static field to be added to the message key.",
				MarkdownDescription: "The name of the static field to be added to the message key.",
			},
			"insert_static_key_value_2": schema.StringAttribute{
				Computed:            true,
				Optional:            true,
				Default:             stringdefault.StaticString(""),
				Description:         "The value of the static field to be added to the message key.",
				MarkdownDescription: "The value of the static field to be added to the message key.",
			},
			"insert_static_value_field_2": schema.StringAttribute{
				Computed:            true,
				Optional:            true,
				Default:             stringdefault.StaticString(""),
				Description:         "The name of the static field to be added to the message value.",
				MarkdownDescription: "The name of the static field to be added to the message value.",
			},
			"insert_static_value_2": schema.StringAttribute{
				Computed:            true,
				Optional:            true,
				Default:             stringdefault.StaticString(""),
				Description:         "The value of the static field to be added to the message value.",
				MarkdownDescription: "The value of the static field to be added to the message value.",
			},
			"poll_interval_ms": schema.Int64Attribute{
				Computed:            true,
				Optional:            true,
				Default:             int64default.StaticInt64(500),
				Description:         "The number of milliseconds the connector waits for new change events to appear before processing a batch.",
				MarkdownDescription: "The number of milliseconds the connector waits for new change events to appear before processing a batch.",
			},
		},
	}
}

func (r *SourceMongoDBResource) Configure(ctx context.Context, req res.ConfigureRequest, resp *res.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(api.StreamkapAPI)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Source MongoDB Configure Type",
			fmt.Sprintf("Expected api.StreamkapAPI, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}
	r.client = client
}

func (r *SourceMongoDBResource) Create(ctx context.Context, req res.CreateRequest, resp *res.CreateResponse) {
	var plan SourceMongoDBResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	plan.Connector = types.StringValue(r.connector_code)

	if resp.Diagnostics.HasError() {
		return
	}

	config := r.model2ConfigMap(plan)

	source, err := r.client.CreateSource(ctx, api.Source{
		Name:      plan.Name.ValueString(),
		Connector: plan.Connector.ValueString(),
		Config:    config,
	})

	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating MongoDB source",
			fmt.Sprintf("Unable to create MongoDB source, got error: %s", err),
		)
		return
	}
	tflog.Info(ctx, "===> config: "+fmt.Sprintf("%+v", source))
	plan.ID = types.StringValue(source.ID)
	plan.Name = types.StringValue(source.Name)
	plan.Connector = types.StringValue(source.Connector)
	r.configMap2Model(source.Config, &plan)

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *SourceMongoDBResource) Read(ctx context.Context, req res.ReadRequest, resp *res.ReadResponse) {
	var state SourceMongoDBResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	sourceID := state.ID.ValueString()
	source, err := r.client.GetSource(ctx, sourceID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading MongoDB source",
			fmt.Sprintf("Unable to read MongoDB source, got error: %s", err),
		)
		return
	}
	if source == nil {
		resp.Diagnostics.AddError(
			"Error reading MongoDB source",
			fmt.Sprintf("MongoDB source %s does not exist", sourceID),
		)
		return
	}

	state.Name = types.StringValue(source.Name)
	state.Connector = types.StringValue(source.Connector)
	r.configMap2Model(source.Config, &state)
	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *SourceMongoDBResource) Update(ctx context.Context, req res.UpdateRequest, resp *res.UpdateResponse) {
	var plan SourceMongoDBResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "===> config: "+fmt.Sprintf("%+v", plan))
	config := r.model2ConfigMap(plan)

	source, err := r.client.UpdateSource(ctx, plan.ID.ValueString(), api.Source{
		Name:      plan.Name.ValueString(),
		Connector: plan.Connector.ValueString(),
		Config:    config,
	})

	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating MongoDB source",
			fmt.Sprintf("Unable to update MongoDB source, got error: %s", err),
		)
		return
	}

	// Update resource state with updated items
	plan.Name = types.StringValue(source.Name)
	plan.Connector = types.StringValue(source.Connector)
	r.configMap2Model(source.Config, &plan)

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *SourceMongoDBResource) Delete(ctx context.Context, req res.DeleteRequest, resp *res.DeleteResponse) {
	var state SourceMongoDBResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteSource(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting MongoDB source",
			fmt.Sprintf("Unable to delete MongoDB source, got error: %s", err),
		)
		return
	}
}

func (r *SourceMongoDBResource) ImportState(ctx context.Context, req res.ImportStateRequest, resp *res.ImportStateResponse) {
	res.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// Helpers
func (r *SourceMongoDBResource) model2ConfigMap(model SourceMongoDBResourceModel) map[string]any {
	return map[string]any{
		"mongodb.connection.string.user.defined":    model.MongoDBConnectionString.ValueString(),
		"transforms.unwrap.array.encoding":          model.ArrayEncoding.ValueString(),
		"transforms.unwrap.document.encoding":       model.NestedDocumentEncoding.ValueString(),
		"database.include.list":                     model.DatabaseIncludeList.ValueString(),
		"collection.include.list.user.defined":      model.CollectionIncludeList.ValueString(),
		"signal.data.collection.schema.or.database": model.SignalDataCollectionSchemaOrDatabase.ValueString(),
		"ssh.enabled": model.SSHEnabled.ValueBool(),
		"ssh.host":    model.SSHHost.ValueStringPointer(),
		"ssh.port":    model.SSHPort.ValueString(),
		"ssh.user":    model.SSHUser.ValueString(),
		"predicates.IsTopicToEnrich.pattern":    model.PredicatesIsTopicToEnrichPattern.ValueString(),
		"transforms.InsertStaticKey1.static.field":      model.InsertStaticKeyField1.ValueString(),
		"transforms.InsertStaticKey1.static.value":      model.InsertStaticKeyValue1.ValueString(),
		"transforms.InsertStaticValue1.static.field":    model.InsertStaticValueField1.ValueString(),
		"transforms.InsertStaticValue1.static.value":    model.InsertStaticValue1.ValueString(),
		"transforms.InsertStaticKey2.static.field":      model.InsertStaticKeyField2.ValueString(),
		"transforms.InsertStaticKey2.static.value":      model.InsertStaticKeyValue2.ValueString(),
		"transforms.InsertStaticValue2.static.field":    model.InsertStaticValueField2.ValueString(),
		"transforms.InsertStaticValue2.static.value":    model.InsertStaticValue2.ValueString(),
		"poll.interval.ms":                              int(model.PollIntervalMs.ValueInt64()),
	}
}

func (r *SourceMongoDBResource) configMap2Model(cfg map[string]any, model *SourceMongoDBResourceModel) {
	// Copy the config map to the model
	model.MongoDBConnectionString = helper.GetTfCfgString(cfg, "mongodb.connection.string.user.defined")
	model.ArrayEncoding = helper.GetTfCfgString(cfg, "transforms.unwrap.array.encoding")
	model.NestedDocumentEncoding = helper.GetTfCfgString(cfg, "transforms.unwrap.document.encoding")
	model.DatabaseIncludeList = helper.GetTfCfgString(cfg, "database.include.list")
	model.CollectionIncludeList = helper.GetTfCfgString(cfg, "collection.include.list.user.defined")
	model.SignalDataCollectionSchemaOrDatabase = helper.GetTfCfgString(cfg, "signal.data.collection.schema.or.database")
	model.SSHEnabled = helper.GetTfCfgBool(cfg, "ssh.enabled")
	model.SSHHost = helper.GetTfCfgString(cfg, "ssh.host")
	model.SSHPort = helper.GetTfCfgString(cfg, "ssh.port")
	model.SSHUser = helper.GetTfCfgString(cfg, "ssh.user")
	model.PredicatesIsTopicToEnrichPattern = helper.GetTfCfgString(cfg, "predicates.IsTopicToEnrich.pattern")
	model.InsertStaticKeyField1 = helper.GetTfCfgString(cfg, "transforms.InsertStaticKey1.static.field")
	model.InsertStaticKeyValue1 = helper.GetTfCfgString(cfg, "transforms.InsertStaticKey1.static.value")
	model.InsertStaticValueField1 = helper.GetTfCfgString(cfg, "transforms.InsertStaticValue1.static.field")
	model.InsertStaticValue1 = helper.GetTfCfgString(cfg, "transforms.InsertStaticValue1.static.value")
	model.InsertStaticKeyField2 = helper.GetTfCfgString(cfg, "transforms.InsertStaticKey2.static.field")
	model.InsertStaticKeyValue2 = helper.GetTfCfgString(cfg, "transforms.InsertStaticKey2.static.value")
	model.InsertStaticValueField2 = helper.GetTfCfgString(cfg, "transforms.InsertStaticValue2.static.field")
	model.InsertStaticValue2 = helper.GetTfCfgString(cfg, "transforms.InsertStaticValue2.static.value")
	model.PollIntervalMs = helper.GetTfCfgInt64(cfg, "poll.interval.ms")
}
