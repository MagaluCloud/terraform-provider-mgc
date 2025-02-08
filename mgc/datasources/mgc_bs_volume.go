package datasources

import (
	"context"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"

	bsSDK "github.com/MagaluCloud/mgc-sdk-go/blockstorage"
	"github.com/MagaluCloud/terraform-provider-mgc/mgc/tfutil"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &DataSourceBsVolume{}

type DataSourceBsVolume struct {
	bsVolume bsSDK.VolumeService
}

func NewDataSourceBsVolume() datasource.DataSource {
	return &DataSourceBsVolume{}
}

func (r *DataSourceBsVolume) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_block_storage_volume"
}

type bsVolumeResourceModel struct {
	ID                   types.String `tfsdk:"id"`
	Name                 types.String `tfsdk:"name"`
	AvailabilityZone     types.String `tfsdk:"availability_zone"`
	UpdatedAt            types.String `tfsdk:"updated_at"`
	CreatedAt            types.String `tfsdk:"created_at"`
	Size                 types.Int64  `tfsdk:"size"`
	State                types.String `tfsdk:"state"`
	Status               types.String `tfsdk:"status"`
	TypeName             types.String `tfsdk:"type_name"`
	DiskType             types.String `tfsdk:"disk_type"`
	TypeId               types.String `tfsdk:"type_id"`
	TypeStatus           types.String `tfsdk:"type_status"`
	AttachedAt           types.String `tfsdk:"attached_at"`
	AttachedDevice       types.String `tfsdk:"attached_device"`
	AttachedInstanceId   types.String `tfsdk:"attached_instance_id"`
	AttachedInstanceName types.String `tfsdk:"attached_instance_name"`
	Encrypted            types.Bool   `tfsdk:"encrypted"`
}

func (r *DataSourceBsVolume) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	dataConfig, ok := req.ProviderData.(tfutil.DataConfig)
	if !ok {
		resp.Diagnostics.AddError("Failed to configure data source", "Invalid provider data")
		return
	}

	r.bsVolume = bsSDK.New(&dataConfig.CoreConfig).Volumes()
}

func (r *DataSourceBsVolume) Schema(_ context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Block storage volume",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The unique identifier of the volume snapshot.",
				Required:    true,
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
			"type_name": schema.StringAttribute{
				Description: "The name of the block storage type.",
				Computed:    true,
			},
			"disk_type": schema.StringAttribute{
				Description: "The disk type of the block storage.",
				Computed:    true,
			},
			"type_id": schema.StringAttribute{
				Description: "The unique identifier of the block storage type.",
				Computed:    true,
			},
			"type_status": schema.StringAttribute{
				Description: "The status of the block storage type.",
				Computed:    true,
			},
			"attached_at": schema.StringAttribute{
				Description: "The timestamp when the block storage was attached.",
				Computed:    true,
			},
			"attached_device": schema.StringAttribute{
				Description: "The device path of the attachment.",
				Computed:    true,
			},
			"attached_instance_id": schema.StringAttribute{
				Description: "The unique identifier of the instance.",
				Computed:    true,
			},
			"attached_instance_name": schema.StringAttribute{
				Description: "The name of the instance.",
				Computed:    true,
			},
			"encrypted": schema.BoolAttribute{
				Description: "The encryption status of the block storage.",
				Computed:    true,
			},
		},
	}
}

func (r *DataSourceBsVolume) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data bsVolumeResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	sdkOutput, err := r.bsVolume.Get(ctx, data.ID.ValueString(), []string{"volume_type", "attachment"})
	if err != nil {
		resp.Diagnostics.AddError("Failed to get versions", err.Error())
		return
	}

	data.ID = types.StringValue(sdkOutput.ID)
	data.Name = types.StringValue(sdkOutput.Name)
	data.AvailabilityZone = types.StringValue(sdkOutput.AvailabilityZone)
	data.UpdatedAt = types.StringValue(sdkOutput.UpdatedAt.Format(time.RFC3339))
	data.CreatedAt = types.StringValue(sdkOutput.CreatedAt.Format(time.RFC3339))
	data.Size = types.Int64Value(int64(sdkOutput.Size))
	data.TypeName = types.StringPointerValue(sdkOutput.Type.Name)
	data.DiskType = types.StringPointerValue(sdkOutput.Type.DiskType)
	data.TypeId = types.StringValue(sdkOutput.Type.ID)
	data.TypeStatus = types.StringPointerValue(sdkOutput.Type.Status)
	data.State = types.StringValue(sdkOutput.State)
	data.Status = types.StringValue(sdkOutput.Status)
	data.Encrypted = types.BoolValue(sdkOutput.Encrypted)
	if sdkOutput.Attachment != nil {
		data.AttachedAt = types.StringValue(sdkOutput.Attachment.AttachedAt.Format(time.RFC3339))
		data.AttachedDevice = types.StringPointerValue(sdkOutput.Attachment.Device)
		data.AttachedInstanceId = types.StringValue(sdkOutput.Attachment.Instance.ID)
		data.AttachedInstanceName = types.StringValue(sdkOutput.Attachment.Instance.Name)
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
