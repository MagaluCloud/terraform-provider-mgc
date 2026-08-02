package database

import (
	"context"
	"fmt"
	"strings"
	"time"

	dbSDK "github.com/MagaluCloud/mgc-sdk-go/dbaas"

	"github.com/MagaluCloud/terraform-provider-mgc/mgc/utils"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type DBaaSClusterSnapshotStatus string

const (
	DBaaSClusterSnapshotStatusPending   DBaaSClusterSnapshotStatus = "PENDING"
	DBaaSClusterSnapshotStatusCreating  DBaaSClusterSnapshotStatus = "CREATING"
	DBaaSClusterSnapshotStatusAvailable DBaaSClusterSnapshotStatus = "AVAILABLE"
	DBaaSClusterSnapshotStatusRestoring DBaaSClusterSnapshotStatus = "RESTORING"
	DBaaSClusterSnapshotStatusError     DBaaSClusterSnapshotStatus = "ERROR"
	DBaaSClusterSnapshotStatusDeleting  DBaaSClusterSnapshotStatus = "DELETING"
	DBaaSClusterSnapshotStatusDeleted   DBaaSClusterSnapshotStatus = "DELETED"
)

func (s DBaaSClusterSnapshotStatus) String() string {
	return string(s)
}

var snapshotStatusPollInterval = 10 * time.Second

type DBaaSClusterSnapshotModel struct {
	ID          types.String `tfsdk:"id"`
	ClusterID   types.String `tfsdk:"cluster_id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
}

type DBaaSClusterSnapshotResource struct {
	dbaasClusters dbSDK.ClusterService
}

func NewDBaaSClusterSnapshotResource() resource.Resource {
	return &DBaaSClusterSnapshotResource{}
}

func (r *DBaaSClusterSnapshotResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dbaas_clusters_snapshots"
}

func (r *DBaaSClusterSnapshotResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	dataConfig, ok := req.ProviderData.(utils.DataConfig)
	if !ok {
		resp.Diagnostics.AddError("Failed to get provider data", "Failed to get provider data")
		return
	}

	r.dbaasClusters = dbSDK.New(dataConfig.CoreFor(utils.ServiceDatabase)).Clusters()
}

func (r *DBaaSClusterSnapshotResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a DBaaS cluster snapshot",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Unique identifier for the snapshot",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"cluster_id": schema.StringAttribute{
				Description: "ID of the DBaaS cluster to snapshot",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Description: "Name of the snapshot",
				Required:    true,
			},
			"description": schema.StringAttribute{
				Description: "Description of the snapshot",
				Required:    true,
			},
		},
	}
}

func (r *DBaaSClusterSnapshotResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data DBaaSClusterSnapshotModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := dbSDK.ClusterSnapshotCreateRequest{
		Name:        data.Name.ValueString(),
		Description: data.Description.ValueStringPointer(),
	}

	created, err := r.dbaasClusters.CreateSnapshot(ctx, data.ClusterID.ValueString(), body)
	if err != nil {
		resp.Diagnostics.AddError(utils.ParseSDKError(err))
		return
	}

	data.ID = types.StringValue(created.ID)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	err = r.waitUntilSnapshotStatusMatches(ctx, data.ClusterID.ValueString(), created.ID, DBaaSClusterSnapshotStatusAvailable)
	if err != nil {
		resp.Diagnostics.AddError(utils.ParseSDKError(err))
		return
	}
}

func (r *DBaaSClusterSnapshotResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data DBaaSClusterSnapshotModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	snapshot, err := r.dbaasClusters.GetSnapshot(ctx, data.ClusterID.ValueString(), data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(utils.ParseSDKError(err))
		return
	}

	data.Name = types.StringValue(snapshot.Name)
	data.Description = types.StringValue(snapshot.Description)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *DBaaSClusterSnapshotResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var planData DBaaSClusterSnapshotModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &planData)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var currentData DBaaSClusterSnapshotModel
	resp.Diagnostics.Append(req.State.Get(ctx, &currentData)...)
	if resp.Diagnostics.HasError() {
		return
	}

	currentData.Name = planData.Name
	currentData.Description = planData.Description

	_, err := r.dbaasClusters.UpdateSnapshot(ctx,
		currentData.ClusterID.ValueString(),
		currentData.ID.ValueString(),
		dbSDK.ClusterSnapshotUpdateRequest{
			Name:        planData.Name.ValueString(),
			Description: planData.Description.ValueStringPointer(),
		},
	)
	if err != nil {
		resp.Diagnostics.AddError(utils.ParseSDKError(err))
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &currentData)...)
}

func (r *DBaaSClusterSnapshotResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data DBaaSClusterSnapshotModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.dbaasClusters.DeleteSnapshot(ctx, data.ClusterID.ValueString(), data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(utils.ParseSDKError(err))
		return
	}
}

func (r *DBaaSClusterSnapshotResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	ids := strings.Split(req.ID, ",")
	if len(ids) != 2 {
		resp.Diagnostics.AddError(
			"Invalid import format",
			"Format should be: cluster_id,snapshot_id",
		)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &DBaaSClusterSnapshotModel{
		ClusterID: types.StringValue(ids[0]),
		ID:        types.StringValue(ids[1])})...)
}

func (r *DBaaSClusterSnapshotResource) waitUntilSnapshotStatusMatches(ctx context.Context, clusterID string, snapshotID string, status DBaaSClusterSnapshotStatus) error {
	timeoutCtx, cancel := context.WithTimeout(ctx, clusterStatusTimeout)
	defer cancel()

	for {
		select {
		case <-timeoutCtx.Done():
			return fmt.Errorf("timeout waiting for snapshot %s to reach status %s", snapshotID, status)
		case <-time.After(snapshotStatusPollInterval):
			snapshot, err := r.dbaasClusters.GetSnapshot(ctx, clusterID, snapshotID)
			if err != nil {
				return err
			}

			currentStatus := DBaaSClusterSnapshotStatus(snapshot.Status)
			if currentStatus == status {
				return nil
			}
			if currentStatus == DBaaSClusterSnapshotStatusError {
				return fmt.Errorf("snapshot %s is in error state", snapshotID)
			}
		}
	}
}
