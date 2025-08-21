package network

import (
	"context"

	netSDK "github.com/MagaluCloud/mgc-sdk-go/network"

	"github.com/MagaluCloud/terraform-provider-mgc/mgc/utils"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type NetworkPublicIPsModel struct {
	IPs []NetworkPublicIPDataSourceModel `tfsdk:"ips"`
}

type NetworkPublicIPsDataSource struct {
	networkPIP netSDK.PublicIPService
}

func NewDataSourceNetworkPublicIPs() datasource.DataSource {
	return &NetworkPublicIPsDataSource{}
}

func (r *NetworkPublicIPsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_network_public_ips"
}

func (r *NetworkPublicIPsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	dataConfig, ok := req.ProviderData.(utils.DataConfig)
	if !ok {
		resp.Diagnostics.AddError("Failed to get provider data", "Failed to get provider data")
		return
	}

	r.networkPIP = netSDK.New(&dataConfig.CoreConfig).PublicIPs()
}

func (r *NetworkPublicIPsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Network Public IPs",
		Attributes: map[string]schema.Attribute{
			"ips": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Description: "The ID of the public IP",
							Computed:    true,
						},
						"public_ip": schema.StringAttribute{
							Description: "The public IP address",
							Computed:    true,
						},
						"description": schema.StringAttribute{
							Description: "The description of the public IP",
							Computed:    true,
						},
						"vpc_id": schema.StringAttribute{
							Description: "The related VPC ID",
							Computed:    true,
						},
						"port_id": schema.StringAttribute{
							Description: "The port ID it's attached to",
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

func (r *NetworkPublicIPsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	data := &NetworkPublicIPsModel{}
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	pip, err := r.networkPIP.List(ctx)
	if err != nil {
		resp.Diagnostics.AddError(utils.ParseSDKError(err))
		return
	}

	for _, ip := range pip {
		data.IPs = append(data.IPs, NetworkPublicIPDataSourceModel{
			Id:       types.StringPointerValue(ip.ID),
			PublicIP: types.StringPointerValue(ip.PublicIP),
			VPCId:    types.StringPointerValue(ip.VPCID),
			PortId:   types.StringPointerValue(ip.PortID),
		})
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
