package network

import (
	"context"

	netSDK "github.com/MagaluCloud/mgc-sdk-go/network"

	"github.com/MagaluCloud/terraform-provider-mgc/mgc/utils"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type NetworkPublicIPDataSourceModel struct {
	Id          types.String `tfsdk:"id"`
	Description types.String `tfsdk:"description"`
	PublicIP    types.String `tfsdk:"public_ip"`
	VPCId       types.String `tfsdk:"vpc_id"`
	PortId      types.String `tfsdk:"port_id"`
}

type NetworkPublicIPDataSource struct {
	networkPIP netSDK.PublicIPService
	networkVpc netSDK.VPCService
}

func NewDataSourceNetworkPublicIP() datasource.DataSource {
	return &NetworkPublicIPDataSource{}
}

func (r *NetworkPublicIPDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_network_public_ip"
}

func (r *NetworkPublicIPDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	dataConfig, ok := req.ProviderData.(utils.DataConfig)
	if !ok {
		resp.Diagnostics.AddError("Failed to get provider data", "Failed to get provider data")
		return
	}

	network := netSDK.New(dataConfig.CoreFor(utils.ServiceNetwork))
	r.networkPIP = network.PublicIPs()
	r.networkVpc = network.VPCs()
}

func (r *NetworkPublicIPDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Network Public IP",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The ID of the public IP",
				Required:    true,
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
	}
}

func (r *NetworkPublicIPDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	data := &NetworkPublicIPDataSourceModel{}
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	publicIPID := data.Id.ValueString()
	pip, err := r.networkPIP.Get(ctx, publicIPID)
	if err != nil {
		resp.Diagnostics.AddError(utils.ParseSDKError(err))
		return
	}

	data.Id = types.StringPointerValue(pip.ID)
	data.Description = types.StringPointerValue(pip.Description)
	data.PublicIP = types.StringPointerValue(pip.PublicIP)
	data.VPCId = types.StringPointerValue(pip.VPCID)
	data.PortId = types.StringPointerValue(pip.PortID)

	if !hasPublicIPVPCID(pip.VPCID) {
		resolved, err := findPublicIPVPCIDs(ctx, r.networkVpc, []string{publicIPID})
		if err != nil {
			resp.Diagnostics.AddError("Failed to resolve public IP VPC ID", err.Error())
			return
		}
		if vpcID, ok := resolved[publicIPID]; ok {
			data.VPCId = types.StringValue(vpcID)
		} else {
			data.VPCId = types.StringNull()
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
