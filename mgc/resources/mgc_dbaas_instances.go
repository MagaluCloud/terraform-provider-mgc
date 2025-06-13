package resources

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	dbSDK "github.com/MagaluCloud/mgc-sdk-go/dbaas"
	"github.com/MagaluCloud/terraform-provider-mgc/mgc/tfutil"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

const (
	instanceStatusTimeout      = 90 * time.Minute
	dbaasInstanceProductFamily = "SINGLE_INSTANCE"
)

type DBaaSInstanceStatus string

const (
	DBaaSInstanceStatusCreating      DBaaSInstanceStatus = "CREATING"
	DBaaSInstanceStatusError         DBaaSInstanceStatus = "ERROR"
	DBaaSInstanceStatusStopped       DBaaSInstanceStatus = "STOPPED"
	DBaaSInstanceStatusReboot        DBaaSInstanceStatus = "REBOOT"
	DBaaSInstanceStatusPending       DBaaSInstanceStatus = "PENDING"
	DBaaSInstanceStatusResizing      DBaaSInstanceStatus = "RESIZING"
	DBaaSInstanceStatusDeleted       DBaaSInstanceStatus = "DELETED"
	DBaaSInstanceStatusActive        DBaaSInstanceStatus = "ACTIVE"
	DBaaSInstanceStatusStarting      DBaaSInstanceStatus = "STARTING"
	DBaaSInstanceStatusStopping      DBaaSInstanceStatus = "STOPPING"
	DBaaSInstanceStatusBackingUp     DBaaSInstanceStatus = "BACKING_UP"
	DBaaSInstanceStatusDeleting      DBaaSInstanceStatus = "DELETING"
	DBaaSInstanceStatusRestoring     DBaaSInstanceStatus = "RESTORING"
	DBaaSInstanceStatusErrorDeleting DBaaSInstanceStatus = "ERROR_DELETING"
	DBaaSInstanceStatusMaintenance   DBaaSInstanceStatus = "MAINTENANCE"
)

func (s DBaaSInstanceStatus) String() string {
	return string(s)
}

func (s DBaaSInstanceStatus) IsError() bool {
	return strings.Contains(string(s), "ERROR")
}

type DBaaSInstanceModel struct {
	Id                  types.String `tfsdk:"id"`
	Name                types.String `tfsdk:"name"`
	User                types.String `tfsdk:"user"`
	Password            types.String `tfsdk:"password"`
	EngineName          types.String `tfsdk:"engine_name"`
	EngineVersion       types.String `tfsdk:"engine_version"`
	InstanceType        types.String `tfsdk:"instance_type"`
	VolumeSize          types.Int64  `tfsdk:"volume_size"`
	BackupRetentionDays types.Int64  `tfsdk:"backup_retention_days"`
	BackupStartAt       types.String `tfsdk:"backup_start_at"`
	AvailabilityZone    types.String `tfsdk:"availability_zone"`
	ParameterGroup      types.String `tfsdk:"parameter_group"`
	Status              types.String `tfsdk:"status"`
}

type DBaaSInstanceResource struct {
	dbaasInstances     dbSDK.InstanceService
	dbaasEngines       dbSDK.EngineService
	dbaasInstanceTypes dbSDK.InstanceTypeService
}

func NewDBaaSInstanceResource() resource.Resource {
	return &DBaaSInstanceResource{}
}

func (r *DBaaSInstanceResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dbaas_instances"
}

func (r *DBaaSInstanceResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	dataConfig, ok := req.ProviderData.(tfutil.DataConfig)
	if !ok {
		resp.Diagnostics.AddError("Failed to get provider data", "Failed to get provider data")
		return
	}

	r.dbaasInstances = dbSDK.New(&dataConfig.CoreConfig).Instances()
	r.dbaasEngines = dbSDK.New(&dataConfig.CoreConfig).Engines()
	r.dbaasInstanceTypes = dbSDK.New(&dataConfig.CoreConfig).InstanceTypes()
}

