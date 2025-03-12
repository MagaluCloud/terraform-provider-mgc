package resources

import (
	"context"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"

	"github.com/hashicorp/terraform-plugin-framework/types"

	storageSDK "github.com/MagaluCloud/mgc-sdk-go/blockstorage"
	"github.com/MagaluCloud/terraform-provider-mgc/mgc/tfutil"
)

const (
	AttachVolumeTimeout         = 5 * time.Minute
	AttachVolumeCompletedStatus = "completed"
)

type VolumeAttach struct {
	blockStorageVolumes storageSDK.VolumeService
}

type VolumeAttachResourceModel struct {
	BlockStorageID   types.String `tfsdk:"block_storage_id"`
	VirtualMachineID types.String `tfsdk:"virtual_machine_id"`
}

func NewVolumeAttachResource() resource.Resource {
	return &VolumeAttach{}
}

func (r *VolumeAttach) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_block_storage_volume_attachment"
}

func (r *VolumeAttach) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	dataConfig, ok := req.ProviderData.(tfutil.DataConfig)
	if !ok {
		resp.Diagnostics.AddError("Failed to get provider data", "Failed to get provider data")
		return
	}

	r.blockStorageVolumes = storageSDK.New(&dataConfig.CoreConfig).Volumes()
}

func (r *VolumeAttach) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "A block storage volume attachment.",
		Attributes: map[string]schema.Attribute{
			"block_storage_id": schema.StringAttribute{
				Description: "The ID of the block storage volume to attach.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"virtual_machine_id": schema.StringAttribute{
				Description: "The ID of the virtual machine to attach the volume to.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
		},
	}
}

func (r *VolumeAttach) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var model VolumeAttachResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.blockStorageVolumes.Attach(ctx, model.BlockStorageID.ValueString(), model.VirtualMachineID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(tfutil.ParseSDKError(err))
		return
	}

	err = r.waitForVolumeAvailability(ctx, model.BlockStorageID.ValueString(), AttachVolumeCompletedStatus)
	if err != nil {
		resp.Diagnostics.AddError(tfutil.ParseSDKError(err))
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}

func (r *VolumeAttach) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var model VolumeAttachResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, err := r.blockStorageVolumes.Get(ctx, model.BlockStorageID.ValueString(), []string{storageSDK.VolumeAttachExpand})
	if err != nil {
		resp.Diagnostics.AddError(tfutil.ParseSDKError(err))
		return
	}

	if result.Attachment == nil {
		resp.Diagnostics.AddWarning("Volume is not attached to any virtual machine", "")
		model.VirtualMachineID = types.StringValue("")
	} else {
		model.VirtualMachineID = types.StringPointerValue(result.Attachment.Instance.ID)
	}
	model.BlockStorageID = types.StringValue(result.ID)

	resp.State.Set(ctx, &model)
}

func (r *VolumeAttach) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError("Update not suported", "Update not suported for this resource")
}

func (r *VolumeAttach) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var model VolumeAttachResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.blockStorageVolumes.Detach(ctx, model.BlockStorageID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(tfutil.ParseSDKError(err))
		return
	}

	err = r.waitForVolumeAvailability(ctx, model.BlockStorageID.ValueString(), AttachVolumeCompletedStatus)
	if err != nil {
		resp.Diagnostics.AddError(tfutil.ParseSDKError(err))
		return
	}
}

func (r *VolumeAttach) waitForVolumeAvailability(ctx context.Context, volumeID string, expetedStatus string) (err error) {
	for startTime := time.Now(); time.Since(startTime) < AttachVolumeTimeout; {
		time.Sleep(10 * time.Second)
		getResult, err := r.blockStorageVolumes.Get(ctx, volumeID, []string{})
		if err != nil {
			return err
		}
		if getResult.Status == expetedStatus {
			break
		}
	}
	return nil
}
