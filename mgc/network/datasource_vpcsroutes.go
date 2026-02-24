package network

import (
	"context"

	netSDK "github.com/MagaluCloud/mgc-sdk-go/network"
	"github.com/MagaluCloud/terraform-provider-mgc/mgc/utils"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type NetworkListVpcsRouteModel struct {
	ID              types.String `tfsdk:"id"`
	PortID          types.String `tfsdk:"port_id"`
	CIDRDestination types.String `tfsdk:"cidr_destination"`
	Description     types.String `tfsdk:"description"`
	NextHop         types.String `tfsdk:"next_hop"`
	Type            types.String `tfsdk:"type"`
	Status          types.String `tfsdk:"status"`
}

type NetworkVpcsRoutesDataSourceModel struct {
	VpcID  types.String                `tfsdk:"vpc_id"`
	Routes []NetworkListVpcsRouteModel `tfsdk:"routes"`
}

type NetworkVpcsRoutesDatasource struct {
	networkRoute netSDK.VpcsRoutesService
}

func NewDataSourceNetworkVpcsRoutes() datasource.DataSource {
	return &NetworkVpcsRoutesDatasource{}
}

func (r *NetworkVpcsRoutesDatasource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_network_vpcs_routes"
}

func (r *NetworkVpcsRoutesDatasource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	dataConfig, ok := req.ProviderData.(utils.DataConfig)
	if !ok {
		resp.Diagnostics.AddError("Failed to get provider data", "Failed to get provider data")
		return
	}

	r.networkRoute = netSDK.New(&dataConfig.CoreConfig).VpcsRoutes()
}

func (r *NetworkVpcsRoutesDatasource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Network Routes",
		Attributes: map[string]schema.Attribute{
			"vpc_id": schema.StringAttribute{
				Description: "ID of the VPC where the routes is associated.",
				Required:    true,
			},
			"routes": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Description: "The ID of the route.",
							Computed:    true,
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
				},
			},
		},
	}
}

func (r *NetworkVpcsRoutesDatasource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	data := &NetworkVpcsRoutesDataSourceModel{}
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	routes, err := r.networkRoute.ListAll(ctx, data.VpcID.ValueString(), &netSDK.ListAllVpcsRoutesOptions{})
	if err != nil {
		resp.Diagnostics.AddError(utils.ParseSDKError(err))
		return
	}

	for _, route := range routes {
		data.Routes = append(data.Routes, *convertSDKListRouteResultToTerraformNetworkListVpcsRouteModel(&route))
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func convertSDKListRouteResultToTerraformNetworkListVpcsRouteModel(sdkResult *netSDK.VpcsRouteDetail) *NetworkListVpcsRouteModel {
	if sdkResult == nil {
		return nil
	}

	tfModel := &NetworkListVpcsRouteModel{
		ID:              types.StringValue(sdkResult.ID),
		PortID:          types.StringValue(sdkResult.PortID),
		CIDRDestination: types.StringValue(sdkResult.CIDRDestination),
		NextHop:         types.StringValue(sdkResult.NextHop),
		Type:            types.StringValue(sdkResult.Type),
		Status:          types.StringValue(string(sdkResult.Status)),
	}

	var description *string
	if sdkResult.Description != "" {
		description = &sdkResult.Description
	}
	tfModel.Description = types.StringPointerValue(description)

	return tfModel
}
