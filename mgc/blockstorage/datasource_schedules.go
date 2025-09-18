package blockstorage

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"

	storageSDK "github.com/MagaluCloud/mgc-sdk-go/blockstorage"
	"github.com/MagaluCloud/terraform-provider-mgc/mgc/utils"
)

var _ datasource.DataSource = &DataSourceBsSchedules{}

type DataSourceBsSchedules struct {
	bsScheduler storageSDK.SchedulerService
}

func NewDataSourceBSSchedules() datasource.DataSource {
	return &DataSourceBsSchedules{}
}

func (r *DataSourceBsSchedules) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_block_storage_schedules"
}

type bsSchedulesDataSourceModel struct {
	Schedules []bsScheduleDataSourceModel `tfsdk:"schedules"`
}

func (r *DataSourceBsSchedules) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (r *DataSourceBsSchedules) Schema(_ context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	description := "Block storage snapshot schedules"
	resp.Schema = schema.Schema{
		Description:         description,
		MarkdownDescription: description,
		Attributes: map[string]schema.Attribute{
			"schedules": schema.ListNestedAttribute{
				Description: "List of snapshot schedules.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: GetBsScheduleAttributes(false),
				},
			},
		},
	}
}

func (r *DataSourceBsSchedules) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data bsSchedulesDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	schedulesResponse, err := r.bsScheduler.List(ctx, storageSDK.SchedulerListOptions{
		Expand: []storageSDK.ExpandSchedulers{storageSDK.ExpandSchedulersVolume},
	})
	if err != nil {
		resp.Diagnostics.AddError(utils.ParseSDKError(err))
		return
	}

	var schedulesData []bsScheduleDataSourceModel
	for _, schedule := range schedulesResponse.Schedulers {
		model := SchedulerResponseToModel(&schedule)

		scheduleModel := bsScheduleDataSourceModel{
			ID:                    model.ID,
			Name:                  model.Name,
			Description:           model.Description,
			Volumes:               model.Volumes,
			SnapshotType:          model.SnapshotType,
			PolicyRetentionInDays: model.PolicyRetentionInDays,
			PolicyFrequencyDaily:  model.PolicyFrequencyDaily,
			CreatedAt:             model.CreatedAt,
			UpdatedAt:             model.UpdatedAt,
			State:                 model.State,
		}
		schedulesData = append(schedulesData, scheduleModel)
	}

	data.Schedules = schedulesData

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
