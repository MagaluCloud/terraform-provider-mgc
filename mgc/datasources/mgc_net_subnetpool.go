package datasources

import (
	"context"

	netSDK "github.com/MagaluCloud/mgc-sdk-go/network"
	"github.com/MagaluCloud/terraform-provider-mgc/mgc/tfutil"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type mgcNetworkSubnetpoolModel struct {
	Cidr        types.String `tfsdk:"cidr"`
	CreatedAt   types.String `tfsdk:"created_at"`
	Description types.String `tfsdk:"description"`
	Id          types.String `tfsdk:"id"`
	IpVersion   types.Int64  `tfsdk:"ip_version"`
	Name        types.String `tfsdk:"name"`
	TenantId    types.String `tfsdk:"tenant_id"`
	IsDefault   types.Bool   `tfsdk:"is_default"`
}

type mgcNetworkSubnetpoolDatasource struct {
	networkSubnetpools netSDK.SubnetPoolService
}

func NewDataSourceNetworkSubnetpool() datasource.DataSource {
	return &mgcNetworkSubnetpoolDatasource{}
}

func (r *mgcNetworkSubnetpoolDatasource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_network_subnetpool"
}

func (r *mgcNetworkSubnetpoolDatasource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Network Subnet Pool",
		Attributes: map[string]schema.Attribute{
			"cidr": schema.StringAttribute{
				Description: "The CIDR range associated with the subnetpool",
				Computed:    true,
			},
			"created_at": schema.StringAttribute{
				Description: "The timestamp when the subnetpool was created",
				Computed:    true,
			},
			"description": schema.StringAttribute{
				Description: "The description of the subnetpool",
				Computed:    true,
			},
			"id": schema.StringAttribute{
				Description: "The unique identifier of the subnetpool",
				Required:    true,
			},
			"ip_version": schema.Int64Attribute{
				Description: "The IP version of the subnetpool (4 or 6)",
				Computed:    true,
			},
			"name": schema.StringAttribute{
				Description: "The name of the subnetpool",
				Computed:    true,
			},
			"tenant_id": schema.StringAttribute{
				Description: "The ID of the tenant that owns the subnetpool",
				Computed:    true,
			},
			"is_default": schema.BoolAttribute{
				Description: "Whether the subnetpool is the default subnetpool or public IP",
				Computed:    true,
			},
		},
	}
}

func (r *mgcNetworkSubnetpoolDatasource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	data := &mgcNetworkSubnetpoolModel{}
	resp.Diagnostics.Append(req.Config.Get(ctx, data)...)

	subnetPool, err := r.networkSubnetpools.Get(ctx, data.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(tfutil.ParseSDKError(err))
		return
	}

	data.Cidr = types.StringPointerValue(subnetPool.CIDR)
	if subnetPool.CreatedAt.String() != "" {
		data.CreatedAt = types.StringValue(subnetPool.CreatedAt.String())
	}
	data.Description = types.StringValue(subnetPool.Description)
	data.IpVersion = types.Int64PointerValue(tfutil.ConvertIntPointerToInt64Pointer(&subnetPool.IPVersion))
	data.Name = types.StringValue(subnetPool.Name)
	data.TenantId = types.StringValue(subnetPool.TenantID)
	data.IsDefault = types.BoolValue(subnetPool.IsDefault)

	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *mgcNetworkSubnetpoolDatasource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	dataConfig, ok := req.ProviderData.(tfutil.DataConfig)
	if !ok {
		resp.Diagnostics.AddError("Failed to get provider data", "Failed to get provider data")
		return
	}

	r.networkSubnetpools = netSDK.New(&dataConfig.CoreConfig).SubnetPools()
}
