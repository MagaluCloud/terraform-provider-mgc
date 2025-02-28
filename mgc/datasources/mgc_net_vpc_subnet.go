package datasources

import (
	"context"

	netSDK "github.com/MagaluCloud/mgc-sdk-go/network"
	tfutil "github.com/MagaluCloud/terraform-provider-mgc/mgc/tfutil"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type mgcNetworkVpcsSubnetModel struct {
	CidrBlock      types.String     `tfsdk:"cidr_block"`
	Description    types.String     `tfsdk:"description"`
	DhcpPools      []DhcpPoolsModel `tfsdk:"dhcp_pools"`
	DnsNameservers []types.String   `tfsdk:"dns_nameservers"`
	GatewayIp      types.String     `tfsdk:"gateway_ip"`
	Id             types.String     `tfsdk:"id"`
	IpVersion      types.String     `tfsdk:"ip_version"`
	Name           types.String     `tfsdk:"name"`
	Updated        types.String     `tfsdk:"updated"`
	VpcId          types.String     `tfsdk:"vpc_id"`
	Zone           types.String     `tfsdk:"zone"`
	SubnetpoolId   types.String     `tfsdk:"subnetpool_id"`
}

type DhcpPoolsModel struct {
	Start types.String `tfsdk:"start"`
	End   types.String `tfsdk:"end"`
}

type mgcNetworkVpcsSubnetDatasource struct {
	networkSubnet netSDK.SubnetService
}

func NewDataSourceNetworkVpcsSubnet() datasource.DataSource {
	return &mgcNetworkVpcsSubnetDatasource{}
}

func (r *mgcNetworkVpcsSubnetDatasource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_network_vpcs_subnet"
}

func (r *mgcNetworkVpcsSubnetDatasource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Network VPC Subnet",
		Attributes: map[string]schema.Attribute{
			"cidr_block": schema.StringAttribute{
				Description: "The CIDR block of the subnet",
				Computed:    true,
			},
			"description": schema.StringAttribute{
				Description: "The description of the subnet",
				Computed:    true,
			},
			"dhcp_pools": schema.ListNestedAttribute{
				Description: "The DHCP pools of the subnet",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"start": schema.StringAttribute{
							Description: "The start of the DHCP pool",
							Computed:    true,
						},
						"end": schema.StringAttribute{
							Description: "The end of the DHCP pool",
							Computed:    true,
						},
					},
				},
				Computed: true,
			},
			"dns_nameservers": schema.ListAttribute{
				Description: "The DNS nameservers of the subnet",
				ElementType: types.StringType,
				Computed:    true,
			},
			"gateway_ip": schema.StringAttribute{
				Description: "The gateway IP of the subnet",
				Computed:    true,
			},
			"id": schema.StringAttribute{
				Description: "The ID of the subnet",
				Required:    true,
			},
			"ip_version": schema.StringAttribute{
				Description: "The IP version of the subnet",
				Computed:    true,
			},
			"name": schema.StringAttribute{
				Description: "The name of the subnet",
				Computed:    true,
			},
			"updated": schema.StringAttribute{
				Description: "The updated timestamp of the subnet",
				Computed:    true,
			},
			"vpc_id": schema.StringAttribute{
				Description: "The VPC ID of the subnet",
				Computed:    true,
			},
			"zone": schema.StringAttribute{
				Description: "The zone of the subnet",
				Computed:    true,
			},
			"subnetpool_id": schema.StringAttribute{
				Description: "The subnet pool ID of the subnet",
				Computed:    true,
			},
		},
	}
}

func (r *mgcNetworkVpcsSubnetDatasource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	dataConfig, ok := req.ProviderData.(tfutil.DataConfig)
	if !ok {
		resp.Diagnostics.AddError("Failed to get provider data", "Failed to get provider data")
		return
	}

	r.networkSubnet = netSDK.New(&dataConfig.CoreConfig).Subnets()
}

func (r *mgcNetworkVpcsSubnetDatasource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	data := &mgcNetworkVpcsSubnetModel{}
	resp.Diagnostics.Append(req.Config.Get(ctx, data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	subnet, err := r.networkSubnet.Get(ctx, data.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(tfutil.ParseSDKError(err))
		return
	}

	data.CidrBlock = types.StringValue(subnet.CIDRBlock)
	data.Description = types.StringPointerValue(subnet.Description)
	data.DhcpPools = make([]DhcpPoolsModel, len(subnet.DHCPPools))
	for _, pool := range subnet.DHCPPools {
		data.DhcpPools = append(data.DhcpPools, DhcpPoolsModel{
			Start: types.StringValue(pool.Start),
			End:   types.StringValue(pool.End),
		})
	}
	var dnsNameservers []types.String
	for _, dns := range subnet.DNSNameservers {
		dnsNameservers = append(dnsNameservers, types.StringValue(dns))
	}
	data.DnsNameservers = dnsNameservers
	data.GatewayIp = types.StringValue(subnet.GatewayIP)
	data.IpVersion = types.StringValue(subnet.IPVersion)
	data.Name = types.StringPointerValue(subnet.Name)
	if subnet.Updated != nil {
		data.Updated = types.StringValue(subnet.Updated.String())
	}
	data.VpcId = types.StringValue(subnet.VPCID)
	data.Zone = types.StringValue(subnet.Zone)
	data.SubnetpoolId = types.StringValue(subnet.SubnetPoolID)

	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}
