package blockstorage

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	storageSDK "github.com/MagaluCloud/mgc-sdk-go/blockstorage"
	"github.com/MagaluCloud/terraform-provider-mgc/mgc/utils"
)

var _ datasource.DataSource = &DataSourceBsSchedule{}

type DataSourceBsSchedule struct {
	bsScheduler storageSDK.SchedulerService
}

func NewDataSourceBSSchedule() datasource.DataSource {
	return &DataSourceBsSchedule{}
}

func (r *DataSourceBsSchedule) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_block_storage_schedule"
}

type bsScheduleDataSourceModel struct {
	ID                    types.String `tfsdk:"id"`
	Name                  types.String `tfsdk:"name"`
	Description           types.String `tfsdk:"description"`
	Volumes               types.List   `tfsdk:"volumes"`
	SnapshotType          types.String `tfsdk:"snapshot_type"`
	PolicyRetentionInDays types.Int64  `tfsdk:"policy_retention_in_days"`
	PolicyFrequencyDaily  types.String `tfsdk:"policy_frequency_daily_start_time"`
	State                 types.String `tfsdk:"state"`
	CreatedAt             types.String `tfsdk:"created_at"`
	UpdatedAt             types.String `tfsdk:"updated_at"`
}

func (r *DataSourceBsSchedule) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	dataConfig, ok := req.ProviderData.(utils.DataConfig)
	if !ok {
		resp.Diagnostics.AddError("Failed to configure data source", "Invalid provider data")
		return
	}

	client := storageSDK.New(&dataConfig.CoreConfig)
	r.bsScheduler = client.Schedulers()
}

func GetBsScheduleAttributes(idRequired bool) map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"id": schema.StringAttribute{
			Description: "The unique identifier of the snapshot schedule.",
			Required:    idRequired,
			Computed:    !idRequired,
		},
		"name": schema.StringAttribute{
			Description: "The name of the snapshot schedule.",
			Computed:    true,
		},
		"description": schema.StringAttribute{
			Description: "The description of the snapshot schedule.",
			Computed:    true,
		},
		"volumes": schema.ListAttribute{
			Description: "List of block storage volume IDs.",
			ElementType: types.StringType,
			Computed:    true,
		},
		"snapshot_type": schema.StringAttribute{
			Description: "Type of snapshot to create.",
			Computed:    true,
		},
		"policy_retention_in_days": schema.Int64Attribute{
			Description: "Number of days to retain snapshots.",
			Computed:    true,
		},
		"policy_frequency_daily_start_time": schema.StringAttribute{
			Description: "Start time for daily snapshot creation (HH:MM:SS format).",
			Computed:    true,
		},
		"state": schema.StringAttribute{
			Description: "The current state of the schedule.",
			Computed:    true,
		},
		"created_at": schema.StringAttribute{
			Description: "The timestamp when the schedule was created.",
			Computed:    true,
		},
		"updated_at": schema.StringAttribute{
			Description: "The timestamp when the schedule was last updated.",
			Computed:    true,
		},
	}
}

func (r *DataSourceBsSchedule) Schema(_ context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	description := "Block storage snapshot schedule"
	resp.Schema = schema.Schema{
		Description:         description,
		MarkdownDescription: description,
		Attributes:          GetBsScheduleAttributes(true),
	}
}

func (r *DataSourceBsSchedule) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data bsScheduleDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	get, err := r.bsScheduler.Get(ctx, data.ID.ValueString(), []storageSDK.ExpandSchedulers{storageSDK.ExpandSchedulersVolume})
	if err != nil {
		resp.Diagnostics.AddError(utils.ParseSDKError(err))
		return
	}

	model := SchedulerResponseToModel(get)
	data.ID = model.ID
	data.Name = model.Name
	data.Description = model.Description
	data.Volumes = model.Volumes
	data.SnapshotType = model.SnapshotType
	data.PolicyRetentionInDays = model.PolicyRetentionInDays
	data.PolicyFrequencyDaily = model.PolicyFrequencyDaily
	data.State = model.State
	data.CreatedAt = model.CreatedAt
	data.UpdatedAt = model.UpdatedAt

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
