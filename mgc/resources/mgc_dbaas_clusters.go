package resources

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	dbSDK "github.com/MagaluCloud/mgc-sdk-go/dbaas"
	"github.com/MagaluCloud/terraform-provider-mgc/mgc/tfutil"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

const (
	clusterStatusTimeout  = 90 * time.Minute
	clusterInstanceFamily = "CLUSTER"
)

type DBaaSClusterAddressModel struct {
	Access  types.String `tfsdk:"access"`
	Type    types.String `tfsdk:"type"`
	Address types.String `tfsdk:"address"`
	Port    types.String `tfsdk:"port"`
}

type DBaaSClusterModel struct {
	ID                     types.String               `tfsdk:"id"`
	Name                   types.String               `tfsdk:"name"`
	User                   types.String               `tfsdk:"user"`
	Password               types.String               `tfsdk:"password"`
	EngineName             types.String               `tfsdk:"engine_name"`
	EngineVersion          types.String               `tfsdk:"engine_version"`
	InstanceType           types.String               `tfsdk:"instance_type"`
	VolumeSize             types.Int64                `tfsdk:"volume_size"`
	VolumeType             types.String               `tfsdk:"volume_type"`
	ParameterGroupID       types.String               `tfsdk:"parameter_group_id"`
	BackupRetentionDays    types.Int64                `tfsdk:"backup_retention_days"`
	BackupStartAt          types.String               `tfsdk:"backup_start_at"`
	Status                 types.String               `tfsdk:"status"`
	Addresses              []DBaaSClusterAddressModel `tfsdk:"addresses"`
	ApplyParametersPending types.Bool                 `tfsdk:"apply_parameters_pending"`
	CreatedAt              types.String               `tfsdk:"created_at"`
	UpdatedAt              types.String               `tfsdk:"updated_at"`
	StartedAt              types.String               `tfsdk:"started_at"`
	FinishedAt             types.String               `tfsdk:"finished_at"`
	InstanceTypeID         types.String               `tfsdk:"instance_type_id"`
	EngineID               types.String               `tfsdk:"engine_id"`
}

type DBaaSClusterResource struct {
	clusterService      dbSDK.ClusterService
	engineService       dbSDK.EngineService
	instanceTypeService dbSDK.InstanceTypeService
}

func NewDBaaSClusterResource() resource.Resource {
	return &DBaaSClusterResource{}
}

func (r *DBaaSClusterResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dbaas_clusters"
}

func (r *DBaaSClusterResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	dataConfig, ok := req.ProviderData.(tfutil.DataConfig)
	if !ok {
		resp.Diagnostics.AddError("Failed to get provider data", "Provider data has unexpected type")
		return
	}

	sdkClient := dbSDK.New(&dataConfig.CoreConfig)
	r.clusterService = sdkClient.Clusters()
	r.engineService = sdkClient.Engines()
	r.instanceTypeService = sdkClient.InstanceTypes()
}

