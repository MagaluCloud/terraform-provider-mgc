package blockstorage

import (
	"context"
	"fmt"
	"strings"
	"time"

	clientSDK "github.com/MagaluCloud/mgc-sdk-go/client"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"

	storageSDK "github.com/MagaluCloud/mgc-sdk-go/blockstorage"

	"github.com/MagaluCloud/terraform-provider-mgc/mgc/internal/project"
	"github.com/MagaluCloud/terraform-provider-mgc/mgc/utils"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

const (
	completedBsStatus     = "completed"
	BsVolumeStatusTimeout = 60 * time.Minute
)

type VolumeStatus string

const (
	Provisioning       VolumeStatus = "provisioning"
	Creating           VolumeStatus = "creating"
	CreatingError      VolumeStatus = "creating_error"
	CreatingErrorQuota VolumeStatus = "creating_error_quota"
	Completed          VolumeStatus = "completed"
	ExtendPending      VolumeStatus = "extend_pending"
	Extending          VolumeStatus = "extending"
	ExtendError        VolumeStatus = "extend_error"
	ExtendErrorQuota   VolumeStatus = "extend_error_quota"
	AttachingPending   VolumeStatus = "attaching_pending"
	AttachingError     VolumeStatus = "attaching_error"
	Attaching          VolumeStatus = "attaching"
	DetachingPending   VolumeStatus = "detaching_pending"
	DetachingError     VolumeStatus = "detaching_error"
	Detaching          VolumeStatus = "detaching"
	RetypePending      VolumeStatus = "retype_pending"
	Retyping           VolumeStatus = "retyping"
	RetypeError        VolumeStatus = "retype_error"
	RetypeErrorQuota   VolumeStatus = "retype_error_quota"
	DeletingPending    VolumeStatus = "deleting_pending"
	Deleting           VolumeStatus = "deleting"
	Deleted            VolumeStatus = "deleted"
	DeletedError       VolumeStatus = "deleted_error"
)

func (s VolumeStatus) String() string {
	return string(s)
}

func (s VolumeStatus) isError() bool {
	return strings.Contains(s.String(), "error")
}

func NewBlockStorageVolumesResource() resource.Resource {
	return &bsVolumes{}
}

type bsVolumes struct {
	bsVolumes storageSDK.VolumeService
}

func (r *bsVolumes) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_block_storage_volumes"
}

func (r *bsVolumes) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	dataConfig, ok := req.ProviderData.(utils.DataConfig)
	if !ok {
		resp.Diagnostics.AddError("Failed to get provider data", "Failed to get provider data")
		return
	}

	r.bsVolumes = storageSDK.New(dataConfig.CoreFor(utils.ServiceBlockStorage)).Volumes()
}

type bsVolumesResourceModel struct {
	ID               types.String `tfsdk:"id"`
	Name             types.String `tfsdk:"name"`
	SnapshotID       types.String `tfsdk:"snapshot_id"`
	AvailabilityZone types.String `tfsdk:"availability_zone"`
	CreatedAt        types.String `tfsdk:"created_at"`
	Size             types.Int64  `tfsdk:"size"`
	Type             types.String `tfsdk:"type"`
	Encrypted        types.Bool   `tfsdk:"encrypted"`
	ProjectID        types.String `tfsdk:"project_id"`
}

