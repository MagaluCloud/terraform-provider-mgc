package blockstorage

import (
	"context"
	"regexp"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	storageSDK "github.com/MagaluCloud/mgc-sdk-go/blockstorage"
	"github.com/MagaluCloud/terraform-provider-mgc/mgc/utils"
)

func NewBlockStorageScheduleResource() resource.Resource {
	return &bsSchedule{}
}

type bsSchedule struct {
	bsScheduler storageSDK.SchedulerService
}

func (r *bsSchedule) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_block_storage_schedule"
}

func (r *bsSchedule) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	dataConfig, ok := req.ProviderData.(utils.DataConfig)
	if !ok {
		resp.Diagnostics.AddError("Failed to get provider data", "Failed to get provider data")
		return
	}

	client := storageSDK.New(&dataConfig.CoreConfig)
	r.bsScheduler = client.Schedulers()
}

type bsScheduleResourceModel struct {
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

func (r *bsSchedule) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "The block storage schedule resource allows you to manage automatic snapshot schedules in the Magalu Cloud.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The unique identifier of the snapshot schedule.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
				Computed: true,
			},
			"name": schema.StringAttribute{
				Description: "The name of the snapshot schedule.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 255),
					stringvalidator.RegexMatches(
						regexp.MustCompile(`^[a-z0-9]+(?:[-_][a-z0-9]+)*$`),
						"The name must contain only lowercase letters, numbers, underlines and hyphens. Hyphens and underlines cannot be located at the edges either.",
					),
				},
				Required: true,
			},
			"description": schema.StringAttribute{
				Description: "The description of the snapshot schedule.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 255),
				},
				Optional: true,
			},
			"volumes": schema.ListAttribute{
				Description: "List of block storage volume IDs to create snapshots for.",
				PlanModifiers: []planmodifier.List{
					listplanmodifier.UseStateForUnknown(),
				},
				ElementType: types.StringType,
				Computed:    true,
			},
			"snapshot_type": schema.StringAttribute{
				Description: "Type of snapshot to create.",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.OneOf("instant", "object"),
				},
			},
			"policy_retention_in_days": schema.Int64Attribute{
				Description: "Number of days to retain snapshots.",
				Validators: []validator.Int64{
					int64validator.Between(1, 365),
				},
				Required: true,
			},
			"policy_frequency_daily_start_time": schema.StringAttribute{
				Description: "Start time for daily snapshot creation (HH:MM:SS format).",
				Validators: []validator.String{
					stringvalidator.RegexMatches(
						regexp.MustCompile(`^([01]?[0-9]|2[0-3]):[0-5][0-9]:[0-5][0-9]$`),
						"Must be in HH:MM:SS format (24-hour)",
					),
				},
				Required: true,
			},
			"state": schema.StringAttribute{
				Description: "The current state of the schedule.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
				Computed: true,
			},
			"created_at": schema.StringAttribute{
				Description: "The timestamp when the schedule was created.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
				Computed: true,
			},
			"updated_at": schema.StringAttribute{
				Description: "The timestamp when the schedule was last updated.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
				Computed: true,
			},
		},
	}
}

func (r *bsSchedule) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	data := &bsScheduleResourceModel{}
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	get, err := r.bsScheduler.Get(ctx, data.ID.ValueString(), []storageSDK.ExpandSchedulers{storageSDK.ExpandSchedulersVolume})
	if err != nil {
		resp.Diagnostics.AddError(utils.ParseSDKError(err))
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, SchedulerResponseToModel(get))...)
}

func (r *bsSchedule) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	plan := &bsScheduleResourceModel{}
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	created, err := r.bsScheduler.Create(ctx, storageSDK.SchedulerPayload{
		Name:        plan.Name.ValueString(),
		Description: plan.Description.ValueStringPointer(),
		Snapshot: storageSDK.SnapshotConfig{
			Type: plan.SnapshotType.ValueString(),
		},
		Policy: storageSDK.Policy{
			RetentionInDays: *utils.ConvertInt64PointerToIntPointer(plan.PolicyRetentionInDays.ValueInt64Pointer()),
			Frequency: storageSDK.Frequency{
				Daily: storageSDK.DailyFrequency{
					StartTime: plan.PolicyFrequencyDaily.ValueString(),
				},
			},
		},
	})
	if err != nil {
		resp.Diagnostics.AddError(utils.ParseSDKError(err))
		return
	}

	get, err := r.bsScheduler.Get(ctx, created, []storageSDK.ExpandSchedulers{storageSDK.ExpandSchedulersVolume})
	if err != nil {
		resp.Diagnostics.AddError(utils.ParseSDKError(err))
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, SchedulerResponseToModel(get))...)
}

func SchedulerResponseToModel(get *storageSDK.SchedulerResponse) *bsScheduleResourceModel {
	data := bsScheduleResourceModel{}
	data.CreatedAt = types.StringValue(get.CreatedAt.Format(time.RFC3339))
	data.UpdatedAt = types.StringValue(get.UpdatedAt.Format(time.RFC3339))
	data.ID = types.StringValue(get.ID)
	data.State = types.StringValue(string(get.State))
	data.PolicyFrequencyDaily = types.StringValue(get.Policy.Frequency.Daily.StartTime)
	data.PolicyRetentionInDays = types.Int64Value(int64(get.Policy.RetentionInDays))
	data.SnapshotType = types.StringValue(get.Snapshot.Type)
	data.Volumes = utils.StringSliceToTypesList(get.Volumes)
	data.Description = types.StringPointerValue(get.Description)
	data.Name = types.StringValue(get.Name)
	return &data
}

func (r *bsSchedule) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError("This resource does not support updates", "To modify a schedule, you must delete and recreate it with the desired changes.")
}

func (r *bsSchedule) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data bsScheduleResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.bsScheduler.Delete(ctx, data.ID.ValueString()); err != nil {
		resp.Diagnostics.AddError(utils.ParseSDKError(err))
		return
	}
}

func (r *bsSchedule) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
