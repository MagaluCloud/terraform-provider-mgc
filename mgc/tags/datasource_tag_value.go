package tags

import (
	"context"
	"fmt"
	"time"

	tagSDK "github.com/MagaluCloud/mgc-sdk-go/tag"
	"github.com/MagaluCloud/terraform-provider-mgc/mgc/utils"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &tagValueDataSource{}

// tagValueModel describes a value in both the singular and the plural data
// source, which keeps the two from drifting apart.
type tagValueModel struct {
	ID          types.String `tfsdk:"id"`
	TagName     types.String `tfsdk:"tag_name"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	CreatedAt   types.String `tfsdk:"created_at"`
	UpdatedAt   types.String `tfsdk:"updated_at"`
}

type tagValueDataSource struct {
	values tagSDK.TagValueService
}

func NewTagValueDataSource() datasource.DataSource {
	return &tagValueDataSource{}
}

func (d *tagValueDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_tag_value"
}

func (d *tagValueDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	dataConfig, ok := req.ProviderData.(utils.DataConfig)
	if !ok {
		resp.Diagnostics.AddError("Failed to configure data source", "Invalid provider data")
		return
	}

	d.values = newTagClient(dataConfig).Values()
}

func (d *tagValueDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	attributes := tagValueAttributes()
	attributes["tag_name"] = schema.StringAttribute{
		Description: "Name of the tag that owns the value.",
		Required:    true,
	}
	attributes["name"] = schema.StringAttribute{
		Description: "Name of the value to look up. Names are case sensitive.",
		Required:    true,
	}

	resp.Schema = schema.Schema{
		Description: "Reads a single value of a tag by name.",
		Attributes:  attributes,
	}
}

func (d *tagValueDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data tagValueModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tagName, name := data.TagName.ValueString(), data.Name.ValueString()
	value, err := d.values.Get(ctx, tagName, name)
	if err != nil {
		if isNotFound(err) {
			// A deleted parent tag takes its values with it, and the API reports it
			// the same way as a missing value, hence the two-sided message.
			resp.Diagnostics.AddError(
				"Tag value not found",
				fmt.Sprintf("The tag %q has no value named %q, or the tag itself does not exist.", tagName, name),
			)
			return
		}
		resp.Diagnostics.AddError(utils.ParseSDKError(err))
		return
	}

	data = convertTagValue(tagName, *value)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// tagValueAttributes describes a value as read from the API. The singular data
// source overrides tag_name and name, which are the search key there.
func tagValueAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"id": schema.StringAttribute{
			Description: "Identifier of the value, in the form `<tag_name>,<name>`.",
			Computed:    true,
		},
		"tag_name": schema.StringAttribute{
			Description: "Name of the tag that owns the value.",
			Computed:    true,
		},
		"name": schema.StringAttribute{
			Description: "Name of the value, unique within the tag.",
			Computed:    true,
		},
		"description": schema.StringAttribute{
			Description: "A brief description of the value.",
			Computed:    true,
		},
		"created_at": schema.StringAttribute{
			Description: "Creation date of the value.",
			Computed:    true,
		},
		"updated_at": schema.StringAttribute{
			Description: "Last update date of the value, null while it was never updated.",
			Computed:    true,
		},
	}
}

// convertTagValue maps a value of the API onto the model. The tag name comes from
// the caller because the API only embeds it when the value is read on its own.
func convertTagValue(tagName string, value tagSDK.TagValue) tagValueModel {
	return tagValueModel{
		ID:          types.StringValue(tagValueID(tagName, value.Name)),
		TagName:     types.StringValue(tagName),
		Name:        types.StringValue(value.Name),
		Description: types.StringPointerValue(value.Description),
		CreatedAt:   types.StringPointerValue(utils.ConvertTimeToRFC3339((*time.Time)(&value.CreatedAt))),
		UpdatedAt:   types.StringPointerValue(utils.ConvertTimeToRFC3339((*time.Time)(value.UpdatedAt))),
	}
}
