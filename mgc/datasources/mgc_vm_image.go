package datasources

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"

	vmSDK "github.com/MagaluCloud/mgc-sdk-go/compute"
	"github.com/MagaluCloud/terraform-provider-mgc/mgc/tfutil"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &DataSourceVmImages{}

type DataSourceVmImages struct {
	vmImageService vmSDK.ImageService
}

type ImageModel struct {
	ID                types.String   `tfsdk:"id"`
	Name              types.String   `tfsdk:"name"`
	Platform          types.String   `tfsdk:"platform"`
	AvailabilityZones []types.String `tfsdk:"availability_zones"`
	MinimumDiskSize   types.Int64    `tfsdk:"minimum_disk_size"`
	MinimumMemorySize types.Int64    `tfsdk:"minimum_memory_size"`
	MinimumVCPU       types.Int64    `tfsdk:"minimum_vcpus"`
}

type ImagesModel struct {
	Images []ImageModel `tfsdk:"images"`
}

func NewDataSourceVMIMages() datasource.DataSource {
	return &DataSourceVmImages{}
}

func (r *DataSourceVmImages) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_virtual_machine_images"
}

func (r *DataSourceVmImages) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	dataConfig, ok := req.ProviderData.(tfutil.DataConfig)
	if !ok {
		resp.Diagnostics.AddError("Failed to get provider data", "Failed to get provider data")
		return
	}

	r.vmImageService = vmSDK.New(&dataConfig.CoreConfig).Images()
}

func (r *DataSourceVmImages) Schema(_ context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"images": schema.ListNestedAttribute{
				Computed:    true,
				Description: "List of available VM Images.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Computed:    true,
							Description: "ID of image.",
						},
						"platform": schema.StringAttribute{
							Computed:    true,
							Description: "The image platform.",
						},
						"name": schema.StringAttribute{
							Computed:    true,
							Description: "The image name.",
						},
						"availability_zones": schema.ListAttribute{
							Computed:    true,
							Description: "The availability zones of the image.",
							ElementType: types.StringType,
						},
						"minimum_disk_size": schema.NumberAttribute{
							Computed:    true,
							Description: "The minimum disk size of the image.",
						},
						"minimum_memory_size": schema.NumberAttribute{
							Computed:    true,
							Description: "The minimum memory size of the image.",
						},
						"minimum_vcpus": schema.NumberAttribute{
							Computed:    true,
							Description: "The minimum vcpus of the image.",
						},
					},
				},
			},
		},
	}
	resp.Schema.Description = "Get the available virtual-machine images."
}

func (r *DataSourceVmImages) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data ImagesModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	sdkOutput, err := r.vmImageService.List(ctx, vmSDK.ImageListOptions{})
	if err != nil {
		resp.Diagnostics.AddError(tfutil.ParseSDKError(err))
		return
	}

	for _, image := range sdkOutput {
		if image.Status != vmSDK.ImageStatusActive {
			continue
		}

		platform := ""
		if image.Platform != nil {
			platform = *image.Platform
		}

		var azs []types.String
		for _, az := range image.AvailabilityZones {
			azs = append(azs, types.StringValue(az))
		}

		data.Images = append(data.Images, ImageModel{
			ID:                types.StringValue(image.ID),
			Name:              types.StringValue(image.Name),
			Platform:          types.StringValue(platform),
			AvailabilityZones: azs,
			MinimumDiskSize:   types.Int64Value(int64(image.MinimumRequirements.Disk)),
			MinimumMemorySize: types.Int64Value(int64(image.MinimumRequirements.RAM)),
			MinimumVCPU:       types.Int64Value(int64(image.MinimumRequirements.VCPU)),
		})

	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
