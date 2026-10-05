package tags

import (
	"context"

	tagSDK "github.com/MagaluCloud/mgc-sdk-go/tag"
	"github.com/MagaluCloud/terraform-provider-mgc/mgc/utils"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &tagAttachmentsDataSource{}

type tagAttachmentsModel struct {
	ResourceType types.String         `tfsdk:"resource_type"`
	Region       types.String         `tfsdk:"region"`
	TagName      types.String         `tfsdk:"tag_name"`
	Attachments  []tagAttachmentModel `tfsdk:"attachments"`
}

type tagAttachmentsDataSource struct {
	resources tagSDK.ResourceService
}

func NewTagAttachmentsDataSource() datasource.DataSource {
	return &tagAttachmentsDataSource{}
}

func (d *tagAttachmentsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_tag_attachments"
}

func (d *tagAttachmentsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	dataConfig, ok := req.ProviderData.(utils.DataConfig)
	if !ok {
		resp.Diagnostics.AddError("Failed to configure data source", "Invalid provider data")
		return
	}

	d.resources = newTagClient(dataConfig).Resources()
}

func (d *tagAttachmentsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Reads the tagged resources, optionally filtered by type, region or tag.",
		Attributes: map[string]schema.Attribute{
			"resource_type": schema.StringAttribute{
				Description: "Filter by resource type, such as `net.vpc`.",
				Optional:    true,
			},
			"region": schema.StringAttribute{
				Description: "Filter by region.",
				Optional:    true,
			},
			"tag_name": schema.StringAttribute{
				Description: "Filter by tag name. To filter by value, use a `for` expression over `attachments`.",
				Optional:    true,
			},
			"attachments": schema.ListNestedAttribute{
				Description: "Tagged resources found.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: attachmentAttributes(),
				},
			},
		},
	}
}

func (d *tagAttachmentsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data tagAttachmentsModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	options := tagSDK.ListResourcesOptions{
		ResourceTypeName: namedStringPointer[tagSDK.ResourceTypeName](data.ResourceType),
		Region:           utils.KnownStringPointer(data.Region),
		TagName:          utils.KnownStringPointer(data.TagName),
	}

	taggedResources, err := listAllPages(func(limit, offset int) ([]tagSDK.Resource, error) {
		options.Limit, options.Offset = &limit, &offset
		return d.resources.List(ctx, options)
	})
	if err != nil {
		resp.Diagnostics.AddError(utils.ParseSDKError(err))
		return
	}

	data.Attachments = make([]tagAttachmentModel, 0, len(taggedResources))
	for _, taggedResource := range taggedResources {
		data.Attachments = append(data.Attachments, convertAttachment(taggedResource))
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// namedStringPointer turns an optional filter into the pointer the SDK takes.
// The filters of the tags API are named string types, so a plain *string does
// not fit them.
func namedStringPointer[T ~string](value types.String) *T {
	known := utils.KnownStringPointer(value)
	if known == nil {
		return nil
	}

	converted := T(*known)
	return &converted
}
