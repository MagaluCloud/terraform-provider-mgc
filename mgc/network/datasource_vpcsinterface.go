package network

import (
	"context"

	netSDK "github.com/MagaluCloud/mgc-sdk-go/network"

	"github.com/MagaluCloud/terraform-provider-mgc/mgc/utils"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type NetworkVPCInterfaceDataSourceModel struct {
	CreatedAt             types.String                        `tfsdk:"created_at"`
	Description           types.String                        `tfsdk:"description"`
	Id                    types.String                        `tfsdk:"id"`
	IpAddress             []NetworkVPCInterfaceIpAddressModel `tfsdk:"ip_address"`
	IsAdminStateUp        types.Bool                          `tfsdk:"is_admin_state_up"`
	IsPortSecurityEnabled types.Bool                          `tfsdk:"is_port_security_enabled"`
	Name                  types.String                        `tfsdk:"name"`
	PublicIp              []NetworkVPCInterfacePublicIpModel  `tfsdk:"public_ip"`
	SecurityGroups        []types.String                      `tfsdk:"security_groups"`
	Updated               types.String                        `tfsdk:"updated"`
	VpcId                 types.String                        `tfsdk:"vpc_id"`
	AvailabilityZone      types.String                        `tfsdk:"availability_zone"`
	AntiSpoofing          types.Bool                          `tfsdk:"anti_spoofing"`
}

type NetworkVPCInterfaceIpAddressModel struct {
	Ethertype types.String `tfsdk:"ethertype"`
	IpAddress types.String `tfsdk:"ip_address"`
	SubnetId  types.String `tfsdk:"subnet_id"`
}

type NetworkVPCInterfacePublicIpModel struct {
	PublicIp   types.String `tfsdk:"public_ip"`
	PublicIpId types.String `tfsdk:"public_ip_id"`
}

type NetworkVPCInterfaceDatasource struct {
	networkInterfaces netSDK.PortService
}

func NewDataSourceNetworkVPCInterface() datasource.DataSource {
	return &NetworkVPCInterfaceDatasource{}
}

func (r *NetworkVPCInterfaceDatasource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_network_vpcs_interface"
}

func (r *NetworkVPCInterfaceDatasource) Schema(_ context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Network VPC Interface",
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
				Required:    true,
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
			"anti_spoofing": schema.BoolAttribute{
				Description: "Indicates whether IP Spoofing protection is enabled",
				Computed:    true,
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
	}
}

func (r *NetworkVPCInterfaceDatasource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	dataConfig, ok := req.ProviderData.(utils.DataConfig)
	if !ok {
		resp.Diagnostics.AddError("Failed to get provider data", "Failed to get provider data")
		return
	}

	r.networkInterfaces = netSDK.New(&dataConfig.CoreConfig).Ports()
}

func (r *NetworkVPCInterfaceDatasource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	data := &NetworkVPCInterfaceDataSourceModel{}
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	vpcInterface, err := r.networkInterfaces.Get(ctx, data.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(utils.ParseSDKError(err))
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, toNetworkVPCInterfaceDataSourceModel(*vpcInterface))...)
}
