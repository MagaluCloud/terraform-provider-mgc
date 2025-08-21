package database

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	dbSDK "github.com/MagaluCloud/mgc-sdk-go/dbaas"
	"github.com/MagaluCloud/terraform-provider-mgc/mgc/utils"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

const (
	dbaasReplicaProductFamily = "SINGLE_INSTANCE_REPLICA"
)

type DBaaSReplicaModel struct {
	ID           types.String `tfsdk:"id"`
	SourceID     types.String `tfsdk:"source_id"`
	Name         types.String `tfsdk:"name"`
	EngineID     types.String `tfsdk:"engine_id"`
	InstanceType types.String `tfsdk:"instance_type"`
	Status       types.String `tfsdk:"status"`
}

type DBaaSReplicaResource struct {
	dbaasReplicas      dbSDK.ReplicaService
	dbaasInstances     dbSDK.InstanceService
	dbaasInstanceTypes dbSDK.InstanceTypeService
}

func NewDBaaSReplicaResource() resource.Resource {
	return &DBaaSReplicaResource{}
}

func (r *DBaaSReplicaResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dbaas_replicas"
}

func (r *DBaaSReplicaResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	cfg, ok := req.ProviderData.(utils.DataConfig)
	if !ok {
		resp.Diagnostics.AddError("invalid provider data", "expected utils.DataConfig")
		return
	}
	r.dbaasReplicas = dbSDK.New(&cfg.CoreConfig).Replicas()
	r.dbaasInstances = dbSDK.New(&cfg.CoreConfig).Instances()
	r.dbaasInstanceTypes = dbSDK.New(&cfg.CoreConfig).InstanceTypes()
}

func (r *DBaaSReplicaResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a DBaaS replica",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Replica ID",
			},
			"source_id": schema.StringAttribute{
				Required:    true,
				Description: "ID of the source instance",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "Replica name",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"instance_type": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Instance type",
			},
			"engine_id": schema.StringAttribute{
				Computed:    true,
				Description: "Engine ID",
			},
			"status": schema.StringAttribute{
				Computed:    true,
				Description: "Replica status",
			},
		},
	}
}

func (r *DBaaSReplicaResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data DBaaSReplicaModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var ptrTypeID *string
	if !data.InstanceType.IsNull() && data.InstanceType.ValueString() != "" {
		sourceData, err := r.dbaasInstances.Get(ctx, data.SourceID.ValueString(), dbSDK.GetInstanceOptions{})
		if err != nil {
			resp.Diagnostics.AddError(utils.ParseSDKError(err))
			return
		}

		instanceTypeID, err := ValidateAndGetInstanceTypeID(ctx, r.dbaasInstanceTypes.List, data.InstanceType.ValueString(),
			sourceData.EngineID, dbaasReplicaProductFamily)
		if err != nil {
			resp.Diagnostics.AddError(utils.ParseSDKError(err))
			return
		}

		v := instanceTypeID
		ptrTypeID = &v
	}
	created, err := r.dbaasReplicas.Create(ctx, dbSDK.ReplicaCreateRequest{
		SourceID:       data.SourceID.ValueString(),
		Name:           data.Name.ValueString(),
		InstanceTypeID: ptrTypeID,
	})
	if err != nil {
		resp.Diagnostics.AddError(utils.ParseSDKError(err))
		return
	}

	data.ID = types.StringValue(created.ID)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)

	found, err := r.waitUntilReplicaStatusMatches(ctx, created.ID, string(dbSDK.InstanceStatusActive))
	if err != nil {
		resp.Diagnostics.AddError("Error waiting for replica to be active", err.Error())
		return
	}

	data.SourceID = types.StringValue(found.SourceID)
	data.Name = types.StringValue(found.Name)
	data.EngineID = types.StringValue(found.EngineID)

	instanceTypeName, err := GetInstanceTypeNameByID(ctx, r.dbaasInstanceTypes.Get, found.InstanceTypeID)
	if err != nil {
		resp.Diagnostics.AddError(utils.ParseSDKError(err))
		return
	}

	data.InstanceType = types.StringValue(instanceTypeName)
	data.Status = types.StringValue(string(found.Status))
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *DBaaSReplicaResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data DBaaSReplicaModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	detail, err := r.dbaasReplicas.Get(ctx, data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(utils.ParseSDKError(err))
		return
	}

	data.SourceID = types.StringValue(detail.SourceID)
	data.Name = types.StringValue(detail.Name)
	data.EngineID = types.StringValue(detail.EngineID)

	instanceType, err := r.dbaasInstanceTypes.Get(ctx, detail.InstanceTypeID)
	if err != nil {
		resp.Diagnostics.AddError(utils.ParseSDKError(err))
		return
	}

	data.InstanceType = types.StringValue(instanceType.Label)
	data.Status = types.StringValue(string(detail.Status))
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *DBaaSReplicaResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan DBaaSReplicaModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var currentData DBaaSReplicaModel
	resp.Diagnostics.Append(req.State.Get(ctx, &currentData)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if plan.InstanceType.ValueString() != "" && currentData.InstanceType.ValueString() != plan.InstanceType.ValueString() {
		instanceTypeID, err := ValidateAndGetInstanceTypeID(ctx, r.dbaasInstanceTypes.List, plan.InstanceType.ValueString(),
			currentData.EngineID.ValueString(), dbaasReplicaProductFamily)
		if err != nil {
			resp.Diagnostics.AddError(utils.ParseSDKError(err))
			return
		}

		_, err = r.dbaasReplicas.Resize(ctx, currentData.ID.ValueString(), dbSDK.ReplicaResizeRequest{
			InstanceTypeID: instanceTypeID,
		})
		if err != nil {
			resp.Diagnostics.AddError(utils.ParseSDKError(err))
			return
		}

		if _, err := r.waitUntilReplicaStatusMatches(ctx, currentData.ID.ValueString(), DBaaSInstanceStatusActive.String()); err != nil {
			resp.Diagnostics.AddError("Error waiting for replica to be active", err.Error())
			return
		}
	}

	currentData.InstanceType = plan.InstanceType
	resp.Diagnostics.Append(resp.State.Set(ctx, &currentData)...)
}

