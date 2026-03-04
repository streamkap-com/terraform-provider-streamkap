package kafka_user

import (
	"context"
	"fmt"
	"regexp"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	res "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/streamkap-com/terraform-provider-streamkap/internal/api"
)

var (
	_ res.Resource                = &KafkaUserResource{}
	_ res.ResourceWithConfigure   = &KafkaUserResource{}
	_ res.ResourceWithImportState = &KafkaUserResource{}
)

func NewKafkaUserResource() res.Resource {
	return &KafkaUserResource{}
}

type KafkaUserResource struct {
	client api.StreamkapAPI
}

type KafkaACLModel struct {
	TopicName           types.String `tfsdk:"topic_name"`
	Operation           types.String `tfsdk:"operation"`
	ResourcePatternType types.String `tfsdk:"resource_pattern_type"`
	Resource            types.String `tfsdk:"resource"`
}

type KafkaUserResourceModel struct {
	ID                     types.String    `tfsdk:"id"`
	Username               types.String    `tfsdk:"username"`
	Password               types.String    `tfsdk:"password"`
	WhitelistIPs           types.String    `tfsdk:"whitelist_ips"`
	KafkaACLs              []KafkaACLModel `tfsdk:"kafka_acls"`
	IsCreateSchemaRegistry types.Bool      `tfsdk:"is_create_schema_registry"`
	KafkaProxyEndpoint     types.String    `tfsdk:"kafka_proxy_endpoint"`
	SchemaProxyEndpoint    types.String    `tfsdk:"schema_proxy_endpoint"`
}

func (r *KafkaUserResource) Metadata(ctx context.Context, req res.MetadataRequest, resp *res.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_kafka_user"
}

