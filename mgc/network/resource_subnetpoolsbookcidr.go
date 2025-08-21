package network

import (
	"context"

	netSDK "github.com/MagaluCloud/mgc-sdk-go/network"

	"github.com/MagaluCloud/terraform-provider-mgc/mgc/utils"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type NetworkSubnetPoolsBookCIDRModel struct {
	SubnetPoolID types.String `tfsdk:"subnet_pool_id"`
	CIDR         types.String `tfsdk:"cidr"`
}

type NetworkSubnetPoolsBookCIDRResource struct {
	subnetPoolsService netSDK.SubnetPoolService
}

func NewNetworkSubnetPoolsBookCIDRResource() resource.Resource {
	return &NetworkSubnetPoolsBookCIDRResource{}
}

func (r *NetworkSubnetPoolsBookCIDRResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_network_subnetpools_book_cidr"
}

func (r *NetworkSubnetPoolsBookCIDRResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	dataConfig, ok := req.ProviderData.(utils.DataConfig)
	if !ok {
		resp.Diagnostics.AddError("Failed to get provider data", "Failed to get provider data")
		return
	}

	r.subnetPoolsService = netSDK.New(&dataConfig.CoreConfig).SubnetPools()
}

func (r *NetworkSubnetPoolsBookCIDRResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Network Subnet Pools Book CIDR",
		Attributes: map[string]schema.Attribute{
			"subnet_pool_id": schema.StringAttribute{
				Description: "Subnet Pool ID",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"cidr": schema.StringAttribute{
				Description: "CIDR",
				Required:    true,
			},
		},
	}
}

func (r *NetworkSubnetPoolsBookCIDRResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	data := NetworkSubnetPoolsBookCIDRModel{}
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.State.Set(ctx, &data)
}

func (r *NetworkSubnetPoolsBookCIDRResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data NetworkSubnetPoolsBookCIDRModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.subnetPoolsService.BookCIDR(ctx, data.SubnetPoolID.ValueString(), netSDK.BookCIDRRequest{
		CIDR: data.CIDR.ValueStringPointer(),
	})
	if err != nil {
		resp.Diagnostics.AddError(utils.ParseSDKError(err))
		return
	}

	resp.State.Set(ctx, &data)
}

func (r *NetworkSubnetPoolsBookCIDRResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data NetworkSubnetPoolsBookCIDRModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.subnetPoolsService.UnbookCIDR(ctx, data.SubnetPoolID.ValueString(), netSDK.UnbookCIDRRequest{
		CIDR: data.CIDR.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError(utils.ParseSDKError(err))
		return
	}
}

func (r *NetworkSubnetPoolsBookCIDRResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError("Update is not supported for network subnet pools book CIDR", "")
}
