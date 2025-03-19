package datasources

import (
	"context"

	netSDK "github.com/MagaluCloud/mgc-sdk-go/network"
	"github.com/MagaluCloud/terraform-provider-mgc/mgc/tfutil"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type NetworkVPCsModel struct {
	Items []NetworkVPCModel `tfsdk:"items"`
}

type NetworkVPCsDatasource struct {
	networkVPC netSDK.VPCService
}

func NewDataSourceNetworkVPCs() datasource.DataSource {
	return &NetworkVPCsDatasource{}
}

func (r *NetworkVPCsDatasource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_network_vpcs"
}

func (r *NetworkVPCsDatasource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	dataConfig, ok := req.ProviderData.(tfutil.DataConfig)
	if !ok {
		resp.Diagnostics.AddError("Failed to get provider data", "Failed to get provider data")
		return
	}

	r.networkVPC = netSDK.New(&dataConfig.CoreConfig).VPCs()
}

func (r *NetworkVPCsDatasource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Network VPC",
		Attributes: map[string]schema.Attribute{
			"items": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Description: "The ID of the VPC",
							Computed:    true,
						},
						"name": schema.StringAttribute{
							Description: "The name of the VPC",
							Computed:    true,
						},
						"description": schema.StringAttribute{
							Description: "The description of the VPC",
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

func (r *NetworkVPCsDatasource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	data := &NetworkVPCsModel{}
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	vpcs, err := r.networkVPC.List(ctx)
	if err != nil {
		resp.Diagnostics.AddError(tfutil.ParseSDKError(err))
		return
	}

	for _, vpc := range vpcs {
		data.Items = append(data.Items, NetworkVPCModel{
			Id:          types.StringPointerValue(vpc.ID),
			Name:        types.StringPointerValue(vpc.Name),
			Description: types.StringPointerValue(vpc.Description),
		})
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