func (r *DBaaSClusterResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a DBaaS (Database-as-a-Service) Cluster.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Unique identifier for the DBaaS cluster. Generated automatically on creation.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "Name of the DBaaS cluster. Must be unique. Cannot be changed after creation.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
					stringvalidator.LengthAtMost(100),
				},
			},
			"user": schema.StringAttribute{
				Description: "Master username for the database cluster. Must start with a letter and contain only alphanumeric characters.",
				Required:    true,
				WriteOnly:   true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
					stringvalidator.LengthAtMost(25),
					stringvalidator.RegexMatches(
						regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9]*$`),
						"must start with a letter and contain only alphanumeric characters",
					),
				},
			},
			"password": schema.StringAttribute{
				Description: "Master password for the database cluster. Must be at least 8 characters long.",
				Required:    true,
				Sensitive:   true,
				WriteOnly:   true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(8),
					stringvalidator.LengthAtMost(50),
				},
			},
			"engine_name": schema.StringAttribute{
				Description: "Type of database engine to use (e.g., 'mysql', 'postgresql'). Cannot be changed after creation.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"engine_version": schema.StringAttribute{
				Description: "Version of the database engine (e.g., '8.0', '13.3'). Must be compatible with the selected engine_name. Cannot be changed after creation.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"engine_id": schema.StringAttribute{
				Description: "ID of the database engine.",
				Computed:    true,
			},
			"instance_type": schema.StringAttribute{
				Description: "Compute and memory capacity of the cluster nodes (e.g., 'BV1-4-10'). Cannot be changed after creation.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"instance_type_id": schema.StringAttribute{
				Description: "ID of the instance type.",
				Computed:    true,
			},
			"volume_size": schema.Int64Attribute{
				Description: "Size of the storage volume in GB. Cannot be changed after creation.",
				Required:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
				Validators: []validator.Int64{
					int64validator.Between(10, 50000),
				},
			},
			"volume_type": schema.StringAttribute{
				Description: "Type of the storage volume (e.g., 'CLOUD_NVME15K' or 'CLOUD_NVME20K').",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"parameter_group_id": schema.StringAttribute{
				Description: "ID of the parameter group to associate with the cluster.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"backup_retention_days": schema.Int64Attribute{
				Description: "Number of days to retain automated backups (1-35 days). Default is 7 days.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
				Validators: []validator.Int64{
					int64validator.Between(1, 35),
				},
			},
			"backup_start_at": schema.StringAttribute{
				Description: "Time to initiate the daily backup in UTC (format: 'HH:MM:SS'). Default is '04:00:00'.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
				Validators: []validator.String{
					stringvalidator.RegexMatches(
						regexp.MustCompile(`^([0-1][0-9]|2[0-3]):[0-5][0-9]:[0-5][0-9]$`),
						"must be in format HH:MM:SS (24-hour)",
					),
				},
			},
			"status": schema.StringAttribute{
				Description: "Current status of the DBaaS cluster.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"addresses": schema.ListNestedAttribute{
				Description: "Network addresses for connecting to the cluster.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"access": schema.StringAttribute{
							Description: "Access type (e.g., 'public', 'private').",
							Computed:    true,
							PlanModifiers: []planmodifier.String{
								stringplanmodifier.UseStateForUnknown(),
							},
						},
						"type": schema.StringAttribute{
							Description: "Address type (e.g., 'read-write', 'read-only').",
							Computed:    true,
							PlanModifiers: []planmodifier.String{
								stringplanmodifier.UseStateForUnknown(),
							},
						},
						"address": schema.StringAttribute{
							Description: "The IP address or hostname.",
							Computed:    true,
							PlanModifiers: []planmodifier.String{
								stringplanmodifier.UseStateForUnknown(),
							},
						},
						"port": schema.StringAttribute{
							Description: "The port number.",
							Computed:    true,
							PlanModifiers: []planmodifier.String{
								stringplanmodifier.UseStateForUnknown(),
							},
						},
					},
				},
			},
			"apply_parameters_pending": schema.BoolAttribute{
				Description: "Indicates if parameter changes are pending application.",
				Computed:    true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"created_at": schema.StringAttribute{
				Description: "Timestamp of when the cluster was created.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"updated_at": schema.StringAttribute{
				Description: "Timestamp of when the cluster was last updated.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"started_at": schema.StringAttribute{
				Description: "Timestamp of when the cluster was last started.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"finished_at": schema.StringAttribute{
				Description: "Timestamp of when the cluster last finished an operation.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *DBaaSClusterResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan DBaaSClusterModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	engineID, err := tfutil.ValidateAndGetEngineID(ctx, r.engineService.List, plan.EngineName.ValueString(), plan.EngineVersion.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid Engine", fmt.Sprintf("Failed to validate engine '%s' version '%s': %s", plan.EngineName.ValueString(), plan.EngineVersion.ValueString(), err.Error()))
		return
	}

	instanceTypeID, err := tfutil.ValidateAndGetInstanceTypeID(ctx, r.instanceTypeService.List, plan.InstanceType.ValueString(), engineID, clusterInstanceFamily)
	if err != nil {
		resp.Diagnostics.AddError("Invalid Instance Type", fmt.Sprintf("Failed to validate instance type '%s': %s", plan.InstanceType.ValueString(), err.Error()))
		return
	}

	createReq := dbSDK.ClusterCreateRequest{
		Name:           plan.Name.ValueString(),
		EngineID:       engineID,
		InstanceTypeID: instanceTypeID,
		User:           plan.User.ValueString(),
		Password:       plan.Password.ValueString(),
		Volume: dbSDK.ClusterVolumeRequest{
			Size: int(plan.VolumeSize.ValueInt64()),
			Type: plan.VolumeType.ValueStringPointer(),
		},
		ParameterGroupID:    plan.ParameterGroupID.ValueStringPointer(),
		BackupRetentionDays: tfutil.ConvertInt64PointerToIntPointer(plan.BackupRetentionDays.ValueInt64Pointer()),
		BackupStartAt:       plan.BackupStartAt.ValueStringPointer(),
	}

	clusterResp, err := r.clusterService.Create(ctx, createReq)
	if err != nil {
		resp.Diagnostics.AddError(tfutil.ParseSDKError(err))
		return
	}

	plan.ID = types.StringValue(clusterResp.ID)
	plan.EngineID = types.StringValue(engineID)
	plan.InstanceTypeID = types.StringValue(instanceTypeID)

	getCluster, err := r.waitUntilClusterStatusMatches(ctx, clusterResp.ID, dbSDK.ClusterStatusActive)
	if err != nil {
		resp.Diagnostics.AddError("Cluster Creation Error", fmt.Sprintf("Error waiting for cluster %s to become active: %s", clusterResp.ID, err.Error()))
		return
	}

	r.populateModelFromDetailResponse(getCluster, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *DBaaSClusterResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state DBaaSClusterModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	detailedCluster, err := r.clusterService.Get(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(tfutil.ParseSDKError(err))
		return
	}

	r.populateModelFromDetailResponse(detailedCluster, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *DBaaSClusterResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan DBaaSClusterModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state DBaaSClusterModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	clusterID := state.ID.ValueString()
	updateReq := dbSDK.ClusterUpdateRequest{}
	changed := false

	if !plan.ParameterGroupID.Equal(state.ParameterGroupID) {
		updateReq.ParameterGroupID = plan.ParameterGroupID.ValueStringPointer()
		state.ParameterGroupID = plan.ParameterGroupID
		changed = true
	}
	if !plan.BackupRetentionDays.Equal(state.BackupRetentionDays) {
		updateReq.BackupRetentionDays = tfutil.ConvertInt64PointerToIntPointer(plan.BackupRetentionDays.ValueInt64Pointer())
		state.BackupRetentionDays = plan.BackupRetentionDays
		changed = true
	}
	if !plan.BackupStartAt.Equal(state.BackupStartAt) {
		updateReq.BackupStartAt = plan.BackupStartAt.ValueStringPointer()
		state.BackupStartAt = plan.BackupStartAt
		changed = true
	}

	if changed {
		_, err := r.clusterService.Update(ctx, clusterID, updateReq)
		if err != nil {
			resp.Diagnostics.AddError(tfutil.ParseSDKError(err))
			return
		}
		_, err = r.waitUntilClusterStatusMatches(ctx, clusterID, dbSDK.ClusterStatusActive)
		if err != nil {
			resp.Diagnostics.AddError("Cluster Update Error", fmt.Sprintf("Error waiting for cluster %s to become stable after update: %s", clusterID, err.Error()))
			return
		}
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *DBaaSClusterResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state DBaaSClusterModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.clusterService.Delete(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(tfutil.ParseSDKError(err))
		return
	}
}

func (r *DBaaSClusterResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *DBaaSClusterResource) populateModelFromDetailResponse(detail *dbSDK.ClusterDetailResponse, model *DBaaSClusterModel) {
	model.ID = types.StringValue(detail.ID)
	model.Name = types.StringValue(detail.Name)
	model.VolumeSize = types.Int64Value(int64(detail.Volume.Size))
	model.VolumeType = types.StringValue(detail.Volume.Type)
	model.ParameterGroupID = types.StringValue(detail.ParameterGroupID)
	if detail.ParameterGroupID == "" {
		model.ParameterGroupID = types.StringNull()
	}
	model.BackupRetentionDays = types.Int64Value(int64(detail.BackupRetentionDays))
	model.BackupStartAt = types.StringValue(detail.BackupStartAt)
	model.Status = types.StringValue(string(detail.Status))
	model.ApplyParametersPending = types.BoolValue(detail.ApplyParametersPending)
	model.InstanceTypeID = types.StringValue(detail.InstanceTypeID)
	model.EngineID = types.StringValue(detail.EngineID)

	model.CreatedAt = types.StringValue(detail.CreatedAt.Format(time.RFC3339))
	if detail.UpdatedAt != nil {
		model.UpdatedAt = types.StringValue(detail.UpdatedAt.Format(time.RFC3339))
	} else {
		model.UpdatedAt = types.StringNull()
	}
	model.StartedAt = types.StringPointerValue(detail.StartedAt)
	model.FinishedAt = types.StringPointerValue(detail.FinishedAt)

	var modelAddresses []DBaaSClusterAddressModel
	for _, lba := range detail.Addresses {
		modelAddresses = append(modelAddresses, DBaaSClusterAddressModel{
			Address: types.StringValue(lba.Address),
			Port:    types.StringValue(lba.Port),
			Access:  types.StringValue(string(lba.Access)),
			Type:    types.StringValue(string(lba.Type)),
		})
	}
	model.Addresses = modelAddresses
	model.Password = types.StringNull()
	model.User = types.StringNull()
}

func (r *DBaaSClusterResource) waitUntilClusterStatusMatches(ctx context.Context, clusterID string, targetStatus dbSDK.ClusterStatus) (*dbSDK.ClusterDetailResponse, error) {
	timeoutCtx, cancel := context.WithTimeout(ctx, clusterStatusTimeout)
	defer cancel()
	for {
		select {
		case <-timeoutCtx.Done():
			return nil, fmt.Errorf("timeout waiting for cluster %s to reach status %s", clusterID, targetStatus)
		case <-time.After(15 * time.Second):
			cluster, err := r.clusterService.Get(ctx, clusterID)
			if err != nil {
				return nil, fmt.Errorf("failed to get cluster %s during status wait: %w", clusterID, err)
			}

			currentStatus := cluster.Status
			if currentStatus == targetStatus {
				if targetStatus == dbSDK.ClusterStatusActive && cluster.ApplyParametersPending {
					continue
				}
				return cluster, nil
			}

			if strings.Contains(strings.ToLower(string(currentStatus)), "error") {
				return nil, fmt.Errorf("cluster %s entered error state: %s", clusterID, currentStatus)
			}
		}
	}
}
