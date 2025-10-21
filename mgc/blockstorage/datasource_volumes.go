package blockstorage

import (
	"context"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"

	bsSDK "github.com/MagaluCloud/mgc-sdk-go/blockstorage"
	"github.com/MagaluCloud/terraform-provider-mgc/mgc/utils"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &DataSourceBsVolumes{}

type DataSourceBsVolumes struct {
	bsVolumes bsSDK.VolumeService
}

type bsVolumesDataSourceItemModel struct {
	ID               types.String `tfsdk:"id"`
	Name             types.String `tfsdk:"name"`
	AvailabilityZone types.String `tfsdk:"availability_zone"`
	UpdatedAt        types.String `tfsdk:"updated_at"`
	CreatedAt        types.String `tfsdk:"created_at"`
	Size             types.Int64  `tfsdk:"size"`
	TypeId           types.String `tfsdk:"type_id"`
	State            types.String `tfsdk:"state"`
	Status           types.String `tfsdk:"status"`
	Encrypted        types.Bool   `tfsdk:"encrypted"`
}

func NewDataSourceBsVolumes() datasource.DataSource {
	return &DataSourceBsVolumes{}
}

func (r *DataSourceBsVolumes) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_block_storage_volumes"
}

type bsVolumesDataSourceModel struct {
	Volumes []bsVolumesDataSourceItemModel `tfsdk:"volumes"`
}

func (r *DataSourceBsVolumes) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	dataConfig, ok := req.ProviderData.(utils.DataConfig)
	if !ok {
		resp.Diagnostics.AddError("Failed to configure data source", "Invalid provider data")
		return
	}

	r.bsVolumes = bsSDK.New(&dataConfig.CoreConfig).Volumes()
}

func (r *DataSourceBsVolumes) Schema(_ context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"volumes": schema.ListNestedAttribute{
				Computed:    true,
				Description: "List of available Block Storage Volumes.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Description: "The unique identifier of the volume snapshot.",
							Computed:    true,
						},
						"name": schema.StringAttribute{
							Description: "The name of the block storage.",
							Computed:    true,
						},
						"availability_zone": schema.StringAttribute{
							Description: "The availability zones where the block storage is available.",
							Computed:    true,
						},
						"size": schema.Int64Attribute{
							Description: "The size of the block storage in GB.",
							Computed:    true,
						},
						"updated_at": schema.StringAttribute{
							Description: "The timestamp when the block storage was last updated.",
							Computed:    true,
						},
						"created_at": schema.StringAttribute{
							Description: "The timestamp when the block storage was created.",
							Computed:    true,
						},
						"state": schema.StringAttribute{
							Description: "The current state of the virtual machine instance.",
							Computed:    true,
						},
						"status": schema.StringAttribute{
							Description: "The status of the virtual machine instance.",
							Computed:    true,
						},
						"type_id": schema.StringAttribute{
							Description: "The unique identifier of the block storage type.",
							Computed:    true,
						},
						"encrypted": schema.BoolAttribute{
							Description: "The encryption status of the block storage.",
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

func (r *DataSourceBsVolumes) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data bsVolumesDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	sdkOutputList, err := r.bsVolumes.ListAll(ctx, bsSDK.VolumeFilterOptions{})
	if err != nil {
		resp.Diagnostics.AddError(utils.ParseSDKError(err))
		return
	}

	for _, sdkOutput := range sdkOutputList {

		var item bsVolumesDataSourceItemModel
		item.ID = types.StringValue(sdkOutput.ID)
		item.Name = types.StringValue(sdkOutput.Name)
		item.AvailabilityZone = types.StringValue(sdkOutput.AvailabilityZone)
		item.UpdatedAt = types.StringValue(sdkOutput.UpdatedAt.Format(time.RFC3339))
		item.CreatedAt = types.StringValue(sdkOutput.CreatedAt.Format(time.RFC3339))
		item.Size = types.Int64Value(int64(sdkOutput.Size))
		item.TypeId = types.StringValue(sdkOutput.Type.ID)
		item.State = types.StringValue(sdkOutput.State)
		item.Status = types.StringValue(sdkOutput.Status)
		item.Encrypted = types.BoolPointerValue(sdkOutput.Encrypted)

		data.Volumes = append(data.Volumes, item)
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
