package datasources

import (
	"context"

	netSDK "github.com/MagaluCloud/mgc-sdk-go/network"

	"github.com/MagaluCloud/terraform-provider-mgc/mgc/tfutil"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type NetworkPublicIPModel struct {
	Id          types.String `tfsdk:"id"`
	Description types.String `tfsdk:"description"`
	PublicIP    types.String `tfsdk:"public_ip"`
	VPCId       types.String `tfsdk:"vpc_id"`
	PortId      types.String `tfsdk:"port_id"`
}

type NetworkPublicIPDataSource struct {
	networkPIP netSDK.PublicIPService
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
	dataConfig, ok := req.ProviderData.(tfutil.DataConfig)
	if !ok {
		resp.Diagnostics.AddError("Failed to get provider data", "Failed to get provider data")
		return
	}

	r.networkPIP = netSDK.New(&dataConfig.CoreConfig).PublicIPs()
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
	data := &NetworkPublicIPModel{}
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	pip, err := r.networkPIP.Get(ctx, data.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(tfutil.ParseSDKError(err))
		return
	}

	data.Id = types.StringPointerValue(pip.ID)
	data.Description = types.StringPointerValue(pip.Description)
	data.PublicIP = types.StringPointerValue(pip.PublicIP)
	data.VPCId = types.StringPointerValue(pip.VPCID)
	data.PortId = types.StringPointerValue(pip.PortID)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
