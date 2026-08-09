package tags

import (
	"context"

	tagSDK "github.com/MagaluCloud/mgc-sdk-go/tag"
	"github.com/MagaluCloud/terraform-provider-mgc/mgc/utils"
	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &tagsDataSource{}

type tagsModel struct {
	Color types.String `tfsdk:"color"`
	Kinds types.Set    `tfsdk:"kinds"`
	Tags  []tagModel   `tfsdk:"tags"`
}

type tagsDataSource struct {
	tags tagSDK.TagService
}

func NewTagsDataSource() datasource.DataSource {
	return &tagsDataSource{}
}

func (d *tagsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_tags"
}

func (d *tagsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *tagsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Reads the tags of the tenant, optionally narrowed by color or kind.",
		Attributes: map[string]schema.Attribute{
			"color": schema.StringAttribute{
				Description: "Only return tags of this color, as a 6-digit hex RGB code without the `#` prefix.",
				Optional:    true,
			},
			"kinds": schema.SetAttribute{
				Description: "Only return tags that have at least one of these kinds.",
				Optional:    true,
				ElementType: types.StringType,
				Validators: []validator.Set{
					setvalidator.ValueStringsAre(kindValidator{}),
				},
			},
			"tags": schema.ListNestedAttribute{
				Description: "The tags found, in the order the API returned them.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: tagAttributes(),
				},
			},
		},
	}
}

func (d *tagsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data tagsModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	options := tagSDK.ListTagsOptions{
		Color: utils.KnownStringPointer(data.Color),
		Kinds: toKinds(utils.ConvertTypeSetToStringArray(data.Kinds)),
	}

	tags, err := listAllPages(func(limit, offset int) ([]tagSDK.Tag, error) {
		options.Limit, options.Offset = &limit, &offset
		return d.tags.List(ctx, options)
	})
	if err != nil {
		resp.Diagnostics.AddError(utils.ParseSDKError(err))
		return
	}

	data.Tags = make([]tagModel, 0, len(tags))
	for _, tag := range tags {
		data.Tags = append(data.Tags, convertTag(tag))
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
