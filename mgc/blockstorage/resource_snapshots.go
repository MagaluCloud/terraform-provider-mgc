package blockstorage

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"

	storageSDK "github.com/MagaluCloud/mgc-sdk-go/blockstorage"

	"github.com/MagaluCloud/terraform-provider-mgc/mgc/utils"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

const volumeSnapshotStatusTimeout = 70 * time.Minute

type SnapshotStatus string

const (
	SnapshotProvisioning       SnapshotStatus = "provisioning"
	SnapshotCreating           SnapshotStatus = "creating"
	SnapshotCreatingError      SnapshotStatus = "creating_error"
	SnapshotCreatingErrorQuota SnapshotStatus = "creating_error_quota"
	SnapshotCompleted          SnapshotStatus = "completed"
	SnapshotDeleting           SnapshotStatus = "deleting"
	SnapshotDeleted            SnapshotStatus = "deleted"
	SnapshotDeletedError       SnapshotStatus = "deleted_error"
	SnapshotReplicating        SnapshotStatus = "replicating"
	SnapshotReplicatingError   SnapshotStatus = "replicating_error"
	SnapshotRestoring          SnapshotStatus = "restoring"
	SnapshotRestoringError     SnapshotStatus = "restoring_error"
	SnapshotReserved           SnapshotStatus = "reserved"
)

func (s SnapshotStatus) String() string {
	return string(s)
}

func (s SnapshotStatus) IsError() bool {
	return strings.HasSuffix(s.String(), "error")
}

func NewBlockStorageSnapshotsResource() resource.Resource {
	return &bsSnapshots{}
}

type bsSnapshots struct {
	bsSnapshots storageSDK.SnapshotService
}

func (r *bsSnapshots) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_block_storage_snapshots"
}

func (r *bsSnapshots) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	dataConfig, ok := req.ProviderData.(utils.DataConfig)
	if !ok {
		resp.Diagnostics.AddError("Failed to get provider data", "Failed to get provider data")
		return
	}

	r.bsSnapshots = storageSDK.New(&dataConfig.CoreConfig).Snapshots()
}

type bsSnapshotsResourceModel struct {
	ID               types.String `tfsdk:"id"`
	Name             types.String `tfsdk:"name"`
	Description      types.String `tfsdk:"description"`
	CreatedAt        types.String `tfsdk:"created_at"`
	VolumeId         types.String `tfsdk:"volume_id"`
	SnapshotSourceID types.String `tfsdk:"snapshot_source_id"`
	Type             types.String `tfsdk:"type"`
}

func (r *bsSnapshots) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "The block storage snapshots resource allows you to manage block storage snapshots in the Magalu Cloud.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The unique identifier of the volume snapshot.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
				Computed: true,
			},
			"name": schema.StringAttribute{
				Description: "The name of the volume snapshot.",
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
				Description: "The description of the volume snapshot.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 255),
				},
				Required: true,
			},
			"created_at": schema.StringAttribute{
				Description: "The timestamp when the block storage was created.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					stringplanmodifier.RequiresReplace(),
				},
				Computed: true,
			},
			"volume_id": schema.StringAttribute{
				Description: "ID of block storage volume. Is required when snapshot source is not set and both volume ID and snapshot source ID cannot be set at the same time.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Optional: true,
			},
			"snapshot_source_id": schema.StringAttribute{
				Description: "The ID of the snapshot source, for creating a snapshot object from a snapshot instant. Is required when volume ID is not set and both volume ID and snapshot source ID cannot be set at the same time.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Optional: true,
			},
			"type": schema.StringAttribute{
				Description: "The type of the snapshot.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.OneOf("instant", "object"),
				},
				Optional: true,
				Computed: true,
				Default:  stringdefault.StaticString("instant"),
			},
		},
	}
}

