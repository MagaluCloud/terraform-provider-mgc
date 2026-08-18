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
		Description: "Reads every resource of the tenant that carries at least one tag, optionally narrowed by type or region. " +
			"The API takes no filter by tag, so narrowing by tag is done over the result, as in " +
			"`[for attachment in data.mgc_tag_attachments.all.attachments : attachment if contains(keys(attachment.tags), \"finops\")]`.",
		Attributes: map[string]schema.Attribute{
			"resource_type": schema.StringAttribute{
				Description: "Only return resources of this type, such as `net.vpc`. " +
					"The types that support tagging are the ones listed by the `mgc_tag_resource_types` data source.",
				Optional: true,
			},
			"region": schema.StringAttribute{
				Description: "Only return resources of this region.",
				Optional:    true,
			},
			"attachments": schema.ListNestedAttribute{
				Description: "The tagged resources found, in the order the API returned them.",
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
