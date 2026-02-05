package network

import (
	"context"

	netSDK "github.com/MagaluCloud/mgc-sdk-go/network"
	"github.com/MagaluCloud/terraform-provider-mgc/mgc/utils"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
)

type NetworkRouteDataSourceModel struct {
	NetworkRouteModel
}

type NetworkRouteDatasource struct {
	networkRoute netSDK.RouteService
}

func NewDataSourceNetworkRoute() datasource.DataSource {
	return &NetworkRouteDatasource{}
}

func (r *NetworkRouteDatasource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_network_route"
}

func (r *NetworkRouteDatasource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	dataConfig, ok := req.ProviderData.(utils.DataConfig)
	if !ok {
		resp.Diagnostics.AddError("Failed to get provider data", "Failed to get provider data")
		return
	}

	r.networkRoute = netSDK.New(&dataConfig.CoreConfig).Routes()
}

func (r *NetworkRouteDatasource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Network Route",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The ID of the route.",
				Required:    true,
			},
			"vpc_id": schema.StringAttribute{
				Description: "ID of the VPC where the route is associated.",
				Required:    true,
			},
			"port_id": schema.StringAttribute{
				Description: "ID of the port used as the next hop for the route.",
				Computed:    true,
			},
			"cidr_destination": schema.StringAttribute{
				Description: "Destination CIDR block that defines the traffic matched by the route.",
				Computed:    true,
			},
			"description": schema.StringAttribute{
				Description: "The description to help identify the route.",
				Computed:    true,
			},
			"next_hop": schema.StringAttribute{
				Description: "Resolved next hop for the route, derived from the associated port.",
				Computed:    true,
			},
			"type": schema.StringAttribute{
				Description: "Type of the route, as defined by the networking service.",
				Computed:    true,
			},
			"status": schema.StringAttribute{
				Description: "Current status of the route.",
				Computed:    true,
			},
		},
	}
}

func (r *NetworkRouteDatasource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	data := &NetworkRouteDataSourceModel{}
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	route, err := r.networkRoute.Get(ctx, data.VpcID.ValueString(), data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(utils.ParseSDKError(err))
		return
	}

	tfResult := convertSDKRouteResultToTerraformNetworkRouteModel(route)
	resp.Diagnostics.Append(resp.State.Set(ctx, &tfResult)...)
}
