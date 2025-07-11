package datasources

import (
	"context"

	dbSDK "github.com/MagaluCloud/mgc-sdk-go/dbaas"

	"github.com/MagaluCloud/terraform-provider-mgc/mgc/tfutil"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &DataSourceDbInstance{}

type DataSourceDbInstance struct {
	dbaasInstances dbSDK.InstanceService
}

type dbInstanceDataSourceModel struct {
	ID                  types.String      `tfsdk:"id"`
	Addresses           []InstanceAddress `tfsdk:"addresses"`
	BackupRetentionDays types.Int64       `tfsdk:"backup_retention_days"`
	CreatedAt           types.String      `tfsdk:"created_at"`
	EngineID            types.String      `tfsdk:"engine_id"`
	InstanceTypeID      types.String      `tfsdk:"instance_type_id"`
	Name                types.String      `tfsdk:"name"`
	Status              types.String      `tfsdk:"status"`
	VolumeSize          types.Int64       `tfsdk:"volume_size"`
	VolumeType          types.String      `tfsdk:"volume_type"`
	AvailabilityZone    types.String      `tfsdk:"availability_zone"`
	ParameterGroup      types.String      `tfsdk:"parameter_group"`
}

func NewDataSourceDbaasInstance() datasource.DataSource {
	return &DataSourceDbInstance{}
}

func (r *DataSourceDbInstance) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dbaas_instance"
}

func (r *DataSourceDbInstance) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	dataConfig, ok := req.ProviderData.(tfutil.DataConfig)
	if !ok {
		resp.Diagnostics.AddError("Failed to get provider data", "Failed to get provider data")
		return
	}

	r.dbaasInstances = dbSDK.New(&dataConfig.CoreConfig).Instances()
}

func (r *DataSourceDbInstance) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Get a database instance by ID.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "ID of the instance to fetch",
				Required:    true,
			},
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
			"created_at": schema.StringAttribute{
				Description: "Creation timestamp of the instance",
				Computed:    true,
			},
			"engine_id": schema.StringAttribute{
				Description: "ID of the engine",
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
			"parameter_group": schema.StringAttribute{
				Description: "ID of the parameter group to use for the instance.",
				Computed:    true,
			},
			"availability_zone": schema.StringAttribute{
				Description: "Availability zone to use for the instance.",
				Computed:    true,
			},
		},
	}
}

func (r *DataSourceDbInstance) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data dbInstanceDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	instance, err := r.dbaasInstances.Get(ctx, data.ID.ValueString(), dbSDK.GetInstanceOptions{})
	if err != nil {
		resp.Diagnostics.AddError(tfutil.ParseSDKError(err))
		return
	}

	data.Addresses = convertToAddressModel(instance.Addresses)
	data.BackupRetentionDays = types.Int64PointerValue(tfutil.ConvertIntPointerToInt64Pointer(&instance.BackupRetentionDays))
	data.CreatedAt = types.StringValue(instance.CreatedAt)
	data.EngineID = types.StringValue(instance.EngineID)
	data.InstanceTypeID = types.StringValue(instance.InstanceTypeID)
	data.Name = types.StringValue(instance.Name)
	data.Status = types.StringValue(string(instance.Status))
	data.VolumeSize = types.Int64PointerValue(tfutil.ConvertIntPointerToInt64Pointer(&instance.Volume.Size))
	data.VolumeType = types.StringValue(string(instance.Volume.Type))
	data.AvailabilityZone = types.StringValue(instance.AvailabilityZone)
	data.ParameterGroup = types.StringValue(instance.ParameterGroupID)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
