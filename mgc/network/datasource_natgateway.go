package network

import (
	"context"

	"github.com/MagaluCloud/mgc-sdk-go/network"
	"github.com/MagaluCloud/terraform-provider-mgc/mgc/utils"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource              = &natGatewayDataSource{}
	_ datasource.DataSourceWithConfigure = &natGatewayDataSource{}
)

func NewDataSourceNetworkNatGateway() datasource.DataSource {
	return &natGatewayDataSource{}
}

type natGatewayDataSource struct {
	sdkNetwork network.NatGatewayService
}

type natGatewayDataSourceModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	VPCID       types.String `tfsdk:"vpc_id"`
	Zone        types.String `tfsdk:"zone"`
}

func (d *natGatewayDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_network_nat_gateway"
}

func (d *natGatewayDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Get information about a NAT Gateway.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The ID of the NAT Gateway.",
				Required:    true,
			},
			"name": schema.StringAttribute{
				Description: "The name of the NAT Gateway.",
				Computed:    true,
			},
			"description": schema.StringAttribute{
				Description: "The description of the NAT Gateway.",
				Computed:    true,
			},
			"vpc_id": schema.StringAttribute{
				Description: "The ID of the VPC where the NAT Gateway is located.",
				Computed:    true,
			},
			"zone": schema.StringAttribute{
				Description: "The zone of the NAT Gateway.",
				Computed:    true,
			},
		},
	}
}

func (d *natGatewayDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	dataConfig, ok := req.ProviderData.(utils.DataConfig)
	if !ok {
		resp.Diagnostics.AddError("Failed to get provider data", "Failed to get provider data")
		return
	}

	d.sdkNetwork = network.New(&dataConfig.CoreConfig).NatGateways()
}

func (d *natGatewayDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state natGatewayDataSourceModel

	diags := req.Config.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	natGateway, err := d.sdkNetwork.Get(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(utils.ParseSDKError(err))
		return
	}

	state.Name = types.StringValue(*natGateway.Name)
	state.Description = types.StringValue(*natGateway.Description)
	state.VPCID = types.StringValue(*natGateway.VPCID)
	state.Zone = types.StringValue(*natGateway.Zone)

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}