func (r *DBaaSReplicaResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data DBaaSReplicaModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.dbaasReplicas.Delete(ctx, data.ID.ValueString()); err != nil {
		resp.Diagnostics.AddError(utils.ParseSDKError(err))
	}

	instanceID := data.ID.ValueString()
	replica, err := r.dbaasReplicas.Get(ctx, instanceID)

	if err != nil && strings.Contains(err.Error(), strconv.Itoa(http.StatusNotFound)) {
		return
	}
	if DBaaSInstanceStatus(replica.Status) != DBaaSInstanceStatusDeleting {
		err := r.dbaasInstances.Delete(ctx, string(instanceID))
		if err != nil {
			resp.Diagnostics.AddError(utils.ParseSDKError(err))
		}
	}

	if _, err := r.waitUntilReplicaIsDeleted(ctx, instanceID); err != nil {
		resp.Diagnostics.AddError(utils.ParseSDKError(err))
		return
	}
}

func (r *DBaaSReplicaResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), types.StringValue(req.ID))...)
}

func (r *DBaaSReplicaResource) waitUntilReplicaStatusMatches(ctx context.Context, instanceID string, status string) (*dbSDK.ReplicaDetailResponse, error) {
	timeoutCtx, cancel := context.WithTimeout(ctx, instanceStatusTimeout)
	defer cancel()
	for {
		select {
		case <-timeoutCtx.Done():
			return nil, fmt.Errorf("timeout waiting for replica %s to reach status %s", instanceID, status)
		case <-time.After(10 * time.Second):
			instance, err := r.dbaasReplicas.Get(ctx, instanceID)
			if err != nil {
				return nil, err
			}
			currentStatus := DBaaSInstanceStatus(instance.Status)
			if currentStatus.String() == status {
				return instance, nil
			}
			if currentStatus.IsAnyError() {
				return nil, fmt.Errorf("replica %s is in error status %s", instanceID, string(currentStatus))
			}
		}
	}
}

func (r *DBaaSReplicaResource) waitUntilReplicaIsDeleted(ctx context.Context, instanceID string) (*dbSDK.ReplicaDetailResponse, error) {
	timeoutCtx, cancel := context.WithTimeout(ctx, instanceStatusTimeout)
	defer cancel()
	for {
		select {
		case <-timeoutCtx.Done():
			return nil, fmt.Errorf("timeout waiting for instance %s to be deleted", instanceID)
		case <-time.After(10 * time.Second):
			replica, err := r.dbaasReplicas.Get(ctx, instanceID)
			if err != nil && strings.Contains(err.Error(), strconv.Itoa(http.StatusNotFound)) {
				return nil, nil
			}
			if err != nil {
				return nil, err
			}
			currentStatus := DBaaSInstanceStatus(replica.Status)
			if currentStatus.IsAnyError() {
				return nil, fmt.Errorf("replica %s is in error status %s", instanceID, string(currentStatus))
			}
		}
	}
}