func (r *DBaaSInstanceResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a DBaaS (Database-as-a-Service) instance",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Unique identifier for the DBaaS instance. Generated automatically on creation.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "Name of the DBaaS instance. Must be unique within your account. Cannot be changed after creation.",
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
				Description: "Master username for the database. Must start with a letter and contain only alphanumeric characters.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
					stringvalidator.LengthAtMost(25),
					stringvalidator.RegexMatches(
						regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9]*$`),
						"must start with a letter and contain only alphanumeric characters",
					),
				},
				Required:  true,
				WriteOnly: true,
			},
			"password": schema.StringAttribute{
				Description: "Master password for the database. Must be at least 8 characters long and contain letters, numbers and special characters.",
				Required:    true,
				Sensitive:   true,
				WriteOnly:   true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
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
				Description: "Version of the database engine (e.g., '8.0', '13.3'). Must be compatible with the selected engine_name.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"instance_type": schema.StringAttribute{
				Description: "Compute and memory capacity of the instance (e.g., 'BV1-4-10'). Can be changed to scale the instance.",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"volume_size": schema.Int64Attribute{
				Description: "Size of the storage volume in GB. Can be increased but not decreased after creation.",
				Required:    true,
				Validators: []validator.Int64{
					int64validator.Between(10, 50000),
				},
			},
			"backup_retention_days": schema.Int64Attribute{
				Description: "Number of days to retain automated backups (1-35 days). Zero disables automated backups. Default is 7 days.",
				Optional:    true,
				Computed:    true,
				Validators: []validator.Int64{
					int64validator.Between(1, 35),
				},
			},
			"backup_start_at": schema.StringAttribute{
				Description: "Time to initiate the daily backup in UTC (format: 'HH:MM:SS'). Default is 04:00:00.",
				Optional:    true,
				Computed:    true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
					stringvalidator.RegexMatches(
						regexp.MustCompile(`^([0-1][0-9]|2[0-3]):[0-5][0-9]:[0-5][0-9]$`),
						"must be in format HH:MM:SS (24-hour)",
					),
				},
			},
			"parameter_group": schema.StringAttribute{
				Description: "ID of the parameter group to use for the instance.",
				Optional:    true,
				Computed:    true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"availability_zone": schema.StringAttribute{
				Description: "Availability zone to use for the instance.",
				Optional:    true,
				Computed:    true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"status": schema.StringAttribute{
				Description: "Status of the instance.",
				Computed:    true,
			},
		},
	}
}

func (r *DBaaSInstanceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data DBaaSInstanceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	engineID, err := r.validateAndGetEngineID(ctx, data.EngineName.ValueString(), data.EngineVersion.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid engine name", err.Error())
		return
	}

	instanceTypeID, err := r.validateAndGetInstanceTypeID(ctx, data.InstanceType.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid instance type", err.Error())
		return
	}

	params := dbSDK.InstanceCreateRequest{
		Name:           data.Name.ValueString(),
		User:           data.User.ValueString(),
		Password:       data.Password.ValueString(),
		EngineID:       &engineID,
		InstanceTypeID: &instanceTypeID,
		Volume: dbSDK.InstanceVolumeRequest{
			Size: *tfutil.ConvertInt64PointerToIntPointer(data.VolumeSize.ValueInt64Pointer()),
		},
		BackupRetentionDays: tfutil.ConvertInt64PointerToIntPointer(data.BackupRetentionDays.ValueInt64Pointer()),
		BackupStartAt:       data.BackupStartAt.ValueStringPointer(),
		AvailabilityZone:    data.AvailabilityZone.ValueStringPointer(),
		ParameterGroupID:    data.ParameterGroup.ValueStringPointer(),
	}

	created, err := r.dbaasInstances.Create(ctx, params)
	if err != nil {
		resp.Diagnostics.AddError(tfutil.ParseSDKError(err))
		return
	}

	data.Id = types.StringValue(created.ID)
	data.Password = types.StringNull()
	data.User = types.StringNull()
	result, err := r.waitUntilInstanceStatusMatches(ctx, data.Id.ValueString(), DBaaSInstanceStatusActive.String())
	if err != nil {
		resp.Diagnostics.AddError(tfutil.ParseSDKError(err))
		return
	}
	data.Status = types.StringValue(string(result.Status))
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *DBaaSInstanceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data DBaaSInstanceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	instance, err := r.dbaasInstances.Get(ctx, data.Id.ValueString(), dbSDK.GetInstanceOptions{})
	if err != nil {
		resp.Diagnostics.AddError(tfutil.ParseSDKError(err))
		return
	}

	engineName, engineVersion, err := r.getEngineNameAndVersionByID(ctx, instance.EngineID)
	if err != nil {
		resp.Diagnostics.AddError(tfutil.ParseSDKError(err))
		return
	}

	instanceTypeName, err := r.getInstanceTypeNameByID(ctx, instance.InstanceTypeID)
	if err != nil {
		resp.Diagnostics.AddError(tfutil.ParseSDKError(err))
		return
	}

	data.Name = types.StringValue(instance.Name)
	data.EngineName = types.StringValue(engineName)
	data.EngineVersion = types.StringValue(engineVersion)
	data.InstanceType = types.StringValue(instanceTypeName)
	data.VolumeSize = types.Int64Value(int64(instance.Volume.Size))
	data.BackupRetentionDays = types.Int64Value(int64(instance.BackupRetentionDays))
	data.BackupStartAt = types.StringValue(instance.BackupStartAt)
	data.AvailabilityZone = types.StringValue(instance.AvailabilityZone)
	data.ParameterGroup = types.StringValue(instance.ParameterGroupID)
	data.Status = types.StringValue(string(instance.Status))
	data.Password = types.StringNull()
	data.User = types.StringNull()
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *DBaaSInstanceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var planData DBaaSInstanceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &planData)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var currentData DBaaSInstanceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &currentData)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if planData.InstanceType.ValueString() != currentData.InstanceType.ValueString() {
		currentData.InstanceType = planData.InstanceType
		instanceTypeID, err := r.validateAndGetInstanceTypeID(ctx, planData.InstanceType.ValueString())
		if err != nil {
			resp.Diagnostics.AddError(tfutil.ParseSDKError(err))
			return
		}

		_, err = r.dbaasInstances.Resize(ctx, currentData.Id.ValueString(), dbSDK.InstanceResizeRequest{
			InstanceTypeID: &instanceTypeID,
		})
		if err != nil {
			resp.Diagnostics.AddError(tfutil.ParseSDKError(err))
			return
		}

		if _, err := r.waitUntilInstanceStatusMatches(ctx, planData.Id.ValueString(), DBaaSInstanceStatusActive.String()); err != nil {
			resp.Diagnostics.AddError(tfutil.ParseSDKError(err))
			return
		}
	}

	if planData.VolumeSize.ValueInt64() != currentData.VolumeSize.ValueInt64() {
		currentData.VolumeSize = planData.VolumeSize
		_, err := r.dbaasInstances.Resize(ctx, currentData.Id.ValueString(), dbSDK.InstanceResizeRequest{
			Volume: &dbSDK.InstanceVolumeResizeRequest{
				Size: *tfutil.ConvertInt64PointerToIntPointer(planData.VolumeSize.ValueInt64Pointer()),
			},
		})
		if err != nil {
			resp.Diagnostics.AddError(tfutil.ParseSDKError(err))
			return
		}

		if _, err := r.waitUntilInstanceStatusMatches(ctx, planData.Id.ValueString(), DBaaSInstanceStatusActive.String()); err != nil {
			resp.Diagnostics.AddError(tfutil.ParseSDKError(err))
			return
		}
	}

	if (planData.BackupRetentionDays.ValueInt64() != currentData.BackupRetentionDays.ValueInt64()) || (planData.BackupStartAt.ValueString() != currentData.BackupStartAt.ValueString()) {
		currentData.BackupRetentionDays = planData.BackupRetentionDays
		currentData.BackupStartAt = planData.BackupStartAt

		_, err := r.dbaasInstances.Update(ctx, planData.Id.ValueString(), dbSDK.DatabaseInstanceUpdateRequest{
			BackupRetentionDays: tfutil.ConvertInt64PointerToIntPointer(planData.BackupRetentionDays.ValueInt64Pointer()),
			BackupStartAt:       planData.BackupStartAt.ValueStringPointer(),
		})
		if err != nil {
			resp.Diagnostics.AddError(tfutil.ParseSDKError(err))
			return
		}

		if _, err := r.waitUntilInstanceStatusMatches(ctx, planData.Id.ValueString(), DBaaSInstanceStatusActive.String()); err != nil {
			resp.Diagnostics.AddError(tfutil.ParseSDKError(err))
			return
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &currentData)...)
}

func (r *DBaaSInstanceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data DBaaSInstanceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.dbaasInstances.Delete(ctx, data.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(tfutil.ParseSDKError(err))
		return
	}
}

func (r *DBaaSInstanceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	data := DBaaSInstanceModel{}
	data.Id = types.StringValue(req.ID)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *DBaaSInstanceResource) validateAndGetEngineID(ctx context.Context, engineName string, engineVersion string) (string, error) {
	active := "ACTIVE"
	engines, err := r.dbaasEngines.List(ctx, dbSDK.ListEngineOptions{
		Status: &active,
	})
	if err != nil {
		return "", err
	}
	for _, engine := range engines {
		if engine.Name == engineName && engine.Version == engineVersion {
			return engine.ID, nil
		}
	}
	return "", errors.New("engine not found")
}

func (r *DBaaSInstanceResource) validateAndGetInstanceTypeID(ctx context.Context, instanceType string) (string, error) {
	active := "ACTIVE"
	maxLimit := 50
	instanceTypes, err := r.dbaasInstanceTypes.List(ctx, dbSDK.ListInstanceTypeOptions{
		Limit:  &maxLimit,
		Status: &active,
	})
	if err != nil {
		return "", err
	}
	for _, instance := range instanceTypes {
		if instance.Label == instanceType && instance.CompatibleProduct == dbaasInstanceProductFamily {
			return instance.ID, nil
		}
	}
	return "", errors.New("instance type not found, not active or not compatible with single instance family")
}

func (r *DBaaSInstanceResource) getEngineNameAndVersionByID(ctx context.Context, engineID string) (name string, version string, err error) {
	engine, err := r.dbaasEngines.Get(ctx, engineID)
	if err != nil {
		return "", "", err
	}
	return engine.Name, engine.Version, nil
}

func (r *DBaaSInstanceResource) getInstanceTypeNameByID(ctx context.Context, instanceTypeID string) (string, error) {
	instanceType, err := r.dbaasInstanceTypes.Get(ctx, instanceTypeID)
	if err != nil {
		return "", err
	}
	return instanceType.Label, nil
}

func (r *DBaaSInstanceResource) waitUntilInstanceStatusMatches(ctx context.Context, instanceID string, status string) (*dbSDK.InstanceDetail, error) {
	timeoutCtx, cancel := context.WithTimeout(ctx, instanceStatusTimeout)
	defer cancel()
	for {
		select {
		case <-timeoutCtx.Done():
			return nil, fmt.Errorf("timeout waiting for instance %s to reach status %s", instanceID, status)
		case <-time.After(10 * time.Second):
			instance, err := r.dbaasInstances.Get(ctx, instanceID, dbSDK.GetInstanceOptions{})
			if err != nil {
				return nil, err
			}
			currentStatus := DBaaSInstanceStatus(instance.Status)
			if currentStatus.String() == status {
				return instance, nil
			}
			if currentStatus.IsError() {
				return nil, fmt.Errorf("instance %s is in error state", instanceID)
			}
		}
	}
}
