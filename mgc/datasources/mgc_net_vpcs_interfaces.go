package datasources

import (
	"context"

	netSDK "github.com/MagaluCloud/mgc-sdk-go/network"

	"github.com/MagaluCloud/terraform-provider-mgc/mgc/tfutil"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type NetworkVPCInterfacesModel struct {
	Items []NetworkVPCInterfaceModel `tfsdk:"items"`
}

type NetworkVPCInterfacesDatasource struct {
	networkInterfaces netSDK.PortService
}

func NewDataSourceNetworkVPCInterfaces() datasource.DataSource {
	return &NetworkVPCInterfacesDatasource{}
}

func (r *NetworkVPCInterfacesDatasource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_network_vpcs_interfaces"
}

func (r *NetworkVPCInterfacesDatasource) Schema(_ context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Network VPC Interfaces",
		Attributes: map[string]schema.Attribute{
			"items": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"created_at": schema.StringAttribute{
							Description: "Creation timestamp of the VPC interface",
							Computed:    true,
						},
						"description": schema.StringAttribute{
							Description: "Description of the VPC interface",
							Computed:    true,
						},
						"id": schema.StringAttribute{
							Description: "The ID of the VPC interface",
							Computed:    true,
						},
						"ip_address": schema.ListNestedAttribute{
							Description: "The IP addresses associated with the VPC interface",
							Computed:    true,
							NestedObject: schema.NestedAttributeObject{
								Attributes: map[string]schema.Attribute{
									"ethertype": schema.StringAttribute{
										Description: "The ethertype of the IP address",
										Computed:    true,
									},
									"ip_address": schema.StringAttribute{
										Description: "The IP address",
										Computed:    true,
									},
									"subnet_id": schema.StringAttribute{
										Description: "The subnet ID",
										Computed:    true,
									},
								},
							},
						},
						"is_admin_state_up": schema.BoolAttribute{
							Description: "Administrative state of the VPC interface",
							Computed:    true,
						},
						"is_port_security_enabled": schema.BoolAttribute{
							Description: "Port security status of the VPC interface",
							Computed:    true,
						},
						"name": schema.StringAttribute{
							Description: "Name of the VPC interface",
							Computed:    true,
						},
						"public_ip": schema.ListNestedAttribute{
							Description: "Public IP configuration of the VPC interface",
							Computed:    true,
							NestedObject: schema.NestedAttributeObject{
								Attributes: map[string]schema.Attribute{
									"public_ip": schema.StringAttribute{
										Description: "The public IP address",
										Computed:    true,
									},
									"public_ip_id": schema.StringAttribute{
										Description: "The public IP ID",
										Computed:    true,
									},
								},
							},
						},
						"security_groups": schema.ListAttribute{
							Description: "List of security group IDs",
							Computed:    true,
							ElementType: types.StringType,
						},
						"updated": schema.StringAttribute{
							Description: "Last update timestamp of the VPC interface",
							Computed:    true,
						},
						"vpc_id": schema.StringAttribute{
							Description: "ID of the VPC this interface belongs to",
							Computed:    true,
						},
						"availability_zone": schema.StringAttribute{
							Description: "Availability zone of the VPC interface",
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

func (r *NetworkVPCInterfacesDatasource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	dataConfig, ok := req.ProviderData.(tfutil.DataConfig)
	if !ok {
		resp.Diagnostics.AddError("Failed to get provider data", "Failed to get provider data")
		return
	}

	r.networkInterfaces = netSDK.New(&dataConfig.CoreConfig).Ports()
}

func (r *NetworkVPCInterfacesDatasource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	data := &NetworkVPCInterfacesModel{}
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	vpcInterfaces, err := r.networkInterfaces.List(ctx)
	if err != nil {
		resp.Diagnostics.AddError(tfutil.ParseSDKError(err))
		return
	}

	for _, vpcInterface := range vpcInterfaces {
		data.Items = append(data.Items, toNetworkVPCInterfaceModel(vpcInterface))
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func toNetworkVPCInterfaceModel(vpcInterface netSDK.PortResponse) NetworkVPCInterfaceModel {
	data := NetworkVPCInterfaceModel{}
	data.Id = types.StringPointerValue(vpcInterface.ID)
	data.Name = types.StringPointerValue(vpcInterface.Name)
	data.VpcId = types.StringPointerValue(vpcInterface.VPCID)
	data.IpAddress = []NetworkVPCInterfaceIpAddressModel{}
	if vpcInterface.IPAddress == nil {
		for _, ipAddress := range *vpcInterface.IPAddress {
			data.IpAddress = append(data.IpAddress, NetworkVPCInterfaceIpAddressModel{
				Ethertype: types.StringPointerValue(ipAddress.Ethertype),
				IpAddress: types.StringValue(ipAddress.IPAddress),
				SubnetId:  types.StringValue(ipAddress.SubnetID),
			})
		}
	}
	data.IsAdminStateUp = types.BoolPointerValue(vpcInterface.IsAdminStateUp)
	data.IsPortSecurityEnabled = types.BoolPointerValue(vpcInterface.IsPortSecurityEnabled)
	data.PublicIp = []NetworkVPCInterfacePublicIpModel{}
	if vpcInterface.PublicIP == nil {
		for _, publicIp := range *vpcInterface.PublicIP {
			data.PublicIp = append(data.PublicIp, NetworkVPCInterfacePublicIpModel{
				PublicIp:   types.StringPointerValue(publicIp.PublicIP),
				PublicIpId: types.StringPointerValue(publicIp.PublicIPID),
			})
		}
	}
	data.SecurityGroups = []types.String{}
	if vpcInterface.SecurityGroups != nil {
		for _, securityGroup := range *vpcInterface.SecurityGroups {
			data.SecurityGroups = append(data.SecurityGroups, types.StringValue(securityGroup))
		}
	}

	if vpcInterface.CreatedAt != nil {
		data.CreatedAt = types.StringValue(vpcInterface.CreatedAt.String())
	}
	if vpcInterface.Updated != nil {
		data.Updated = types.StringValue(vpcInterface.Updated.String())
	}
	data.Description = types.StringPointerValue(vpcInterface.Description)
	data.AvailabilityZone = types.StringPointerValue(vpcInterface.Network.AvailabilityZone)

	return data
}
