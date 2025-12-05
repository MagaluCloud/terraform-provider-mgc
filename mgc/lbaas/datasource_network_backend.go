package lbaas

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"

	lbSDK "github.com/MagaluCloud/mgc-sdk-go/lbaas"
	"github.com/MagaluCloud/terraform-provider-mgc/mgc/utils"
)

type DataSourceLbaasNetworkBackend struct {
	lbNetworkBackend lbSDK.NetworkBackendService
}

func NewDataSourceLbaasNetworkBackend() datasource.DataSource {
	return &DataSourceLbaasNetworkBackend{}
}

func (r *DataSourceLbaasNetworkBackend) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_lbaas_network_backend"
}

func (r *DataSourceLbaasNetworkBackend) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	dataConfig, ok := req.ProviderData.(utils.DataConfig)
	if !ok {
		resp.Diagnostics.AddError("Failed to get provider data", "Failed to get provider data")
		return
	}
	client := lbSDK.New(&dataConfig.CoreConfig)
	r.lbNetworkBackend = client.NetworkBackends()
}

func (r *DataSourceLbaasNetworkBackend) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: experimentalWarning + "Get a specific Network Load Balancer Backend by Load Balancer ID and Backend ID. Includes targets.",
		Attributes: map[string]schema.Attribute{
			"lb_id": schema.StringAttribute{
				Required:    true,
				Description: "The Network Load Balancer ID.",
			},
			"id": schema.StringAttribute{
				Required:    true,
				Description: "The Backend ID.",
			},
			"name": schema.StringAttribute{
				Computed:    true,
				Description: "The name of the backend.",
			},
			"description": schema.StringAttribute{
				Computed:    true,
				Description: "The description of the backend.",
			},
			"balance_algorithm": schema.StringAttribute{
				Computed:    true,
				Description: "The load balancing algorithm.",
			},
			"health_check_id": schema.StringAttribute{
				Computed:    true,
				Description: "The associated health check ID, if any.",
			},
			"panic_threshold": schema.Float64Attribute{
				Computed:    true,
				Description: "The panic threshold percentage for the backend.",
			},
			"close_connections_on_host_health_failure": schema.BoolAttribute{
				Computed:    true,
				Description: "Whether to close connections when a host health check fails.",
			},
			"targets_type": schema.StringAttribute{
				Computed:    true,
				Description: "The type of targets. Example: 'instance' or 'raw'.",
			},
			"targets": schema.ListNestedAttribute{
				Computed:    true,
				Description: "List of backend targets.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"nic_id": schema.StringAttribute{
							Computed:    true,
							Description: "NIC ID of the target (when applicable).",
						},
						"ip_address": schema.StringAttribute{
							Computed:    true,
							Description: "IP address of the target (when applicable).",
						},
						"port": schema.Int64Attribute{
							Computed:    true,
							Description: "Port of the target.",
						},
					},
				},
			},
		},
	}
}

func (r *DataSourceLbaasNetworkBackend) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data networkBackendItemModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	lb, err := r.lbNetworkBackend.Get(ctx, data.LBID.ValueString(), data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(utils.ParseSDKError(err))
		return
	}

	result := networkBackendItemModel{
		LBID:             data.LBID,
		backendItemModel: backendItemModel{}.fromSDKBackend(lb),
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &result)...)
}
