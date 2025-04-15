package resources

import (
	"context"

	netSDK "github.com/MagaluCloud/mgc-sdk-go/network"

	"github.com/MagaluCloud/terraform-provider-mgc/mgc/tfutil"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type NetworkSubnetPoolModel struct {
	ID          types.String `tfsdk:"id"`
	Cidr        types.String `tfsdk:"cidr"`
	Description types.String `tfsdk:"description"`
	Name        types.String `tfsdk:"name"`
}

type mgcNetworkSubnetpoolsResource struct {
	subnetPoolsService netSDK.SubnetPoolService
}

func NewNetworkSubnetpoolsResource() resource.Resource {
	return &mgcNetworkSubnetpoolsResource{}
}

func (r *mgcNetworkSubnetpoolsResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_network_subnetpools"
}

func (r *mgcNetworkSubnetpoolsResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Network Subnet Pool",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The ID of the subnet pool",
				Computed:    true,
			},
			"cidr": schema.StringAttribute{
				Description: "The CIDR block of the subnet pool",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("172.26.0.0/16"),
				Validators: []validator.String{
					tfutil.CidrValidator{},
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"description": schema.StringAttribute{
				Description: "The description of the subnet pool",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Description: "The name of the subnet pool",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
		},
	}
}

func (r *mgcNetworkSubnetpoolsResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	dataConfig, ok := req.ProviderData.(tfutil.DataConfig)
	if !ok {
		resp.Diagnostics.AddError("Failed to get provider data", "Failed to get provider data")
		return
	}

	r.subnetPoolsService = netSDK.New(&dataConfig.CoreConfig).SubnetPools()
}

func (r *mgcNetworkSubnetpoolsResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	data := NetworkSubnetPoolModel{}
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	subnetPool, err := r.subnetPoolsService.Create(ctx, netSDK.CreateSubnetPoolRequest{
		CIDR:        data.Cidr.ValueStringPointer(),
		Description: data.Description.ValueString(),
		Name:        data.Name.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError(tfutil.ParseSDKError(err))
		return
	}

	data.ID = types.StringValue(subnetPool)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *mgcNetworkSubnetpoolsResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data NetworkSubnetPoolModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	subnetPool, err := r.subnetPoolsService.Get(ctx, data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(tfutil.ParseSDKError(err))
		return
	}

	data.Cidr = types.StringPointerValue(subnetPool.CIDR)
	data.Description = types.StringValue(subnetPool.Description)
	data.Name = types.StringValue(subnetPool.Name)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *mgcNetworkSubnetpoolsResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError("Update is not supported for subnet pools", "")
}

func (r *mgcNetworkSubnetpoolsResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data NetworkSubnetPoolModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.subnetPoolsService.Delete(ctx, data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(tfutil.ParseSDKError(err))
		return
	}
}

func (r *mgcNetworkSubnetpoolsResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
}
