package datasources

import (
	"context"

	netSDK "github.com/MagaluCloud/mgc-sdk-go/network"
	"github.com/MagaluCloud/terraform-provider-mgc/mgc/tfutil"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type NetworkVPCModel struct {
	Id          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
}

type NetworkVPCDatasource struct {
	networkVPC netSDK.VPCService
}

func NewDataSourceNetworkVPC() datasource.DataSource {
	return &NetworkVPCDatasource{}
}

func (r *NetworkVPCDatasource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_network_vpc"
}

func (r *NetworkVPCDatasource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (r *NetworkVPCDatasource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Network VPC",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The ID of the VPC",
				Required:    true,
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
	}
}

func (r *NetworkVPCDatasource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	data := &NetworkVPCModel{}
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	vpc, err := r.networkVPC.Get(ctx, data.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(tfutil.ParseSDKError(err))
		return
	}

	data.Id = types.StringPointerValue(vpc.ID)
	data.Name = types.StringPointerValue(vpc.Name)
	data.Description = types.StringPointerValue(vpc.Description)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
