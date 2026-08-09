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

var _ datasource.DataSource = &tagAttachmentDataSource{}

// tagAttachmentModel describes a tagged resource in both the singular and the
// plural data source, which keeps the two from drifting apart.
type tagAttachmentModel struct {
	ID                  types.String `tfsdk:"id"`
	ResourceID          types.String `tfsdk:"resource_id"`
	Tags                types.Map    `tfsdk:"tags"`
	ResourceType        types.String `tfsdk:"resource_type"`
	Product             types.String `tfsdk:"product"`
	Region              types.String `tfsdk:"region"`
	CreatedAt           types.String `tfsdk:"created_at"`
	UpdatedAt           types.String `tfsdk:"updated_at"`
	LastTagAssociatedAt types.String `tfsdk:"last_tag_associated_at"`
}

type tagAttachmentDataSource struct {
	resources tagSDK.ResourceService
}

func NewTagAttachmentDataSource() datasource.DataSource {
	return &tagAttachmentDataSource{}
}

func (d *tagAttachmentDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_tag_attachment"
}

func (d *tagAttachmentDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *tagAttachmentDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	attributes := attachmentAttributes()
	attributes["resource_id"] = schema.StringAttribute{
		Description: "Id of the tagged resource, as used by its own product, such as the id of a cluster or of a VPC.",
		Required:    true,
	}

	resp.Schema = schema.Schema{
		Description: "Reads the tags attached to a cloud resource. " +
			"Unlike the resource of the same name, this only reads: it takes no ownership of the tags, " +
			"so it is the way to look at a resource whose tags are managed elsewhere.",
		Attributes: attributes,
	}
}

func (d *tagAttachmentDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data tagAttachmentModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resourceID := data.ResourceID.ValueString()
	taggedResource, err := d.resources.Get(ctx, resourceID)
	if err != nil {
		if isNotFound(err) {
			// The API has no answer for a resource without tags, so the two cases
			// cannot be told apart and the message has to name both.
			resp.Diagnostics.AddError(
				"Tagged resource not found",
				fmt.Sprintf("The resource %q carries no tag, or does not exist.", resourceID),
			)
			return
		}
		resp.Diagnostics.AddError(utils.ParseSDKError(err))
		return
	}

	data = convertAttachment(*taggedResource)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// attachmentAttributes describes a tagged resource as read from the API. The
// singular data source overrides resource_id, which is the search key there.
func attachmentAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"id": schema.StringAttribute{
			Description: "Identifier of the attachment, which is the id of the tagged resource.",
			Computed:    true,
		},
		"resource_id": schema.StringAttribute{
			Description: "Id of the tagged resource, as used by its own product.",
			Computed:    true,
		},
		"tags": schema.MapAttribute{
			Description: "Every tag the resource carries, as a map of tag name to value.",
			Computed:    true,
			ElementType: types.StringType,
		},
		"resource_type": schema.StringAttribute{
			Description: "Type of the resource, as classified by the API, such as `net.vpc`. " +
				"The types that support tagging are the ones listed by the `mgc_tag_resource_types` data source.",
			Computed: true,
		},
		"product": schema.StringAttribute{
			Description: "Product that owns the resource type, such as `network`.",
			Computed:    true,
		},
		"region": schema.StringAttribute{
			Description: "Region of the resource.",
			Computed:    true,
		},
		"created_at": schema.StringAttribute{
			Description: "Date the resource became known to the tags service.",
			Computed:    true,
		},
		"updated_at": schema.StringAttribute{
			Description: "Last update date of the resource, null while it was never updated.",
			Computed:    true,
		},
		"last_tag_associated_at": schema.StringAttribute{
			Description: "Date of the last attach or detach on the resource, null while no tag was ever attached. " +
				"The `mgc_tag_attachment` resource does not expose this, because it changes whenever any tag moves " +
				"and would show up as permanent drift.",
			Computed: true,
		},
	}
}

// convertAttachment maps a tagged resource of the API onto the model. A data
// source has no previous configuration to preserve, so this is a plain
// conversion, unlike flattenAttachment on the resource.
func convertAttachment(taggedResource tagSDK.Resource) tagAttachmentModel {
	return tagAttachmentModel{
		// The resource is identified by the id its own product uses for it, which
		// is what every other resource of the provider knows it by. The id the tags
		// service keeps for its own bookkeeping is left out on purpose.
		ID:           types.StringValue(taggedResource.ExternalID),
		ResourceID:   types.StringValue(taggedResource.ExternalID),
		Tags:         mirrorTags(taggedResource),
		ResourceType: types.StringValue(string(taggedResource.ResourceType.Name)),
		Product:      types.StringValue(string(taggedResource.ResourceType.Product)),
		Region:       types.StringValue(taggedResource.Region),

		// The timestamp type of the SDK lives in an internal package and cannot be
		// named here, but it converts to time.Time.
		CreatedAt:           types.StringPointerValue(utils.ConvertTimeToRFC3339((*time.Time)(&taggedResource.CreatedAt))),
		UpdatedAt:           types.StringPointerValue(utils.ConvertTimeToRFC3339((*time.Time)(taggedResource.UpdatedAt))),
		LastTagAssociatedAt: types.StringPointerValue(utils.ConvertTimeToRFC3339((*time.Time)(taggedResource.LastTagAssociatedAt))),
	}
}