func (r *bsVolumes) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Block storage volumes are storage devices that can be attached to virtual machines. They are used to store data and can be detached and attached to other virtual machines.",
		Attributes: map[string]schema.Attribute{
			"project_id": project.ResourceAttribute(),
			"id": schema.StringAttribute{
				Description: "The unique identifier of the volume.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					stringplanmodifier.RequiresReplace(),
				},
				Computed: true,
			},
			"name": schema.StringAttribute{
				Description: "The name of the volume.",
				Required:    true,
			},
			"snapshot_id": schema.StringAttribute{
				Description: "Create a volume from a snapshot.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
				Optional: true,
			},
			"availability_zone": schema.StringAttribute{
				Description: "The availability zones where the volume is available.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"size": schema.Int64Attribute{
				Description: "The size of the volume in GB.",
				Required:    true,
			},
			"created_at": schema.StringAttribute{
				Description: "The timestamp when the volume was created.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
				Computed: true,
			},
			"type": schema.StringAttribute{
				Required:    true,
				Description: "The name of the volume type.",
			},
			"encrypted": schema.BoolAttribute{
				Description: "Indicates if the volume is encrypted.",
				Computed:    true,
				Optional:    true,
				PlanModifiers: []planmodifier.Bool{
					utils.ReplaceIfChangeAndNotIsNotSetOnPlanBool{},
					boolplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}

}

func (r *bsVolumes) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	plan := &bsVolumesResourceModel{}
	resp.Diagnostics.Append(req.State.Get(ctx, &plan)...)

	getResult, err := r.bsVolumes.Get(ctx, project.CallOptions(plan.ProjectID), plan.ID.ValueString(), []string{storageSDK.VolumeTypeExpand})
	if err != nil {
		resp.Diagnostics.AddError(project.ParseSDKError(err, plan.ProjectID))
		return
	}

	convertedResult := r.toTerraformModel(*getResult, plan.SnapshotID.ValueStringPointer(), plan.ProjectID)
	resp.Diagnostics.Append(resp.State.Set(ctx, &convertedResult)...)
	resp.Diagnostics.Append(project.SetIdentity(ctx, resp.Identity, convertedResult.ID, convertedResult.ProjectID)...)
}

func (r *bsVolumes) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	state := &bsVolumesResourceModel{}
	resp.Diagnostics.Append(req.Plan.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	createParam := storageSDK.CreateVolumeRequest{
		Name:      state.Name.ValueString(),
		Size:      int(state.Size.ValueInt64()),
		Encrypted: state.Encrypted.ValueBoolPointer(),
		Type: storageSDK.IDOrName{
			Name: state.Type.ValueStringPointer(),
		},
	}
	if !state.SnapshotID.IsNull() {
		createParam.Snapshot = &storageSDK.IDOrName{
			ID: state.SnapshotID.ValueStringPointer(),
		}
	}
	if !state.AvailabilityZone.IsUnknown() {
		createParam.AvailabilityZone = state.AvailabilityZone.ValueStringPointer()
	}

	callOpts := project.CallOptions(state.ProjectID)
	createResult, err := r.bsVolumes.Create(ctx, callOpts, createParam)
	if err != nil {
		resp.Diagnostics.AddError(utils.ParseSDKError(err))
		return
	}
	getResult, err := r.waitUntilVolumeStatusMatches(ctx, callOpts, createResult, Completed)
	if err != nil {
		resp.Diagnostics.AddError(utils.ParseSDKError(err))
		return
	}
	convertedResult := r.toTerraformModel(*getResult, state.SnapshotID.ValueStringPointer(), state.ProjectID)
	resp.Diagnostics.Append(resp.State.Set(ctx, &convertedResult)...)
	resp.Diagnostics.Append(project.SetIdentity(ctx, resp.Identity, convertedResult.ID, convertedResult.ProjectID)...)
}

