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

var _ datasource.DataSource = &DataSourceBsSnapshots{}

type DataSourceBsSnapshots struct {
	bsSnapshotService bsSDK.SnapshotService
}

func NewDataSourceBSSnapshots() datasource.DataSource {
	return &DataSourceBsSnapshots{}
}

func (r *DataSourceBsSnapshots) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_block_storage_snapshots"
}

type bsSnapshotsListDataSourceModel struct {
	Snapshots []bsSnapshotsDataSourceItemModel `tfsdk:"snapshots"`
}

type bsSnapshotsDataSourceItemModel struct {
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

func (r *DataSourceBsSnapshots) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	dataConfig, ok := req.ProviderData.(utils.DataConfig)

	if !ok {
		resp.Diagnostics.AddError("Failed to configure data source", "Invalid provider data")
		return
	}

	r.bsSnapshotService = bsSDK.New(&dataConfig.CoreConfig).Snapshots()
}

func (r *DataSourceBsSnapshots) Schema(_ context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"snapshots": schema.ListNestedAttribute{
				Computed:    true,
				Description: "List of available Block Storage Snapshots.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: GetBsSnapshotAttributes(false),
				},
			},
		},
	}
}

func (r *DataSourceBsSnapshots) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data bsSnapshotsListDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	sdkOutputList, err := r.bsSnapshotService.ListAll(ctx, bsSDK.SnapshotFilterOptions{})
	if err != nil {
		resp.Diagnostics.AddError(utils.ParseSDKError(err))
		return
	}

	for _, sdkOutput := range sdkOutputList {
		list, diags := types.ListValueFrom(ctx, types.StringType, sdkOutput.AvailabilityZones)
		resp.Diagnostics.Append(diags...)

		var item bsSnapshotsDataSourceItemModel

		item.ID = types.StringValue(sdkOutput.ID)
		item.Name = types.StringValue(sdkOutput.Name)
		item.Description = types.StringPointerValue(sdkOutput.Description)
		item.UpdatedAt = types.StringValue(sdkOutput.UpdatedAt.Format(time.RFC3339))
		item.CreatedAt = types.StringValue(sdkOutput.CreatedAt.Format(time.RFC3339))
		item.VolumeId = types.StringPointerValue(sdkOutput.Volume.ID)
		item.State = types.StringValue(string(sdkOutput.State))
		item.Status = types.StringValue(string(sdkOutput.Status))
		item.Size = types.Int64Value(int64(sdkOutput.Size))
		item.Type = types.StringValue(sdkOutput.Type)
		item.AvailabilityZones = list

		data.Snapshots = append(data.Snapshots, item)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
