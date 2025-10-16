package lbaas

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	lbSDK "github.com/MagaluCloud/mgc-sdk-go/lbaas"
	"github.com/MagaluCloud/terraform-provider-mgc/mgc/utils"
)

type DataSourceLbaasNetworkBackends struct {
	lbNetworkBackend lbSDK.NetworkBackendService
}

func NewDataSourceLbaasNetworkBackends() datasource.DataSource {
	return &DataSourceLbaasNetworkBackends{}
}

func (r *DataSourceLbaasNetworkBackends) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_lbaas_network_backends"
}

func (r *DataSourceLbaasNetworkBackends) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (r *DataSourceLbaasNetworkBackends) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "List Network Load Balancer Backends. Includes targets for each backend.",
		Attributes: map[string]schema.Attribute{
			"lb_id": schema.StringAttribute{
				Required:    true,
				Description: "The Network Load Balancer ID.",
			},
			"backends": schema.ListNestedAttribute{
				Computed:    true,
				Description: "List of backends.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Computed:    true,
							Description: "Backend ID.",
						},
						"name": schema.StringAttribute{
							Computed:    true,
							Description: "Backend name.",
						},
						"description": schema.StringAttribute{
							Computed:    true,
							Description: "Backend description.",
						},
						"balance_algorithm": schema.StringAttribute{
							Computed:    true,
							Description: "Load balancing algorithm.",
						},
						"health_check_id": schema.StringAttribute{
							Computed:    true,
							Description: "Associated health check ID, if any.",
						},
						"panic_threshold": schema.Float64Attribute{
							Computed:    true,
							Description: "Panic threshold percentage.",
						},
						"close_connections_on_host_health_failure": schema.BoolAttribute{
							Computed:    true,
							Description: "Whether to close connections when a host health check fails.",
						},
						"targets_type": schema.StringAttribute{
							Computed:    true,
							Description: "Type of targets.",
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
				},
			},
		},
	}
}

type backendListState struct {
	LBID     types.String       `tfsdk:"lb_id"`
	Backends []backendItemModel `tfsdk:"backends"`
}

func (r *DataSourceLbaasNetworkBackends) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var backendList backendListState
	resp.Diagnostics.Append(req.Config.Get(ctx, &backendList)...)
	if resp.Diagnostics.HasError() {
		return
	}

	lb, err := r.lbNetworkBackend.ListAll(ctx, backendList.LBID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(utils.ParseSDKError(err))
		return
	}

	for _, b := range lb {
		backendList.Backends = append(backendList.Backends, backendItemModel{}.fromSDKBackend(&b))
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &backendList)...)
}
