package blockstorage

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	storageSDK "github.com/MagaluCloud/mgc-sdk-go/blockstorage"
	"github.com/MagaluCloud/terraform-provider-mgc/mgc/utils"
)

type bsScheduleAttach struct {
	bsScheduler storageSDK.SchedulerService
}

type bsScheduleAttachResourceModel struct {
	ScheduleID types.String `tfsdk:"schedule_id"`
	VolumeID   types.String `tfsdk:"volume_id"`
}

func NewBlockStorageScheduleAttachResource() resource.Resource {
	return &bsScheduleAttach{}
}

func (r *bsScheduleAttach) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_block_storage_schedule_attach"
}

func (r *bsScheduleAttach) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *bsScheduleAttach) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Attaches a block storage volume to a snapshot schedule. This creates a relationship between a volume and a schedule, allowing the schedule to create snapshots of the volume.",
		Attributes: map[string]schema.Attribute{
			"schedule_id": schema.StringAttribute{
				Description: "The ID of the snapshot schedule to attach the volume to.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"volume_id": schema.StringAttribute{
				Description: "The ID of the block storage volume to attach to the schedule.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
		},
	}
}

func (r *bsScheduleAttach) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	data := &bsScheduleAttachResourceModel{}
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	get, err := r.bsScheduler.Get(ctx, data.ScheduleID.ValueString(), []storageSDK.ExpandSchedulers{storageSDK.ExpandSchedulersVolume})
	if err != nil {
		resp.Diagnostics.AddError(utils.ParseSDKError(err))
		return
	}

	if !slices.Contains(get.Volumes, data.VolumeID.ValueString()) {
		resp.Diagnostics.AddError(
			"Volume not attached to schedule",
			fmt.Sprintf("Volume %s is not attached to schedule %s", data.VolumeID.ValueString(), data.ScheduleID.ValueString()),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *bsScheduleAttach) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	plan := &bsScheduleAttachResourceModel{}
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.bsScheduler.AttachVolume(ctx, plan.ScheduleID.ValueString(), storageSDK.SchedulerVolumeIdentifierPayload{
		Volume: storageSDK.IDOrName{
			ID: plan.VolumeID.ValueStringPointer(),
		},
	})
	if err != nil {
		resp.Diagnostics.AddError(utils.ParseSDKError(err))
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *bsScheduleAttach) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError("Update not supported", "This resource does not support updates. To modify the attachment, you must delete and recreate it.")
}

func (r *bsScheduleAttach) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data bsScheduleAttachResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.bsScheduler.DetachVolume(ctx, data.ScheduleID.ValueString(), storageSDK.SchedulerVolumeIdentifierPayload{
		Volume: storageSDK.IDOrName{
			ID: data.VolumeID.ValueStringPointer(),
		},
	})
	if err != nil {
		resp.Diagnostics.AddError(utils.ParseSDKError(err))
		return
	}
}

func (r *bsScheduleAttach) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	input := strings.Split(req.ID, ",")
	if len(input) != 2 {
		resp.Diagnostics.AddError("Invalid ID", "ID must be in the format schedule_id,volume_id")
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &bsScheduleAttachResourceModel{
		ScheduleID: types.StringValue(input[0]),
		VolumeID:   types.StringValue(input[1]),
	})...)
}
