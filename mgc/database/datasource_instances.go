package database

import (
	"context"

	dbSDK "github.com/MagaluCloud/mgc-sdk-go/dbaas"

	"github.com/MagaluCloud/terraform-provider-mgc/mgc/utils"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &DataSourceDbInstances{}

type DataSourceDbInstances struct {
	dbaasInstances dbSDK.InstanceService
}

type dbInstanceModel struct {
	Instances []dbInstance `tfsdk:"instances"`
	Status    types.String `tfsdk:"status"`
}

type dbInstance struct {
	Addresses           []InstanceAddress `tfsdk:"addresses"`
	BackupRetentionDays types.Int64       `tfsdk:"backup_retention_days"`
	CreatedAt           types.String      `tfsdk:"created_at"`
	EngineID            types.String      `tfsdk:"engine_id"`
	ID                  types.String      `tfsdk:"id"`
	InstanceTypeID      types.String      `tfsdk:"instance_type_id"`
	Name                types.String      `tfsdk:"name"`
	Status              types.String      `tfsdk:"status"`
	VolumeSize          types.Int64       `tfsdk:"volume_size"`
	VolumeType          types.String      `tfsdk:"volume_type"`
	AvailabilityZone    types.String      `tfsdk:"availability_zone"`
	ParameterGroup      types.String      `tfsdk:"parameter_group"`
}

type InstanceAddress struct {
	Access  types.String `tfsdk:"access"`
	Address types.String `tfsdk:"address"`
	Type    types.String `tfsdk:"type"`
}

func NewDataSourceDbaasInstances() datasource.DataSource {
	return &DataSourceDbInstances{}
}

func (r *DataSourceDbInstances) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dbaas_instances"
}

func (r *DataSourceDbInstances) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	dataConfig, ok := req.ProviderData.(utils.DataConfig)
	if !ok {
		resp.Diagnostics.AddError("Failed to get provider data", "Failed to get provider data")
		return
	}

	r.dbaasInstances = dbSDK.New(&dataConfig.CoreConfig).Instances()
}

func (r *DataSourceDbInstances) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "A list of database instances.",
		Attributes: map[string]schema.Attribute{
			"instances": schema.ListNestedAttribute{
				Description: "List of database instances",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"addresses": schema.ListNestedAttribute{
							Description: "List of instance addresses",
							Computed:    true,
							NestedObject: schema.NestedAttributeObject{
								Attributes: map[string]schema.Attribute{
									"access": schema.StringAttribute{
										Description: "Access type of the address",
										Computed:    true,
									},
									"address": schema.StringAttribute{
										Description: "IP address",
										Computed:    true,
									},
									"type": schema.StringAttribute{
										Description: "Type of the address",
										Computed:    true,
									},
								},
							},
						},
						"backup_retention_days": schema.Int64Attribute{
							Description: "Number of days to retain backups",
							Computed:    true,
						},
						"engine_id": schema.StringAttribute{
							Description: "ID of the engine",
							Computed:    true,
						},
						"id": schema.StringAttribute{
							Description: "ID of the instance",
							Computed:    true,
						},
						"instance_type_id": schema.StringAttribute{
							Description: "ID of the instance type",
							Computed:    true,
						},
						"name": schema.StringAttribute{
							Description: "Name of the instance",
							Computed:    true,
						},
						"status": schema.StringAttribute{
							Description: "Status of the instance",
							Computed:    true,
						},
						"volume_size": schema.Int64Attribute{
							Description: "Size of the volume",
							Computed:    true,
						},
						"volume_type": schema.StringAttribute{
							Description: "Type of the volume",
							Computed:    true,
						},
						"created_at": schema.StringAttribute{
							Description: "Creation timestamp of the instance",
							Computed:    true,
						},
						"parameter_group": schema.StringAttribute{
							Description: "ID of the parameter group to use for the instance.",
							Computed:    true,
						},
						"availability_zone": schema.StringAttribute{
							Description: "Availability zone to use for the instance.",
							Computed:    true,
						},
					},
				},
			},
			"status": schema.StringAttribute{
				Description: "Status of the instances",
				Computed:    true,
				Optional:    true,
				Validators: []validator.String{
					stringvalidator.OneOf("ACTIVE", "BACKING_UP", "CREATING", "DELETED", "DELETING", "ERROR", "ERROR_DELETING", "MAINTENANCE", "PENDING", "REBOOT", "RESIZING", "RESTORING", "STARTING", "STOPPED", "STOPPING"),
				},
			},
		},
	}
}

func (r *DataSourceDbInstances) getAllInstances(ctx context.Context, params dbSDK.ListInstanceOptions) ([]dbSDK.InstanceDetail, error) {
	var allResults []dbSDK.InstanceDetail
	params.Offset = new(int)
	for {
		instances, err := r.dbaasInstances.List(ctx, params)
		if err != nil {
			return nil, err
		}
		allResults = append(allResults, instances...)
		if len(instances) == 0 {
			break
		}
		*params.Offset = *params.Offset + len(instances)
	}
	return allResults, nil
}

func (r *DataSourceDbInstances) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	data := dbInstanceModel{}
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	params := dbSDK.ListInstanceOptions{
		Limit: new(int),
	}
	if data.Status.ValueString() != "" {
		status := dbSDK.InstanceStatus(data.Status.ValueString())
		params.Status = &status
	}

	*params.Limit = 25
	instances, err := r.getAllInstances(ctx, params)
	if err != nil {
		resp.Diagnostics.AddError(utils.ParseSDKError(err))
		return
	}

	var instanceModels []dbInstance
	for _, instance := range instances {
		instanceModels = append(instanceModels, dbInstance{
			Addresses:           convertToAddressModel(instance.Addresses),
			BackupRetentionDays: types.Int64PointerValue(utils.ConvertIntPointerToInt64Pointer(&instance.BackupRetentionDays)),
			CreatedAt:           types.StringValue(instance.CreatedAt),
			EngineID:            types.StringValue(instance.EngineID),
			ID:                  types.StringValue(instance.ID),
			InstanceTypeID:      types.StringValue(instance.InstanceTypeID),
			Name:                types.StringValue(instance.Name),
			Status:              types.StringValue(string(instance.Status)),
			VolumeSize:          types.Int64PointerValue(utils.ConvertIntPointerToInt64Pointer(&instance.Volume.Size)),
			VolumeType:          types.StringValue(string(instance.Volume.Type)),
			AvailabilityZone:    types.StringValue(instance.AvailabilityZone),
			ParameterGroup:      types.StringValue(instance.ParameterGroupID),
		})
	}
	data.Instances = instanceModels
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func convertToStringMapInstance(params []dbSDK.InstanceParametersResponse) map[string]types.String {
	result := make(map[string]types.String, len(params))
	for _, value := range params {
		result[value.Name] = types.StringValue(utils.SdkParamValueToString(value.Value))
	}
	return result
}

func convertToAddressModel(addresses []dbSDK.Address) []InstanceAddress {
	var modelAd []InstanceAddress
	for _, address := range addresses {
		a := InstanceAddress{
			Access:  types.StringValue(string(address.Access)),
			Address: types.StringPointerValue(address.Address),
		}

		if address.Type != nil {
			a.Type = types.StringValue(string(*address.Type))
		}

		modelAd = append(modelAd, a)
	}
	return modelAd
}
