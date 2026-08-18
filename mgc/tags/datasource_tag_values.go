package tags

import (
	"context"
	"fmt"

	tagSDK "github.com/MagaluCloud/mgc-sdk-go/tag"
	"github.com/MagaluCloud/terraform-provider-mgc/mgc/utils"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &tagValuesDataSource{}

type tagValuesModel struct {
	TagName types.String    `tfsdk:"tag_name"`
	Values  []tagValueModel `tfsdk:"values"`
}

type tagValuesDataSource struct {
	values tagSDK.TagValueService
}

func NewTagValuesDataSource() datasource.DataSource {
	return &tagValuesDataSource{}
}

func (d *tagValuesDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_tag_values"
}

func (d *tagValuesDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *tagValuesDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Reads every value defined for a tag.",
		Attributes: map[string]schema.Attribute{
			"tag_name": schema.StringAttribute{
				Description: "Name of the tag whose values are read.",
				Required:    true,
			},
			"values": schema.ListNestedAttribute{
				Description: "The values of the tag, in the order the API returned them.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: tagValueAttributes(),
				},
			},
		},
	}
}

func (d *tagValuesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data tagValuesModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tagName := data.TagName.ValueString()
	values, err := listAllPages(func(limit, offset int) ([]tagSDK.TagValue, error) {
		return d.values.List(ctx, tagName, tagSDK.ListTagValuesOptions{Limit: &limit, Offset: &offset})
	})
	if err != nil {
		if isNotFound(err) {
			resp.Diagnostics.AddError(
				"Tag not found",
				fmt.Sprintf("No tag named %q was found in this tenant.", tagName),
			)
			return
		}
		resp.Diagnostics.AddError(utils.ParseSDKError(err))
		return
	}

	data.Values = make([]tagValueModel, 0, len(values))
	for _, value := range values {
		data.Values = append(data.Values, convertTagValue(tagName, value))
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