func (r *KafkaUserResource) Schema(ctx context.Context, req res.SchemaRequest, resp *res.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Streamkap Kafka user with ACL rules for topic access control.",
		MarkdownDescription: "Manages a **Streamkap Kafka user** with ACL rules for topic access control.\n\n" +
			"This resource creates and manages Kafka users that can connect to the Streamkap Kafka proxy " +
			"endpoint with fine-grained access control via ACL rules.\n\n" +
			"[Documentation](https://docs.streamkap.com/kafka-access)",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description:         "Unique identifier for the Kafka user. Same as the username.",
				MarkdownDescription: "Unique identifier for the Kafka user. Same as the username.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"username": schema.StringAttribute{
				Description:         "Username for the Kafka user. Must be 1-24 characters, alphanumeric with dashes only (cannot start or end with a dash). Cannot be changed after creation.",
				MarkdownDescription: "Username for the Kafka user. Must be 1-24 characters, alphanumeric with dashes only (cannot start or end with a dash). **Cannot be changed after creation.**",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 24),
					stringvalidator.RegexMatches(
						regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9-]*[a-zA-Z0-9]$|^[a-zA-Z0-9]$`),
						"must be alphanumeric with dashes only, cannot start or end with a dash",
					),
				},
			},
			"password": schema.StringAttribute{
				Description:         "Password for the Kafka user. Must be 12-128 characters. This value is sensitive and will not appear in logs or CLI output. Write-only: not returned by the API on read.",
				MarkdownDescription: "Password for the Kafka user. Must be 12-128 characters.\n\n**Security:** This value is marked sensitive and will not appear in CLI output or logs. Write-only: not returned by the API on read.",
				Required:            true,
				Sensitive:           true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
				Validators: []validator.String{
					stringvalidator.LengthBetween(12, 128),
				},
			},
			"whitelist_ips": schema.StringAttribute{
				Description:         "Comma-separated list of whitelisted IP addresses or CIDR ranges. Maximum 1000 characters.",
				MarkdownDescription: "Comma-separated list of whitelisted IP addresses or CIDR ranges. Maximum 1000 characters.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(""),
				Validators: []validator.String{
					stringvalidator.LengthAtMost(1000),
				},
			},
			"is_create_schema_registry": schema.BoolAttribute{
				Description:         "Whether to create a schema registry user alongside the Kafka user. Defaults to false.",
				MarkdownDescription: "Whether to create a schema registry user alongside the Kafka user. Defaults to `false`.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"kafka_proxy_endpoint": schema.StringAttribute{
				Description:         "The Kafka proxy endpoint for this user. Computed by the server.",
				MarkdownDescription: "The Kafka proxy endpoint for this user. Computed by the server.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"schema_proxy_endpoint": schema.StringAttribute{
				Description:         "The schema registry proxy endpoint for this user. Computed by the server.",
				MarkdownDescription: "The schema registry proxy endpoint for this user. Computed by the server.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
		Blocks: map[string]schema.Block{
			"kafka_acls": schema.ListNestedBlock{
				Description:         "List of Kafka ACL rules for this user.",
				MarkdownDescription: "List of Kafka ACL rules for this user.",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"topic_name": schema.StringAttribute{
							Description:         "The topic name or pattern for this ACL rule.",
							MarkdownDescription: "The topic name or pattern for this ACL rule.",
							Required:            true,
						},
						"operation": schema.StringAttribute{
							Description:         "The ACL operation. Valid values for TOPIC resource: ALL, ALTER, ALTER_CONFIGS, CREATE, DELETE, DESCRIBE, DESCRIBE_CONFIGS, READ, WRITE. Valid values for GROUP resource: DELETE, DESCRIBE, READ.",
							MarkdownDescription: "The ACL operation. Valid values for TOPIC resource: `ALL`, `ALTER`, `ALTER_CONFIGS`, `CREATE`, `DELETE`, `DESCRIBE`, `DESCRIBE_CONFIGS`, `READ`, `WRITE`. Valid values for GROUP resource: `DELETE`, `DESCRIBE`, `READ`.",
							Required:            true,
							Validators: []validator.String{
								stringvalidator.OneOf("ALL", "ALTER", "ALTER_CONFIGS", "CREATE", "DELETE", "DESCRIBE", "DESCRIBE_CONFIGS", "READ", "WRITE"),
							},
						},
						"resource_pattern_type": schema.StringAttribute{
							Description:         "The resource pattern type. Valid values: LITERAL, PREFIXED.",
							MarkdownDescription: "The resource pattern type. Valid values: `LITERAL`, `PREFIXED`.",
							Required:            true,
							Validators: []validator.String{
								stringvalidator.OneOf("LITERAL", "PREFIXED"),
							},
						},
						"resource": schema.StringAttribute{
							Description:         "The resource type. Valid values: TOPIC, GROUP. Defaults to \"TOPIC\".",
							MarkdownDescription: "The resource type. Valid values: `TOPIC`, `GROUP`. Defaults to `TOPIC`.",
							Optional:            true,
							Computed:            true,
							Default:             stringdefault.StaticString("TOPIC"),
							Validators: []validator.String{
								stringvalidator.OneOf("TOPIC", "GROUP"),
							},
						},
					},
				},
			},
		},
	}
}

func (r *KafkaUserResource) Configure(ctx context.Context, req res.ConfigureRequest, resp *res.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	client, ok := req.ProviderData.(api.StreamkapAPI)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Kafka User Configure Type",
			fmt.Sprintf("Expected api.StreamkapAPI, got: %T.", req.ProviderData),
		)
		return
	}
	r.client = client
}

func (r *KafkaUserResource) Create(ctx context.Context, req res.CreateRequest, resp *res.CreateResponse) {
	var plan KafkaUserResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	acls := make([]api.KafkaACL, len(plan.KafkaACLs))
	for i, acl := range plan.KafkaACLs {
		acls[i] = api.KafkaACL{
			TopicName:           acl.TopicName.ValueString(),
			Operation:           acl.Operation.ValueString(),
			ResourcePatternType: acl.ResourcePatternType.ValueString(),
			Resource:            acl.Resource.ValueString(),
		}
	}

	user, err := r.client.CreateKafkaUser(ctx, api.CreateKafkaUserRequest{
		Username:               plan.Username.ValueString(),
		Password:               plan.Password.ValueString(),
		WhitelistIPs:           plan.WhitelistIPs.ValueString(),
		KafkaACLs:              acls,
		IsCreateSchemaRegistry: plan.IsCreateSchemaRegistry.ValueBool(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Error creating Kafka user", fmt.Sprintf("Unable to create Kafka user: %s", err))
		return
	}

	r.modelFromAPIObject(user, &plan)
	tflog.Info(ctx, "Created Kafka user: "+user.Username)
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *KafkaUserResource) Read(ctx context.Context, req res.ReadRequest, resp *res.ReadResponse) {
	var state KafkaUserResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	user, err := r.client.GetKafkaUser(ctx, state.Username.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading Kafka user", fmt.Sprintf("Unable to read Kafka user: %s", err))
		return
	}
	if user == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	r.modelFromAPIObject(user, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *KafkaUserResource) Update(ctx context.Context, req res.UpdateRequest, resp *res.UpdateResponse) {
	var plan KafkaUserResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	acls := make([]api.KafkaACL, len(plan.KafkaACLs))
	for i, acl := range plan.KafkaACLs {
		acls[i] = api.KafkaACL{
			TopicName:           acl.TopicName.ValueString(),
			Operation:           acl.Operation.ValueString(),
			ResourcePatternType: acl.ResourcePatternType.ValueString(),
			Resource:            acl.Resource.ValueString(),
		}
	}

	user, err := r.client.UpdateKafkaUser(ctx, plan.Username.ValueString(), api.UpdateKafkaUserRequest{
		Password:               plan.Password.ValueString(),
		WhitelistIPs:           plan.WhitelistIPs.ValueString(),
		KafkaACLs:              acls,
		IsCreateSchemaRegistry: plan.IsCreateSchemaRegistry.ValueBool(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Error updating Kafka user", fmt.Sprintf("Unable to update Kafka user: %s", err))
		return
	}

	r.modelFromAPIObject(user, &plan)
	tflog.Info(ctx, "Updated Kafka user: "+user.Username)
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *KafkaUserResource) Delete(ctx context.Context, req res.DeleteRequest, resp *res.DeleteResponse) {
	var state KafkaUserResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteKafkaUser(ctx, state.Username.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error deleting Kafka user", fmt.Sprintf("Unable to delete Kafka user: %s", err))
		return
	}
	tflog.Info(ctx, "Deleted Kafka user: "+state.Username.ValueString())
}

func (r *KafkaUserResource) ImportState(ctx context.Context, req res.ImportStateRequest, resp *res.ImportStateResponse) {
	// For Kafka users, the import ID is the username
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("username"), req.ID)...)
}

func (r *KafkaUserResource) modelFromAPIObject(apiObject *api.KafkaUser, model *KafkaUserResourceModel) {
	model.ID = types.StringValue(apiObject.Username)
	model.Username = types.StringValue(apiObject.Username)
	model.WhitelistIPs = types.StringValue(apiObject.WhitelistIPs)
	model.KafkaProxyEndpoint = types.StringValue(apiObject.KafkaProxyEndpoint)
	model.SchemaProxyEndpoint = types.StringValue(apiObject.SchemaProxyEndpoint)
	model.IsCreateSchemaRegistry = types.BoolValue(apiObject.IsCreateSchemaRegistry)

	acls := make([]KafkaACLModel, len(apiObject.KafkaACLs))
	for i, acl := range apiObject.KafkaACLs {
		acls[i] = KafkaACLModel{
			TopicName:           types.StringValue(acl.TopicName),
			Operation:           types.StringValue(acl.Operation),
			ResourcePatternType: types.StringValue(acl.ResourcePatternType),
			Resource:            types.StringValue(acl.Resource),
		}
	}
	model.KafkaACLs = acls
}