func (r *bsVolumes) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	planData := &bsVolumesResourceModel{}
	state := &bsVolumesResourceModel{}

	resp.Diagnostics.Append(req.Plan.Get(ctx, &planData)...)
	resp.Diagnostics.Append(req.State.Get(ctx, state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	callOpts := project.CallOptions(state.ProjectID)

	if planData.Name.ValueString() != state.Name.ValueString() {
		err := r.bsVolumes.Rename(ctx, callOpts, planData.ID.ValueString(), planData.Name.ValueString())
		if err != nil {
			resp.Diagnostics.AddError(utils.ParseSDKError(err))
			return
		}
		_, err = r.waitUntilVolumeStatusMatches(ctx, callOpts, state.ID.ValueString(), Completed)
		if err != nil {
			resp.Diagnostics.AddError(utils.ParseSDKError(err))
			return
		}
	}

	if planData.Type.ValueString() != state.Type.ValueString() {
		err := r.bsVolumes.Retype(ctx, callOpts, planData.ID.ValueString(), storageSDK.RetypeVolumeRequest{
			NewType: storageSDK.IDOrName{
				Name: planData.Type.ValueStringPointer(),
			},
		})
		if err != nil {
			resp.Diagnostics.AddError(utils.ParseSDKError(err))
			return
		}
		_, err = r.waitUntilVolumeStatusMatches(ctx, callOpts, state.ID.ValueString(), Completed)
		if err != nil {
			resp.Diagnostics.AddError(utils.ParseSDKError(err))
			return
		}
	}

	if planData.Size.ValueInt64() != state.Size.ValueInt64() {
		err := r.bsVolumes.Extend(ctx, callOpts, planData.ID.ValueString(), storageSDK.ExtendVolumeRequest{
			Size: int(planData.Size.ValueInt64()),
		})
		if err != nil {
			resp.Diagnostics.AddError(utils.ParseSDKError(err))
			return
		}
		_, err = r.waitUntilVolumeStatusMatches(ctx, callOpts, state.ID.ValueString(), Completed)
		if err != nil {
			resp.Diagnostics.AddError(utils.ParseSDKError(err))
			return
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &planData)...)
}

func (r *bsVolumes) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data bsVolumesResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.bsVolumes.Delete(ctx, project.CallOptions(data.ProjectID), data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(utils.ParseSDKError(err))
	}
}

func (r *bsVolumes) IdentitySchema(_ context.Context, _ resource.IdentitySchemaRequest, resp *resource.IdentitySchemaResponse) {
	resp.IdentitySchema = project.IdentitySchema()
}

func (r *bsVolumes) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	target, diags := project.ResolveImport(ctx, req)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	data := &bsVolumesResourceModel{ID: target.ID, ProjectID: target.ProjectID}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	resp.Diagnostics.Append(project.SetIdentity(ctx, resp.Identity, target.ID, target.ProjectID)...)
}

func (r *bsVolumes) toTerraformModel(volume storageSDK.Volume, snapshotId *string, projectID types.String) bsVolumesResourceModel {
	return bsVolumesResourceModel{
		ID:               types.StringValue(volume.ID),
		Name:             types.StringValue(volume.Name),
		AvailabilityZone: types.StringValue(volume.AvailabilityZone),
		CreatedAt:        types.StringValue(*utils.ConvertTimeToRFC3339(&volume.CreatedAt)),
		Size:             types.Int64PointerValue(utils.ConvertIntPointerToInt64Pointer(&volume.Size)),
		Type:             types.StringPointerValue(volume.Type.Name),
		SnapshotID:       types.StringPointerValue(snapshotId),
		Encrypted:        types.BoolPointerValue(volume.Encrypted),
		ProjectID:        projectID,
	}
}

func (r *bsVolumes) waitUntilVolumeStatusMatches(ctx context.Context, callOpts clientSDK.CallOptions, volumeID string, status VolumeStatus) (*storageSDK.Volume, error) {
	timeoutCtx, cancel := context.WithTimeout(ctx, BsVolumeStatusTimeout)
	defer cancel()

	for {
		select {
		case <-timeoutCtx.Done():
			return nil, fmt.Errorf("timeout waiting for volume %s to reach status %s", volumeID, status)
		case <-time.After(10 * time.Second):
			volume, err := r.bsVolumes.Get(ctx, callOpts, volumeID, []string{storageSDK.VolumeTypeExpand})
			if err != nil {
				return nil, err
			}
			currentStatus := VolumeStatus(volume.Status)
			if currentStatus == status {
				return volume, nil
			}
			if currentStatus.isError() {
				return nil, fmt.Errorf("volume %s is in error state: %s", volumeID, currentStatus)
			}
		}
	}
}
