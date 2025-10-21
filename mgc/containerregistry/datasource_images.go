package containerregistry

import (
	"context"

	crSDK "github.com/MagaluCloud/mgc-sdk-go/containerregistry"
	"github.com/MagaluCloud/terraform-provider-mgc/mgc/utils"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &DataSourceCRImages{}

type DataSourceCRImages struct {
	crImages crSDK.ImagesService
}

type crImage struct {
	Digest    types.String   `tfsdk:"digest"`
	SizeBytes types.Int64    `tfsdk:"size_bytes"`
	PulledAt  types.String   `tfsdk:"pullet_at"`
	PushedAt  types.String   `tfsdk:"pushed_at"`
	Tags      []types.String `tfsdk:"tags"`
}

type crImagesList struct {
	RegistryID     types.String `tfsdk:"registry_id"`
	RepositoryName types.String `tfsdk:"repository_name"`
	Images         []crImage    `tfsdk:"images"`
}

func NewDataSourceCRImages() datasource.DataSource {
	return &DataSourceCRImages{}
}

func (r *DataSourceCRImages) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_container_images"
}

func (r *DataSourceCRImages) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	dataConfig, ok := req.ProviderData.(utils.DataConfig)
	if !ok {
		resp.Diagnostics.AddError("Failed to configure data source", "Invalid provider data")
		return
	}

	r.crImages = crSDK.New(&dataConfig.CoreConfig).Images()
}

func (r *DataSourceCRImages) Schema(_ context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Data source for Container Registry Images",
		Attributes: map[string]schema.Attribute{
			"repository_name": schema.StringAttribute{
				Description: "Name of the repository",
				Required:    true,
			},
			"registry_id": schema.StringAttribute{
				Description: "ID of the registry",
				Required:    true,
			},
			"images": schema.ListNestedAttribute{
				Description: "List of container images",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"digest": schema.StringAttribute{
							Description: "The digest hash of the image",
							Computed:    true,
						},
						"size_bytes": schema.Int64Attribute{
							Description: "The size of the image in bytes",
							Computed:    true,
						},
						"pullet_at": schema.StringAttribute{
							Description: "The timestamp when the image was last pulled",
							Computed:    true,
						},
						"pushed_at": schema.StringAttribute{
							Description: "The timestamp when the image was pushed",
							Computed:    true,
						},
						"tags": schema.ListAttribute{
							Description: "List of tags associated with the image",
							Computed:    true,
							ElementType: types.StringType,
						},
					},
				},
			},
		},
	}
}

func (r *DataSourceCRImages) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data crImagesList

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	sdkOutputList, err := r.crImages.ListAll(ctx, data.RegistryID.ValueString(), data.RepositoryName.ValueString(), crSDK.ImageFilterOptions{})
	if err != nil {
		resp.Diagnostics.AddError(utils.ParseSDKError(err))
		return
	}

	for _, sdkOutput := range sdkOutputList {
		var item crImage

		item.Digest = types.StringValue(sdkOutput.Digest)
		item.PulledAt = types.StringValue(sdkOutput.PulledAt)
		item.PushedAt = types.StringValue(sdkOutput.PushedAt)
		item.SizeBytes = types.Int64Value(int64(sdkOutput.SizeBytes))

		for _, tag := range sdkOutput.Tags {
			item.Tags = append(item.Tags, types.StringValue(tag))
		}

		data.Images = append(data.Images, item)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
