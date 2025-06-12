package datasources

import (
	"context"
	"time"

	dbSDK "github.com/MagaluCloud/mgc-sdk-go/dbaas/v2"
	"github.com/MagaluCloud/terraform-provider-mgc/mgc/tfutil"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type DBaaSReplicaListModel struct {
	Replicas []DBaaSReplicaGetModel `tfsdk:"replicas"`
}

type DataSourceDbReplicaList struct {
	service dbSDK.ReplicaService
}

func NewDataSourceDbReplicaList() datasource.DataSource {
	return &DataSourceDbReplicaList{}
}

func (r *DataSourceDbReplicaList) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dbaas_replicas"
}

func (r *DataSourceDbReplicaList) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (r *DataSourceDbReplicaList) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "List all DBaaS replicas",
		Attributes: map[string]schema.Attribute{
			"replicas": schema.ListNestedAttribute{
				Computed:    true,
				Description: "List of replicas",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Computed:    true,
							Description: "Replica ID",
						},
						"source_id": schema.StringAttribute{
							Computed:    true,
							Description: "Source instance ID",
						},
						"name": schema.StringAttribute{
							Computed:    true,
							Description: "Replica name",
						},
						"engine_id": schema.StringAttribute{
							Computed:    true,
							Description: "Engine ID",
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
							Computed: true,
							NestedObject: schema.NestedAttributeObject{
								Attributes: map[string]schema.Attribute{
									"access": schema.StringAttribute{
										Computed:    true,
										Description: "Access level",
									},
									"type": schema.StringAttribute{
										Computed:    true,
										Description: "Address type",
									},
									"address": schema.StringAttribute{
										Computed:    true,
										Description: "Address value",
									},
								},
							},
						},
						"status": schema.StringAttribute{
							Computed:    true,
							Description: "Current status",
						},
						"generation": schema.StringAttribute{
							Computed:    true,
							Description: "Generation identifier",
						},
						"created_at": schema.StringAttribute{
							Computed:    true,
							Description: "Creation timestamp",
						},
						"updated_at": schema.StringAttribute{
							Computed:    true,
							Description: "Last update timestamp",
						},
						"started_at": schema.StringAttribute{
							Computed:    true,
							Description: "Start timestamp",
						},
						"finished_at": schema.StringAttribute{
							Computed:    true,
							Description: "Finish timestamp",
						},
						"maintenance_scheduled_at": schema.StringAttribute{
							Computed:    true,
							Description: "Maintenance scheduled timestamp",
						},
					},
				},
			},
		},
	}
}

func (r *DataSourceDbReplicaList) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data DBaaSReplicaListModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	list, err := r.service.List(ctx, dbSDK.ListReplicaOptions{})
	if err != nil {
		resp.Diagnostics.AddError(tfutil.ParseSDKError(err))
		return
	}

	for _, d := range list {
		var replica DBaaSReplicaGetModel
		replica.ID = types.StringValue(d.ID)
		replica.SourceID = types.StringValue(d.SourceID)
		replica.Name = types.StringValue(d.Name)
		replica.EngineID = types.StringValue(d.EngineID)
		replica.InstanceTypeID = types.StringValue(d.InstanceTypeID)
		replica.VolumeSize = types.Int64Value(int64(d.Volume.Size))
		replica.VolumeType = types.StringValue(d.Volume.Type)
		for _, a := range d.Addresses {
			replica.Addresses = append(replica.Addresses, ReplicaAddressModel{
				Access:  types.StringPointerValue(tfutil.SdkEnumToTFString(&a.Access)),
				Type:    types.StringPointerValue(tfutil.SdkEnumToTFString(a.Type)),
				Address: types.StringPointerValue(a.Address),
			})
		}
		replica.Status = types.StringValue(string(d.Status))
		replica.Generation = types.StringValue(d.Generation)
		replica.CreatedAt = types.StringValue(d.CreatedAt.Format(time.RFC3339))
		if d.UpdatedAt != nil {
			replica.UpdatedAt = types.StringValue(d.UpdatedAt.Format(time.RFC3339))
		}
		replica.StartedAt = types.StringPointerValue(d.StartedAt)
		replica.FinishedAt = types.StringPointerValue(d.FinishedAt)
		replica.MaintenanceScheduledAt = types.StringPointerValue(d.MaintenanceScheduledAt)
		data.Replicas = append(data.Replicas, replica)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
