package resources

import (
	"context"
	"fmt"
	"strings"
	"time"

	dbSDK "github.com/MagaluCloud/mgc-sdk-go/dbaas/v2"

	"github.com/MagaluCloud/terraform-provider-mgc/mgc/tfutil"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

const snapshotStatusTimeout = 70 * time.Minute

type DBaaSInstanceSnapshotStatus string

const (
	DBaaSInstanceSnapshotStatusPending   DBaaSInstanceSnapshotStatus = "PENDING"
	DBaaSInstanceSnapshotStatusCreating  DBaaSInstanceSnapshotStatus = "CREATING"
	DBaaSInstanceSnapshotStatusAvailable DBaaSInstanceSnapshotStatus = "AVAILABLE"
	DBaaSInstanceSnapshotStatusRestoring DBaaSInstanceSnapshotStatus = "RESTORING"
	DBaaSInstanceSnapshotStatusError     DBaaSInstanceSnapshotStatus = "ERROR"
	DBaaSInstanceSnapshotStatusDeleting  DBaaSInstanceSnapshotStatus = "DELETING"
	DBaaSInstanceSnapshotStatusDeleted   DBaaSInstanceSnapshotStatus = "DELETED"
)

func (s DBaaSInstanceSnapshotStatus) String() string {
	return string(s)
}

type DBaaSInstanceSnapshotModel struct {
	Id          types.String `tfsdk:"id"`
	InstanceId  types.String `tfsdk:"instance_id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
}

type DBaaSInstanceSnapshotResource struct {
	instanceService dbSDK.InstanceService
}

func NewDBaaSInstanceSnapshotResource() resource.Resource {
	return &DBaaSInstanceSnapshotResource{}
}

func (r *DBaaSInstanceSnapshotResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dbaas_instances_snapshots"
}

func (r *DBaaSInstanceSnapshotResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	dataConfig, ok := req.ProviderData.(tfutil.DataConfig)
	if !ok {
		resp.Diagnostics.AddError("Failed to get provider data", "Failed to get provider data")
		return
	}

	r.instanceService = dbSDK.New(&dataConfig.CoreConfig).Instances()
}

func (r *DBaaSInstanceSnapshotResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a DBaaS instance snapshot",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Unique identifier for the snapshot",
				Computed:    true,
			},
			"instance_id": schema.StringAttribute{
				Description: "ID of the DBaaS instance to snapshot",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Description: "Name of the snapshot",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"description": schema.StringAttribute{
				Description: "Description of the snapshot",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
		},
	}
}

func (r *DBaaSInstanceSnapshotResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data DBaaSInstanceSnapshotModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	created, err := r.instanceService.CreateSnapshot(ctx, data.InstanceId.ValueString(), dbSDK.SnapshotCreateRequest{
		Name:        data.Name.ValueString(),
		Description: data.Description.ValueStringPointer(),
	})
	if err != nil {
		resp.Diagnostics.AddError(tfutil.ParseSDKError(err))
		return
	}

	data.Id = types.StringValue(created.ID)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	err = r.waitUntilSnapshotStatusMatches(ctx, data.InstanceId.ValueString(), created.ID, DBaaSInstanceSnapshotStatusAvailable)
	if err != nil {
		resp.Diagnostics.AddError(tfutil.ParseSDKError(err))
		return
	}
}

func (r *DBaaSInstanceSnapshotResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data DBaaSInstanceSnapshotModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	snapshot, err := r.instanceService.GetSnapshot(ctx, data.InstanceId.ValueString(), data.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(tfutil.ParseSDKError(err))
		return
	}

	data.Name = types.StringValue(snapshot.Name)
	data.Description = types.StringValue(snapshot.Description)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *DBaaSInstanceSnapshotResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var planData DBaaSInstanceSnapshotModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &planData)...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.instanceService.UpdateSnapshot(ctx,
		planData.InstanceId.ValueString(),
		planData.Id.ValueString(),
		dbSDK.SnapshotUpdateRequest{
			Name:        planData.Name.ValueString(),
			Description: planData.Description.ValueStringPointer(),
		},
	)
	if err != nil {
		resp.Diagnostics.AddError(tfutil.ParseSDKError(err))
		return
	}
}

func (r *DBaaSInstanceSnapshotResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data DBaaSInstanceSnapshotModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.instanceService.DeleteSnapshot(ctx, data.InstanceId.ValueString(), data.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(tfutil.ParseSDKError(err))
		return
	}
}

func (r *DBaaSInstanceSnapshotResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	ids := strings.Split(req.ID, ",")
	if len(ids) != 2 {
		resp.Diagnostics.AddError(
			"Invalid import format",
			"Format should be: instance_id,snapshot_id",
		)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &DBaaSInstanceSnapshotModel{
		InstanceId: types.StringValue(ids[0]),
		Id:         types.StringValue(ids[1])})...)
}

func (r *DBaaSInstanceSnapshotResource) waitUntilSnapshotStatusMatches(ctx context.Context, instanceID string, snapshotID string, status DBaaSInstanceSnapshotStatus) error {
	timeoutCtx, cancel := context.WithTimeout(ctx, snapshotStatusTimeout)
	defer cancel()

	for {
		select {
		case <-timeoutCtx.Done():
			return fmt.Errorf("timeout waiting for snapshot %s to reach status %s", snapshotID, status)
		case <-time.After(10 * time.Second):
			snapshot, err := r.instanceService.GetSnapshot(ctx, instanceID, snapshotID)
			if err != nil {
				return err
			}

			currentStatus := DBaaSInstanceSnapshotStatus(snapshot.Status)
			if currentStatus == status {
				return nil
			}
			if currentStatus == DBaaSInstanceSnapshotStatusError {
				return fmt.Errorf("snapshot %s is in error state", snapshotID)
			}
		}
	}
}
