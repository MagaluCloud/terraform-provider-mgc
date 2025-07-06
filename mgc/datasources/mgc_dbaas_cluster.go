package datasources

import (
	"context"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	dbSDK "github.com/MagaluCloud/mgc-sdk-go/dbaas"
	"github.com/MagaluCloud/terraform-provider-mgc/mgc/tfutil"
)

type dbaasClusterDataModel struct {
	ID                     types.String                   `tfsdk:"id"`
	Name                   types.String                   `tfsdk:"name"`
	EngineID               types.String                   `tfsdk:"engine_id"`
	InstanceTypeID         types.String                   `tfsdk:"instance_type_id"`
	VolumeSize             types.Int64                    `tfsdk:"volume_size"`
	VolumeType             types.String                   `tfsdk:"volume_type"`
	ParameterGroupID       types.String                   `tfsdk:"parameter_group_id"`
	BackupRetentionDays    types.Int64                    `tfsdk:"backup_retention_days"`
	BackupStartAt          types.String                   `tfsdk:"backup_start_at"`
	Status                 types.String                   `tfsdk:"status"`
	Addresses              []dbaasClusterAddressDataModel `tfsdk:"addresses"`
	ApplyParametersPending types.Bool                     `tfsdk:"apply_parameters_pending"`
	CreatedAt              types.String                   `tfsdk:"created_at"`
	UpdatedAt              types.String                   `tfsdk:"updated_at"`
	StartedAt              types.String                   `tfsdk:"started_at"`
	FinishedAt             types.String                   `tfsdk:"finished_at"`
}

type dbaasClusterAddressDataModel struct {
	Access  types.String `tfsdk:"access"`
	Type    types.String `tfsdk:"type"`
	Address types.String `tfsdk:"address"`
	Port    types.String `tfsdk:"port"`
}

func convertSDKClusterToDataModel(detail dbSDK.ClusterDetailResponse) dbaasClusterDataModel {
	var addresses []dbaasClusterAddressDataModel
	for _, addr := range detail.Addresses {
		addresses = append(addresses, dbaasClusterAddressDataModel{
			Access:  types.StringValue(string(addr.Access)),
			Type:    types.StringValue(string(addr.Type)),
			Address: types.StringValue(addr.Address),
			Port:    types.StringValue(addr.Port),
		})
	}

	return dbaasClusterDataModel{
		ID:                     types.StringValue(detail.ID),
		Name:                   types.StringValue(detail.Name),
		EngineID:               types.StringValue(detail.EngineID),
		InstanceTypeID:         types.StringValue(detail.InstanceTypeID),
		VolumeSize:             types.Int64Value(int64(detail.Volume.Size)),
		VolumeType:             types.StringValue(detail.Volume.Type),
		ParameterGroupID:       types.StringValue(detail.ParameterGroupID),
		BackupRetentionDays:    types.Int64Value(int64(detail.BackupRetentionDays)),
		BackupStartAt:          types.StringValue(detail.BackupStartAt),
		Status:                 types.StringValue(string(detail.Status)),
		Addresses:              addresses,
		ApplyParametersPending: types.BoolValue(detail.ApplyParametersPending),
		CreatedAt:              types.StringValue(detail.CreatedAt.Format(time.RFC3339)),
		UpdatedAt:              types.StringPointerValue(tfutil.ConvertTimeToRFC3339(detail.UpdatedAt)),
		StartedAt:              types.StringPointerValue(detail.StartedAt),
		FinishedAt:             types.StringPointerValue(detail.FinishedAt),
	}
}

func dbaasClusterAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"id": schema.StringAttribute{
			Description: "ID of the cluster to fetch",
			Required:    true,
		},
		"name": schema.StringAttribute{
			Description: "Name of the DBaaS cluster.",
			Computed:    true,
		},
		"engine_id": schema.StringAttribute{
			Description: "ID of the database engine used by the cluster.",
			Computed:    true,
		},
		"instance_type_id": schema.StringAttribute{
			Description: "ID of the instance type for the cluster nodes.",
			Computed:    true,
		},
		"volume_size": schema.Int64Attribute{
			Description: "Size of the storage volume in GB.",
			Computed:    true,
		},
		"volume_type": schema.StringAttribute{
			Description: "Type of the storage volume.",
			Computed:    true,
		},
		"parameter_group_id": schema.StringAttribute{
			Description: "ID of the parameter group associated with the cluster.",
			Computed:    true,
		},
		"backup_retention_days": schema.Int64Attribute{
			Description: "Number of days to retain automated backups.",
			Computed:    true,
		},
		"backup_start_at": schema.StringAttribute{
			Description: "Time to initiate the daily backup in UTC (format: 'HH:MM:SS').",
			Computed:    true,
		},
		"status": schema.StringAttribute{
			Description: "Current status of the DBaaS cluster.",
			Computed:    true,
		},
		"addresses": schema.ListNestedAttribute{
			Description: "Network addresses for connecting to the cluster.",
			Computed:    true,
			NestedObject: schema.NestedAttributeObject{
				Attributes: map[string]schema.Attribute{
					"access": schema.StringAttribute{
						Description: "Access type (e.g., 'public', 'private').",
						Computed:    true,
					},
					"type": schema.StringAttribute{
						Description: "Address type (e.g., 'read-write', 'read-only').",
						Computed:    true,
					},
					"address": schema.StringAttribute{
						Description: "The IP address or hostname.",
						Computed:    true,
					},
					"port": schema.StringAttribute{
						Description: "The port number.",
						Computed:    true,
					},
				},
			},
		},
		"apply_parameters_pending": schema.BoolAttribute{
			Description: "Indicates if parameter changes are pending application.",
			Computed:    true,
		},
		"created_at": schema.StringAttribute{
			Description: "Timestamp of when the cluster was created.",
			Computed:    true,
		},
		"updated_at": schema.StringAttribute{
			Description: "Timestamp of when the cluster was last updated.",
			Computed:    true,
		},
		"started_at": schema.StringAttribute{
			Description: "Timestamp of when the cluster was last started.",
			Computed:    true,
		},
		"finished_at": schema.StringAttribute{
			Description: "Timestamp of when the cluster last finished an operation.",
			Computed:    true,
		},
	}
}

type DBaaSClusterDataSource struct {
	clusterService dbSDK.ClusterService
}

type dbaasClusterDataSourceModel struct {
	ID                     types.String                   `tfsdk:"id"`
	Name                   types.String                   `tfsdk:"name"`
	EngineID               types.String                   `tfsdk:"engine_id"`
	InstanceTypeID         types.String                   `tfsdk:"instance_type_id"`
	VolumeSize             types.Int64                    `tfsdk:"volume_size"`
	VolumeType             types.String                   `tfsdk:"volume_type"`
	ParameterGroupID       types.String                   `tfsdk:"parameter_group_id"`
	BackupRetentionDays    types.Int64                    `tfsdk:"backup_retention_days"`
	BackupStartAt          types.String                   `tfsdk:"backup_start_at"`
	Status                 types.String                   `tfsdk:"status"`
	Addresses              []dbaasClusterAddressDataModel `tfsdk:"addresses"`
	ApplyParametersPending types.Bool                     `tfsdk:"apply_parameters_pending"`
	CreatedAt              types.String                   `tfsdk:"created_at"`
	UpdatedAt              types.String                   `tfsdk:"updated_at"`
	StartedAt              types.String                   `tfsdk:"started_at"`
	FinishedAt             types.String                   `tfsdk:"finished_at"`
}

func NewDBaaSClusterDataSource() datasource.DataSource {
	return &DBaaSClusterDataSource{}
}

func (ds *DBaaSClusterDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dbaas_cluster"
}

func (ds *DBaaSClusterDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	dataConfig, ok := req.ProviderData.(tfutil.DataConfig)
	if !ok {
		resp.Diagnostics.AddError("Failed to get provider data", "Provider data has unexpected type")
		return
	}
	ds.clusterService = dbSDK.New(&dataConfig.CoreConfig).Clusters()
}

func (ds *DBaaSClusterDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	attrs := dbaasClusterAttributes()
	attrs["id"] = schema.StringAttribute{
		Description: "The ID of the DBaaS cluster to retrieve.",
		Required:    true,
	}
	resp.Schema = schema.Schema{
		Description: "Retrieves information about a specific DBaaS cluster.",
		Attributes:  attrs,
	}
}

func (ds *DBaaSClusterDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config dbaasClusterDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	sdkCluster, err := ds.clusterService.Get(ctx, config.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(tfutil.ParseSDKError(err))
		return
	}

	state := convertSDKClusterToDataModel(*sdkCluster)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