func (r *bsSnapshots) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	data := &bsSnapshotsResourceModel{}
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	result, err := r.bsSnapshots.Get(ctx, data.ID.ValueString(), []string{})
	if err != nil {
		resp.Diagnostics.AddError(utils.ParseSDKError(err))
		return
	}

	convertedResult := r.toTerraformModel(*result, data.SnapshotSourceID.ValueStringPointer())
	resp.Diagnostics.Append(resp.State.Set(ctx, &convertedResult)...)
}

func (r *bsSnapshots) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	plan := &bsSnapshotsResourceModel{}
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	createRequest := storageSDK.CreateSnapshotRequest{
		Name:        plan.Name.ValueString(),
		Description: plan.Description.ValueStringPointer(),
		Volume: &storageSDK.IDOrName{
			ID: plan.VolumeId.ValueStringPointer(),
		},
		Type: plan.Type.ValueStringPointer(),
	}

	if !plan.SnapshotSourceID.IsNull() {
		createRequest.SourceSnapshot = &storageSDK.IDOrName{
			ID: plan.SnapshotSourceID.ValueStringPointer(),
		}
	}

	createID, err := r.bsSnapshots.Create(ctx, createRequest)
	if err != nil {
		resp.Diagnostics.AddError(utils.ParseSDKError(err))
		return
	}
	plan.ID = types.StringValue(createID)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)

	getResult, err := r.waitUntilSnapshotStatusMatches(ctx, createID, SnapshotCompleted)
	if err != nil {
		resp.Diagnostics.AddError(utils.ParseSDKError(err))
		return
	}

	convertedGet := r.toTerraformModel(*getResult, plan.SnapshotSourceID.ValueStringPointer())
	resp.Diagnostics.Append(resp.State.Set(ctx, &convertedGet)...)
}

func (r *bsSnapshots) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	state := &bsSnapshotsResourceModel{}
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	plan := &bsSnapshotsResourceModel{}
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if state.Name != plan.Name {
		err := r.bsSnapshots.Rename(ctx, state.ID.ValueString(), plan.Name.ValueString())
		if err != nil {
			resp.Diagnostics.AddError(utils.ParseSDKError(err))
			return
		}
		_, err = r.waitUntilSnapshotStatusMatches(ctx, state.ID.ValueString(), SnapshotCompleted)
		if err != nil {
			resp.Diagnostics.AddError(utils.ParseSDKError(err))
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *bsSnapshots) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data bsSnapshotsResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	err := r.bsSnapshots.Delete(ctx, data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(utils.ParseSDKError(err))
	}
}

func (r *bsSnapshots) toTerraformModel(snapshot storageSDK.Snapshot, sourceSnapshotId *string) bsSnapshotsResourceModel {
	return bsSnapshotsResourceModel{
		ID:               types.StringValue(snapshot.ID),
		Name:             types.StringValue(snapshot.Name),
		Description:      types.StringPointerValue(snapshot.Description),
		CreatedAt:        types.StringValue(*utils.ConvertTimeToRFC3339(&snapshot.CreatedAt)),
		VolumeId:         types.StringPointerValue(snapshot.Volume.ID),
		SnapshotSourceID: types.StringPointerValue(sourceSnapshotId),
		Type:             types.StringValue(snapshot.Type),
	}
}

func (r *bsSnapshots) waitUntilSnapshotStatusMatches(ctx context.Context, snapshotID string, status SnapshotStatus) (*storageSDK.Snapshot, error) {
	timeoutCtx, cancel := context.WithTimeout(ctx, volumeSnapshotStatusTimeout)
	defer cancel()

	for {
		select {
		case <-timeoutCtx.Done():
			return nil, fmt.Errorf("timeout waiting for snapshot %s to reach status %s", snapshotID, status)
		case <-time.After(10 * time.Second):
			snapshot, err := r.bsSnapshots.Get(ctx, snapshotID, []string{})
			if err != nil {
				return nil, err
			}
			currentStatus := SnapshotStatus(snapshot.Status)
			if currentStatus == status {
				return snapshot, nil
			}
			if currentStatus.IsError() {
				return nil, fmt.Errorf("snapshot %s is in error state: %s", snapshotID, currentStatus)
			}
		}
	}
}
