package database

import (
	"context"
	"time"

	dbSDK "github.com/MagaluCloud/mgc-sdk-go/dbaas"
	"github.com/MagaluCloud/terraform-provider-mgc/mgc/utils"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type ReplicaAddressModel struct {
	Access  types.String `tfsdk:"access"`
	Type    types.String `tfsdk:"type"`
	Address types.String `tfsdk:"address"`
}

type DBaaSReplicaGetModel struct {
	ID                     types.String          `tfsdk:"id"`
	SourceID               types.String          `tfsdk:"source_id"`
	Name                   types.String          `tfsdk:"name"`
	EngineID               types.String          `tfsdk:"engine_id"`
	InstanceTypeID         types.String          `tfsdk:"instance_type_id"`
	VolumeSize             types.Int64           `tfsdk:"volume_size"`
	VolumeType             types.String          `tfsdk:"volume_type"`
	Addresses              []ReplicaAddressModel `tfsdk:"addresses"`
	Status                 types.String          `tfsdk:"status"`
	Generation             types.String          `tfsdk:"generation"`
	CreatedAt              types.String          `tfsdk:"created_at"`
	UpdatedAt              types.String          `tfsdk:"updated_at"`
	StartedAt              types.String          `tfsdk:"started_at"`
	FinishedAt             types.String          `tfsdk:"finished_at"`
	MaintenanceScheduledAt types.String          `tfsdk:"maintenance_scheduled_at"`
}

type DataSourceDbReplica struct {
	dbaasReplicas dbSDK.ReplicaService
}

func NewDataSourceDbReplica() datasource.DataSource {
	return &DataSourceDbReplica{}
}

func (r *DataSourceDbReplica) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dbaas_replica"
}

func (r *DataSourceDbReplica) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	cfg, ok := req.ProviderData.(utils.DataConfig)
	if !ok {
		resp.Diagnostics.AddError("invalid provider data", "expected utils.DataConfig")
		return
	}
	r.dbaasReplicas = dbSDK.New(&cfg.CoreConfig).Replicas()
}

func (r *DataSourceDbReplica) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Find DBaaS replica by ID",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Required:    true,
				Description: "ID of the replica",
			},
			"source_id": schema.StringAttribute{
				Computed:    true,
				Description: "ID of the source instance",
			},
			"name": schema.StringAttribute{
				Computed:    true,
				Description: "Name of the replica",
			},
			"engine_id": schema.StringAttribute{
				Computed:    true,
				Description: "Engine ID of the replica",
			},
			"instance_type_id": schema.StringAttribute{
				Computed:    true,
				Description: "Instance type ID",
			},
			"volume_size": schema.Int64Attribute{
				Computed:    true,
				Description: "Volume size in GB",
			},
			"volume_type": schema.StringAttribute{
				Computed:    true,
				Description: "Volume type",
			},
			"addresses": schema.ListNestedAttribute{
				Computed:    true,
				Description: "List of replica network addresses",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"access": schema.StringAttribute{
							Computed:    true,
							Description: "Access level",
						},
						"type": schema.StringAttribute{
							Computed:    true,
							Description: "Type of address if available",
						},
						"address": schema.StringAttribute{
							Computed:    true,
							Description: "The actual address",
						},
					},
				},
			},
			"status": schema.StringAttribute{
				Computed:    true,
				Description: "Current replica status",
			},
			"generation": schema.StringAttribute{
				Computed:    true,
				Description: "Generation identifier",
			},
			"created_at": schema.StringAttribute{
				Computed:    true,
				Description: "Creation timestamp in RFC3339 format",
			},
			"updated_at": schema.StringAttribute{
				Computed:    true,
				Description: "Last update timestamp in RFC3339 format",
			},
			"started_at": schema.StringAttribute{
				Computed:    true,
				Description: "Start timestamp in RFC3339 format",
			},
			"finished_at": schema.StringAttribute{
				Computed:    true,
				Description: "Stop timestamp in RFC3339 format",
			},
			"maintenance_scheduled_at": schema.StringAttribute{
				Computed:    true,
				Description: "Next maintenance timestamp in RFC3339 format",
			},
		},
	}
}

func (r *DataSourceDbReplica) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data DBaaSReplicaGetModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	d, err := r.dbaasReplicas.Get(ctx, data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(utils.ParseSDKError(err))
		return
	}
	data.SourceID = types.StringValue(d.SourceID)
	data.Name = types.StringValue(d.Name)
	data.EngineID = types.StringValue(d.EngineID)
	data.InstanceTypeID = types.StringValue(d.InstanceTypeID)
	data.VolumeSize = types.Int64Value(int64(d.Volume.Size))
	data.VolumeType = types.StringValue(d.Volume.Type)
	for _, a := range d.Addresses {
		data.Addresses = append(data.Addresses, ReplicaAddressModel{
			Access:  types.StringPointerValue(utils.SdkEnumToTFString(&a.Access)),
			Type:    types.StringPointerValue(utils.SdkEnumToTFString(a.Type)),
			Address: types.StringPointerValue(a.Address),
		})
	}
	data.Status = types.StringValue(string(d.Status))
	data.Generation = types.StringValue(d.Generation)
	data.CreatedAt = types.StringValue(d.CreatedAt.Format(time.RFC3339))
	if d.UpdatedAt != nil {
		data.UpdatedAt = types.StringValue(d.UpdatedAt.Format(time.RFC3339))
	}
	data.StartedAt = types.StringPointerValue(d.StartedAt)
	data.FinishedAt = types.StringPointerValue(d.FinishedAt)
	data.MaintenanceScheduledAt = types.StringPointerValue(d.MaintenanceScheduledAt)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
