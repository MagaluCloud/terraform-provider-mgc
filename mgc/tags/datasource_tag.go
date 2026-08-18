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

var _ datasource.DataSource = &tagDataSource{}

type tagModel struct {
	ID          types.String    `tfsdk:"id"`
	Name        types.String    `tfsdk:"name"`
	Description types.String    `tfsdk:"description"`
	Color       types.String    `tfsdk:"color"`
	Kinds       types.Set       `tfsdk:"kinds"`
	Values      []tagValueModel `tfsdk:"values"`
	CreatedAt   types.String    `tfsdk:"created_at"`
	UpdatedAt   types.String    `tfsdk:"updated_at"`
}

type tagDataSource struct {
	tags tagSDK.TagService
}

func NewTagDataSource() datasource.DataSource {
	return &tagDataSource{}
}

func (d *tagDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_tag"
}

func (d *tagDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	dataConfig, ok := req.ProviderData.(utils.DataConfig)
	if !ok {
		resp.Diagnostics.AddError("Failed to configure data source", "Invalid provider data")
		return
	}

	d.tags = newTagClient(dataConfig).Tags()
}

func (d *tagDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	attributes := tagAttributes()
	attributes["name"] = schema.StringAttribute{
		Description: "Name of the tag to look up. Names are case sensitive: `finops` and `FinOps` are different tags.",
		Required:    true,
	}

	resp.Schema = schema.Schema{
		Description: "Reads a tag of the tenant by name, along with the values defined for it.",
		Attributes:  attributes,
	}
}

func (d *tagDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data tagModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	name := data.Name.ValueString()
	tag, err := d.tags.Get(ctx, name)
	if err != nil {
		if isNotFound(err) {
			resp.Diagnostics.AddError(
				"Tag not found",
				fmt.Sprintf("No tag named %q was found in this tenant.", name),
			)
			return
		}
		resp.Diagnostics.AddError(utils.ParseSDKError(err))
		return
	}

	data = convertTag(*tag)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// tagAttributes describes a tag as read from the API. The singular data source
// overrides name, which is the search key there.
func tagAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"id": schema.StringAttribute{
			Description: "The tag name, which is also its identifier.",
			Computed:    true,
		},
		"name": schema.StringAttribute{
			Description: "Name of the tag, unique within the tenant.",
			Computed:    true,
		},
		"description": schema.StringAttribute{
			Description: "A brief description of the tag.",
			Computed:    true,
		},
		"color": schema.StringAttribute{
			Description: "Color of the tag, as a 6-digit hex RGB code without the `#` prefix, stored lowercased.",
			Computed:    true,
		},
		"kinds": schema.SetAttribute{
			Description: "Kinds that describe what the tag is for, such as `finops`.",
			Computed:    true,
			ElementType: types.StringType,
		},
		"values": schema.ListNestedAttribute{
			Description: "The values defined for the tag, as the API embedded them in the answer.",
			Computed:    true,
			NestedObject: schema.NestedAttributeObject{
				Attributes: tagValueAttributes(),
			},
		},
		"created_at": schema.StringAttribute{
			Description: "Creation date of the tag.",
			Computed:    true,
		},
		"updated_at": schema.StringAttribute{
			Description: "Last update date of the tag, null while it was never updated.",
			Computed:    true,
		},
	}
}

// convertTag maps a tag of the API onto the model. A data source has no previous
// configuration to preserve, so this is a plain conversion: the merge the
// resources do with flattenTag would have nothing to merge here.
func convertTag(tag tagSDK.Tag) tagModel {
	// A tag with no values answers with [], which is not the same as an answer
	// that leaves the field out, as the listing endpoint does.
	var values []tagValueModel
	if tag.Values != nil {
		values = make([]tagValueModel, 0, len(tag.Values))
		for _, value := range tag.Values {
			values = append(values, convertTagValue(tag.Name, value))
		}
	}

	return tagModel{
		ID:          types.StringValue(tag.Name),
		Name:        types.StringValue(tag.Name),
		Description: types.StringPointerValue(tag.Description),
		Color:       types.StringPointerValue(tag.Color),
		Kinds:       utils.StringSliceToTypesSet(kindsToStrings(tag.Kinds)),
		Values:      values,

		// The timestamp type of the SDK lives in an internal package and cannot be
		// named here, but it converts to time.Time.
		CreatedAt: types.StringPointerValue(utils.ConvertTimeToRFC3339((*time.Time)(&tag.CreatedAt))),
		UpdatedAt: types.StringPointerValue(utils.ConvertTimeToRFC3339((*time.Time)(tag.UpdatedAt))),
	}
}
