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

var _ datasource.DataSource = &DataSourceBsSnapshot{}

type DataSourceBsSnapshot struct {
	bsSnapshotService bsSDK.SnapshotService
}

func NewDataSourceBSSnapshot() datasource.DataSource {
	return &DataSourceBsSnapshot{}
}

func (r *DataSourceBsSnapshot) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_block_storage_snapshot"
}

type bsSnapshotsResourceModel struct {
	ID                types.String `tfsdk:"id"`
	Name              types.String `tfsdk:"name"`
	Description       types.String `tfsdk:"description"`
	UpdatedAt         types.String `tfsdk:"updated_at"`
	CreatedAt         types.String `tfsdk:"created_at"`
	VolumeId          types.String `tfsdk:"volume_id"`
	State             types.String `tfsdk:"state"`
	Status            types.String `tfsdk:"status"`
	Size              types.Int64  `tfsdk:"size"`
	Type              types.String `tfsdk:"type"`
	AvailabilityZones types.List   `tfsdk:"availability_zones"`
}

func (r *DataSourceBsSnapshot) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	dataConfig, ok := req.ProviderData.(tfutil.DataConfig)

	if !ok {
		resp.Diagnostics.AddError("Failed to configure data source", "Invalid provider data")
		return
	}

	r.bsSnapshotService = bsSDK.New(&dataConfig.CoreConfig).Snapshots()
}

func GetBsSnapshotAttributes(idRequired bool) map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"id": schema.StringAttribute{
			Description: "The unique identifier of the volume snapshot.",
			Required:    idRequired,
			Computed:    !idRequired,
		},
		"name": schema.StringAttribute{
			Description: "The name of the volume snapshot.",
			Computed:    true,
		},
		"description": schema.StringAttribute{
			Description: "The description of the volume snapshot.",
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
		"size": schema.Int64Attribute{
			Description: "The size of the snapshot in GB.",
			Computed:    true,
		},
		"volume_id": schema.StringAttribute{
			Description: "ID of block storage volume",
			Computed:    true,
		},
		"type": schema.StringAttribute{
			Description: "The type of the snapshot.",
			Computed:    true,
		},
		"availability_zones": schema.ListAttribute{
			Description: "The availability zones of the snapshot.",
			Computed:    true,
			ElementType: types.StringType,
		},
	}
}

func (r *DataSourceBsSnapshot) Schema(_ context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	description := "Block storage snapshots"
	resp.Schema = schema.Schema{
		Description:         description,
		MarkdownDescription: description,
		Attributes:          GetBsSnapshotAttributes(true),
	}
}

func (r *DataSourceBsSnapshot) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data bsSnapshotsResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	sdkOutput, err := r.bsSnapshotService.Get(ctx, data.ID.ValueString(), []string{})
	if err != nil {
		resp.Diagnostics.AddError(tfutil.ParseSDKError(err))
		return
	}

	list, diags := types.ListValueFrom(ctx, types.StringType, sdkOutput.AvailabilityZones)
	resp.Diagnostics.Append(diags...)

	data.ID = types.StringValue(sdkOutput.ID)
	data.Name = types.StringValue(sdkOutput.Name)
	data.Description = types.StringValue(sdkOutput.Description)
	data.UpdatedAt = types.StringValue(sdkOutput.UpdatedAt.Format(time.RFC3339))
	data.CreatedAt = types.StringValue(sdkOutput.CreatedAt.Format(time.RFC3339))
	data.VolumeId = types.StringPointerValue(sdkOutput.Volume.ID)
	data.State = types.StringValue(string(sdkOutput.State))
	data.Status = types.StringValue(string(sdkOutput.Status))
	data.Size = types.Int64Value(int64(sdkOutput.Size))
	data.Type = types.StringValue(sdkOutput.Type)
	data.AvailabilityZones = list

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
