package datasources

import (
	"context"

	netSDK "github.com/MagaluCloud/mgc-sdk-go/network"
	"github.com/MagaluCloud/terraform-provider-mgc/mgc/tfutil"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type mgcNetworkSubnetpoolsModel struct {
	Items []mgcNetworkSubnetpoolsModelItem `tfsdk:"items"`
}

type mgcNetworkSubnetpoolsModelItem struct {
	Cidr        types.String `tfsdk:"cidr"`
	Description types.String `tfsdk:"description"`
	Id          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	IsDefault   types.Bool   `tfsdk:"is_default"`
}

type mgcNetworkSubnetpoolsDatasource struct {
	networkSubnetpools netSDK.SubnetPoolService
}

func NewDataSourceNetworkSubnetpools() datasource.DataSource {
	return &mgcNetworkSubnetpoolsDatasource{}
}

func (r *mgcNetworkSubnetpoolsDatasource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_network_subnetpools"
}

func (r *mgcNetworkSubnetpoolsDatasource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Network Subnet Pool",
		Attributes: map[string]schema.Attribute{
			"items": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"cidr": schema.StringAttribute{
							Description: "The CIDR range associated with the subnetpool",
							Computed:    true,
						},
						"description": schema.StringAttribute{
							Description: "The description of the subnetpool",
							Computed:    true,
						},
						"id": schema.StringAttribute{
							Description: "The unique identifier of the subnetpool",
							Computed:    true,
						},
						"name": schema.StringAttribute{
							Description: "The name of the subnetpool",
							Computed:    true,
						},
						"is_default": schema.BoolAttribute{
							Description: "Whether the subnetpool is the default subnetpool or public IP",
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

func (r *mgcNetworkSubnetpoolsDatasource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	data := &mgcNetworkSubnetpoolsModel{}
	resp.Diagnostics.Append(req.Config.Get(ctx, data)...)

	subnetPool, err := r.networkSubnetpools.List(ctx, netSDK.ListOptions{})
	if err != nil {
		resp.Diagnostics.AddError(tfutil.ParseSDKError(err))
		return
	}

	for _, item := range subnetPool {
		data.Items = append(data.Items, mgcNetworkSubnetpoolsModelItem{
			Cidr:        types.StringPointerValue(item.CIDR),
			Description: types.StringPointerValue(item.Description),
			Id:          types.StringValue(item.ID),
			Name:        types.StringValue(item.Name),
			IsDefault:   types.BoolValue(item.IsDefault),
		})
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *mgcNetworkSubnetpoolsDatasource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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
