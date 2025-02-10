package datasources

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"

	bsSDK "github.com/MagaluCloud/mgc-sdk-go/blockstorage"
	"github.com/MagaluCloud/terraform-provider-mgc/mgc/tfutil"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &DataSourceBsVolumeTypes{}

type DataSourceBsVolumeTypes struct {
	bsVolumeTypes bsSDK.VolumeTypeService
}

func NewDataSourceBsVolumeTypes() datasource.DataSource {
	return &DataSourceBsVolumeTypes{}
}

func (r *DataSourceBsVolumeTypes) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_block_storage_volume_types"
}

type volumeTypes struct {
	VolumeTypes []volumeType `tfsdk:"volume_types"`
}

type volumeType struct {
	AvailabilityZones types.List   `tfsdk:"availability_zones"`
	DiskType          types.String `tfsdk:"disk_type"`
	Id                types.String `tfsdk:"id"`
	Iops              types.Int64  `tfsdk:"iops"`
	Name              types.String `tfsdk:"name"`
	Status            types.String `tfsdk:"status"`
}

func (r *DataSourceBsVolumeTypes) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	dataConfig, ok := req.ProviderData.(tfutil.DataConfig)

	if !ok {
		resp.Diagnostics.AddError("Failed to configure data source", "Invalid provider data")
		return
	}

	r.bsVolumeTypes = bsSDK.New(&dataConfig.CoreConfig).VolumeTypes()
}

func (r *DataSourceBsVolumeTypes) Schema(_ context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Block-storage Volume Types",
		Attributes: map[string]schema.Attribute{
			"volume_types": schema.ListNestedAttribute{
				Computed:    true,
				Description: "List of available Block-storage Volume Types.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Computed:    true,
							Description: "ID of image.",
						},
						"disk_type": schema.StringAttribute{
							Computed:    true,
							Description: "The disk type.",
						},
						"name": schema.StringAttribute{
							Computed:    true,
							Description: "The volume type name.",
						},
						"status": schema.StringAttribute{
							Computed:    true,
							Description: "The volume type status.",
						},
						"iops": schema.Int64Attribute{
							Computed:    true,
							Description: "The volume type IOPS.",
						},
						"availability_zones": schema.ListAttribute{
							Computed:    true,
							Description: "The volume type availability zones.",
							ElementType: types.StringType,
						},
					},
				},
			},
		},
	}
}

func (r *DataSourceBsVolumeTypes) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data volumeTypes

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	sdkOutput, err := r.bsVolumeTypes.List(ctx, bsSDK.ListVolumeTypesOptions{ /*todo*/ })
	if err != nil {
		resp.Diagnostics.AddError(tfutil.ParseSDKError(err))
		return
	}

	for _, stype := range sdkOutput {
		list, diags := types.ListValueFrom(ctx, types.StringType, stype.AvailabilityZones)
		resp.Diagnostics.Append(diags...)

		data.VolumeTypes = append(data.VolumeTypes, volumeType{
			AvailabilityZones: list,
			DiskType:          types.StringValue(stype.DiskType),
			Id:                types.StringValue(stype.ID),
			Iops:              types.Int64Value(int64(stype.IOPS.Total)),
			Name:              types.StringValue(stype.Name),
			Status:            types.StringValue(stype.Status),
		})

	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
