package network

import (
	"context"

	netSDK "github.com/MagaluCloud/mgc-sdk-go/network"
	"github.com/MagaluCloud/terraform-provider-mgc/mgc/utils"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type NetworkVpcsPeeringsDataSourceModel struct {
	VpcID types.String                        `tfsdk:"vpc_id"`
	Items []NetworkVpcsPeeringDataSourceModel `tfsdk:"items"`
}

type NetworkVpcsPeeringsDatasource struct {
	networkPeering netSDK.VpcsPeeringsService
}

func NewDataSourceNetworkVpcsPeerings() datasource.DataSource {
	return &NetworkVpcsPeeringsDatasource{}
}

func (r *NetworkVpcsPeeringsDatasource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_network_vpcs_peerings"
}

func (r *NetworkVpcsPeeringsDatasource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	dataConfig, ok := req.ProviderData.(utils.DataConfig)
	if !ok {
		resp.Diagnostics.AddError("Failed to get provider data", "Failed to get provider data")
		return
	}

	r.networkPeering = netSDK.New(dataConfig.CoreFor(utils.ServiceNetwork)).VpcsPeerings()
}

func (r *NetworkVpcsPeeringsDatasource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Network VPC Peerings",
		Attributes: map[string]schema.Attribute{
			"vpc_id": schema.StringAttribute{
				Description: "When set, only the peerings this VPC takes part in are returned.",
				Optional:    true,
			},
			"items": schema.ListNestedAttribute{
				Description: "List of VPC peerings.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Description: "The ID of the peering.",
							Computed:    true,
						},
						"name": schema.StringAttribute{
							Description: "The name of the peering.",
							Computed:    true,
						},
						"description": schema.StringAttribute{
							Description: "The description to help identify the peering.",
							Computed:    true,
						},
						"status": schema.StringAttribute{
							Description: "Current status of the peering.",
							Computed:    true,
						},
						"requester_vpc_id": schema.StringAttribute{
							Description: "ID of the VPC that requested the peering.",
							Computed:    true,
						},
						"accepter_vpc_id": schema.StringAttribute{
							Description: "ID of the VPC that received the peering invitation.",
							Computed:    true,
						},
						"created_at": schema.StringAttribute{
							Description: "Timestamp of the peering creation.",
							Computed:    true,
						},
						"updated_at": schema.StringAttribute{
							Description: "Timestamp of the last peering update.",
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

func (r *NetworkVpcsPeeringsDatasource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	data := &NetworkVpcsPeeringsDataSourceModel{}
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	peerings, err := r.networkPeering.ListAll(ctx, &netSDK.ListAllVpcsPeeringsOptions{
		VpcID: data.VpcID.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError(utils.ParseSDKError(err))
		return
	}

	for _, peering := range peerings {
		// The API keeps answering deleted peerings for a while; the resource
		// treats those as gone and the data source has to agree.
		if peering.Status == netSDK.VpcsPeeringStatusDeleted {
			continue
		}
		data.Items = append(data.Items, convertSDKVpcsPeeringToDataSourceModel(peering))
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
