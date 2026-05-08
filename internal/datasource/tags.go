package datasource

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	ds "github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/streamkap-com/terraform-provider-streamkap/internal/api"
	"github.com/streamkap-com/terraform-provider-streamkap/internal/constants"
)

var _ ds.DataSource = &TagsDataSource{}

func NewTagsDataSource() ds.DataSource {
	return &TagsDataSource{}
}

type TagsDataSource struct {
	client api.StreamkapAPI
}

type TagsDataSourceModel struct {
	FilterName types.String `tfsdk:"filter_name"`
	FilterType types.Set    `tfsdk:"filter_type"`
	FilterIds  types.Set    `tfsdk:"filter_ids"`
	Tags       types.List   `tfsdk:"tags"`
}

// tagAttrTypes — single source of truth for the nested-object type used in the
// `tags` computed attribute. Used both in the schema definition and when
// building values in Read.
func tagAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"id":          types.StringType,
		"name":        types.StringType,
		"description": types.StringType,
		"type":        types.SetType{ElemType: types.StringType},
		"system":      types.BoolType,
		"custom":      types.BoolType,
	}
}

func (d *TagsDataSource) Metadata(ctx context.Context, req ds.MetadataRequest, resp *ds.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_tags"
}

func (d *TagsDataSource) Schema(ctx context.Context, req ds.SchemaRequest, resp *ds.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Lists Streamkap tags for the current tenant, optionally filtered by name, type, or specific IDs.",
		MarkdownDescription: "Lists **Streamkap tags** for the current tenant, optionally filtered " +
			"by name, type, or specific IDs.\n\n" +
			"Use this data source to look up tag IDs by name (instead of hardcoding them) or to " +
			"build dynamic tag selections (e.g. all `environment` tags).\n\n" +
			"All filters are optional. When multiple filters are set the backend ANDs them. " +
			"For very large `filter_ids` lists this data source automatically routes through " +
			"`POST /tags/search` to avoid URL length limits.\n\n" +
			"[Documentation](https://docs.streamkap.com/streamkap-provider-for-terraform)",
		Attributes: map[string]schema.Attribute{
			"filter_name": schema.StringAttribute{
				Description:         "Optional filter: only tags whose name matches this value.",
				MarkdownDescription: "Optional filter: only tags whose `name` matches this value.",
				Optional:            true,
			},
			"filter_type": schema.SetAttribute{
				Description: "Optional filter: only tags whose type set contains any of these entity-type values. " +
					"Valid values: environment, general, sources, destinations, pipelines, transforms, " +
					"topics, services, users, tenant.",
				MarkdownDescription: "Optional filter: only tags whose `type` set contains any of these " +
					"entity-type values. Valid values: `environment`, `general`, `sources`, `destinations`, " +
					"`pipelines`, `transforms`, `topics`, `services`, `users`, `tenant`.",
				Optional:    true,
				ElementType: types.StringType,
				Validators: []validator.Set{
					setvalidator.ValueStringsAre(stringvalidator.OneOf(constants.TagTypeEnum...)),
				},
			},
			"filter_ids": schema.SetAttribute{
				Description: "Optional filter: only tags with these IDs. Useful for resolving a known set " +
					"of tag IDs to their full records (name, type, etc.) in one call.",
				MarkdownDescription: "Optional filter: only tags with these IDs. Useful for resolving a known " +
					"set of tag IDs to their full records (name, type, etc.) in one call.",
				Optional:    true,
				ElementType: types.StringType,
			},
			"tags": schema.ListNestedAttribute{
				Description:         "Tags matching the filters.",
				MarkdownDescription: "Tags matching the filters.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Computed:            true,
							Description:         "Unique identifier of the tag.",
							MarkdownDescription: "Unique identifier of the tag.",
						},
						"name": schema.StringAttribute{
							Computed:            true,
							Description:         "Display name of the tag.",
							MarkdownDescription: "Display name of the tag.",
						},
						"description": schema.StringAttribute{
							Computed:            true,
							Description:         "Description of the tag.",
							MarkdownDescription: "Description of the tag.",
						},
						"type": schema.SetAttribute{
							Computed:            true,
							ElementType:         types.StringType,
							Description:         "Set of entity types this tag applies to.",
							MarkdownDescription: "Set of entity types this tag applies to.",
						},
						"system": schema.BoolAttribute{
							Computed:            true,
							Description:         "True if this is a built-in system tag.",
							MarkdownDescription: "True if this is a built-in system tag.",
						},
						"custom": schema.BoolAttribute{
							Computed:            true,
							Description:         "True if this is a custom (user-created) tag.",
							MarkdownDescription: "True if this is a custom (user-created) tag.",
						},
					},
				},
			},
		},
	}
}

func (d *TagsDataSource) Configure(ctx context.Context, req ds.ConfigureRequest, resp *ds.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	client, ok := req.ProviderData.(api.StreamkapAPI)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Tags Configure Type",
			fmt.Sprintf("Expected api.StreamkapAPI, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}
	d.client = client
}

func (d *TagsDataSource) Read(ctx context.Context, req ds.ReadRequest, resp *ds.ReadResponse) {
	var cfg TagsDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &cfg)...)
	if resp.Diagnostics.HasError() {
		return
	}

	filters := api.TagListFilters{
		Name: cfg.FilterName.ValueString(),
	}
	if !cfg.FilterType.IsNull() && !cfg.FilterType.IsUnknown() {
		resp.Diagnostics.Append(cfg.FilterType.ElementsAs(ctx, &filters.Types, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
	}
	if !cfg.FilterIds.IsNull() && !cfg.FilterIds.IsUnknown() {
		resp.Diagnostics.Append(cfg.FilterIds.ElementsAs(ctx, &filters.IDs, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	tags, err := d.client.ListTags(ctx, filters)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error listing tags",
			fmt.Sprintf("Unable to list tags: %s", err),
		)
		return
	}

	objs := make([]attr.Value, len(tags))
	for i, t := range tags {
		typeElems := make([]attr.Value, len(t.Type))
		for j, v := range t.Type {
			typeElems[j] = types.StringValue(v)
		}
		typeSet, diags := types.SetValue(types.StringType, typeElems)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}

		obj, diags := types.ObjectValue(tagAttrTypes(), map[string]attr.Value{
			"id":          types.StringValue(t.ID),
			"name":        types.StringValue(t.Name),
			"description": types.StringValue(t.Description),
			"type":        typeSet,
			"system":      types.BoolValue(t.System),
			"custom":      types.BoolPointerValue(t.Custom),
		})
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		objs[i] = obj
	}

	tagsList, diags := types.ListValue(types.ObjectType{AttrTypes: tagAttrTypes()}, objs)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	cfg.Tags = tagsList

	resp.Diagnostics.Append(resp.State.Set(ctx, &cfg)...)
}
