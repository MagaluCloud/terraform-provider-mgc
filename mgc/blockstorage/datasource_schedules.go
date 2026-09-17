package blockstorage

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	storageSDK "github.com/MagaluCloud/mgc-sdk-go/blockstorage"
	"github.com/MagaluCloud/terraform-provider-mgc/mgc/internal/project"
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
	ProjectID types.String                `tfsdk:"project_id"`
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

	client := storageSDK.New(dataConfig.CoreFor(utils.ServiceBlockStorage))
	r.bsScheduler = client.Schedulers()
}

func (r *DataSourceBsSchedules) Schema(_ context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	description := "Block storage snapshot schedules"
	resp.Schema = schema.Schema{
		Description:         description,
		MarkdownDescription: description,
		Attributes: map[string]schema.Attribute{
			"project_id": project.DataSourceAttribute(),
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

	schedulesResponse, err := r.bsScheduler.ListAll(ctx, project.CallOptions(data.ProjectID), storageSDK.SchedulerFilterOptions{
		Expand: []storageSDK.ExpandSchedulers{storageSDK.ExpandSchedulersVolume},
	})
	if err != nil {
		resp.Diagnostics.AddError(project.ParseSDKError(err, data.ProjectID))
		return
	}

	var schedulesData []bsScheduleDataSourceModel
	for _, schedule := range schedulesResponse {
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
