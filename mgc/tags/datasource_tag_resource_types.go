package tags

//go:generate go run github.com/vektra/mockery/v2@v2.53.6 --name=ResourceTypeService --srcpkg=github.com/MagaluCloud/mgc-sdk-go/tag --output=../internal/mocks --outpkg=mocks

import (
	"context"
	"time"

	tagSDK "github.com/MagaluCloud/mgc-sdk-go/tag"
	"github.com/MagaluCloud/terraform-provider-mgc/mgc/utils"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &tagResourceTypesDataSource{}

type tagResourceTypeModel struct {
	Name      types.String `tfsdk:"name"`
	Product   types.String `tfsdk:"product"`
	CreatedAt types.String `tfsdk:"created_at"`
	UpdatedAt types.String `tfsdk:"updated_at"`
}

type tagResourceTypesModel struct {
	Product       types.String           `tfsdk:"product"`
	ResourceTypes []tagResourceTypeModel `tfsdk:"resource_types"`
}

type tagResourceTypesDataSource struct {
	resourceTypes tagSDK.ResourceTypeService
}

func NewTagResourceTypesDataSource() datasource.DataSource {
	return &tagResourceTypesDataSource{}
}

func (d *tagResourceTypesDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_tag_resource_types"
}

func (d *tagResourceTypesDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	dataConfig, ok := req.ProviderData.(utils.DataConfig)
	if !ok {
		resp.Diagnostics.AddError("Failed to configure data source", "Invalid provider data")
		return
	}

	d.resourceTypes = newTagClient(dataConfig).ResourceTypes()
}

func (d *tagResourceTypesDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Reads the types of resource that can be tagged, and the product that owns each one. " +
			"The list grows as products adopt tagging, so this is what tells whether a resource can carry tags at all.",
		Attributes: map[string]schema.Attribute{
			"product": schema.StringAttribute{
				Description: "Only return the types owned by this product, such as `network`.",
				Optional:    true,
			},
			"resource_types": schema.ListNestedAttribute{
				Description: "The resource types found, in the order the API returned them.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							Description: "Name of the type, prefixed by the product that owns it, as in `net.vpc`. " +
								"This is what the `resource_type` attribute of a tag attachment reports.",
							Computed: true,
						},
						"product": schema.StringAttribute{
							Description: "Product that owns the type.",
							Computed:    true,
						},
						"created_at": schema.StringAttribute{
							Description: "Date the type became known to the tags service.",
							Computed:    true,
						},
						"updated_at": schema.StringAttribute{
							Description: "Last update date of the type, null while it was never updated.",
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

func (d *tagResourceTypesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data tagResourceTypesModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	options := tagSDK.ListResourceTypesOptions{
		Product: namedStringPointer[tagSDK.Product](data.Product),
	}

	resourceTypes, err := listAllPages(func(limit, offset int) ([]tagSDK.ResourceType, error) {
		options.Limit, options.Offset = &limit, &offset
		return d.resourceTypes.List(ctx, options)
	})
	if err != nil {
		resp.Diagnostics.AddError(utils.ParseSDKError(err))
		return
	}

	data.ResourceTypes = make([]tagResourceTypeModel, 0, len(resourceTypes))
	for _, resourceType := range resourceTypes {
		data.ResourceTypes = append(data.ResourceTypes, convertResourceType(resourceType))
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// convertResourceType maps a resource type of the API onto the model.
func convertResourceType(resourceType tagSDK.ResourceType) tagResourceTypeModel {
	return tagResourceTypeModel{
		Name:    types.StringValue(string(resourceType.Name)),
		Product: types.StringValue(string(resourceType.Product)),

		// The timestamp type of the SDK lives in an internal package and cannot be
		// named here, but it converts to time.Time.
		CreatedAt: types.StringPointerValue(utils.ConvertTimeToRFC3339((*time.Time)(&resourceType.CreatedAt))),
		UpdatedAt: types.StringPointerValue(utils.ConvertTimeToRFC3339((*time.Time)(resourceType.UpdatedAt))),
	}
}
