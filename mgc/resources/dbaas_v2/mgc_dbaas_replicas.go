package resources

import (
	"context"
	"fmt"
	"time"

	dbSDK "github.com/MagaluCloud/mgc-sdk-go/dbaas/v2"
	"github.com/MagaluCloud/terraform-provider-mgc/mgc/tfutil"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type DBaaSReplicaModel struct {
	ID             types.String `tfsdk:"id"`
	SourceID       types.String `tfsdk:"source_id"`
	Name           types.String `tfsdk:"name"`
	EngineID       types.String `tfsdk:"engine_id"`
	InstanceTypeID types.String `tfsdk:"instance_type_id"`
	Status         types.String `tfsdk:"status"`
}

type DBaaSReplicaResource struct {
	service dbSDK.ReplicaService
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
	cfg, ok := req.ProviderData.(tfutil.DataConfig)
	if !ok {
		resp.Diagnostics.AddError("invalid provider data", "expected tfutil.DataConfig")
		return
	}
	r.service = dbSDK.New(&cfg.CoreConfig).Replicas()
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
			"instance_type_id": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Instance type ID",
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
	if !data.InstanceTypeID.IsNull() && data.InstanceTypeID.ValueString() != "" {
		v := data.InstanceTypeID.ValueString()
		ptrTypeID = &v
	}
	created, err := r.service.Create(ctx, dbSDK.ReplicaCreateRequest{
		SourceID:       data.SourceID.ValueString(),
		Name:           data.Name.ValueString(),
		InstanceTypeID: ptrTypeID,
	})
	if err != nil {
		resp.Diagnostics.AddError(tfutil.ParseSDKError(err))
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
	data.InstanceTypeID = types.StringValue(found.InstanceTypeID)
	data.Status = types.StringValue(string(found.Status))
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *DBaaSReplicaResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data DBaaSReplicaModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	detail, err := r.service.Get(ctx, data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(tfutil.ParseSDKError(err))
		return
	}
	data.SourceID = types.StringValue(detail.SourceID)
	data.Name = types.StringValue(detail.Name)
	data.EngineID = types.StringValue(detail.EngineID)
	data.InstanceTypeID = types.StringValue(detail.InstanceTypeID)
	data.Status = types.StringValue(string(detail.Status))
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *DBaaSReplicaResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan DBaaSReplicaModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if plan.InstanceTypeID.ValueString() != "" {
		_, err := r.service.Resize(ctx, plan.ID.ValueString(), dbSDK.ReplicaResizeRequest{
			InstanceTypeID: plan.InstanceTypeID.ValueString(),
		})
		if err != nil {
			resp.Diagnostics.AddError(tfutil.ParseSDKError(err))
			return
		}
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *DBaaSReplicaResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data DBaaSReplicaModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.service.Delete(ctx, data.ID.ValueString()); err != nil {
		resp.Diagnostics.AddError(tfutil.ParseSDKError(err))
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
			instance, err := r.service.Get(ctx, instanceID)
			if err != nil {
				return nil, err
			}
			currentStatus := DBaaSInstanceStatus(instance.Status)
			if currentStatus.String() == status {
				return instance, nil
			}
			if currentStatus.IsError() {
				return nil, fmt.Errorf("replica %s is in error state", instanceID)
			}
		}
	}
}
