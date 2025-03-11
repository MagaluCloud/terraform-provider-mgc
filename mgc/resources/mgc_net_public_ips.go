package resources

import (
	"context"

	netSDK "github.com/MagaluCloud/mgc-sdk-go/network"

	"github.com/MagaluCloud/terraform-provider-mgc/mgc/tfutil"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type NetworkPublicIPModel struct {
	Id          types.String `tfsdk:"id"`
	PublicIP    types.String `tfsdk:"public_ip"`
	Description types.String `tfsdk:"description"`
	VPCId       types.String `tfsdk:"vpc_id"`
}

type NetworkPublicIPResource struct {
	networkPIP netSDK.PublicIPService
	networkVpc netSDK.VPCService
}

func NewNetworkPublicIPResource() resource.Resource {
	return &NetworkPublicIPResource{}
}

func (r *NetworkPublicIPResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_network_public_ips"
}

func (r *NetworkPublicIPResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	dataConfig, ok := req.ProviderData.(tfutil.DataConfig)
	if !ok {
		resp.Diagnostics.AddError("Failed to get provider data", "Failed to get provider data")
		return
	}

	r.networkPIP = netSDK.New(&dataConfig.CoreConfig).PublicIPs()
	r.networkVpc = netSDK.New(&dataConfig.CoreConfig).VPCs()
}

func (r *NetworkPublicIPResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Network Public IP",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The ID of the public IP",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"public_ip": schema.StringAttribute{
				Description: "The public IP address",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"description": schema.StringAttribute{
				Description: "The description of the public IP",
				Optional:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"vpc_id": schema.StringAttribute{
				Description: "The related VPC ID",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
		},
	}
}

func (r *NetworkPublicIPResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data NetworkPublicIPModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	createdPIP, err := r.networkVpc.CreatePublicIP(ctx, data.VPCId.ValueString(), netSDK.PublicIPCreateRequest{
		Description: data.Description.ValueStringPointer(),
	})
	if err != nil {
		resp.Diagnostics.AddError(tfutil.ParseSDKError(err))
		return
	}

	pip, err := r.networkPIP.Get(ctx, createdPIP)
	if err != nil {
		resp.Diagnostics.AddError("Failed to fetch public IP address", err.Error())
	}

	data.Id = types.StringPointerValue(pip.ID)
	data.PublicIP = types.StringPointerValue(pip.PublicIP)
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *NetworkPublicIPResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data NetworkPublicIPModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	pip, err := r.networkPIP.Get(ctx, data.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(tfutil.ParseSDKError(err))
		return
	}

	data.Id = types.StringPointerValue(pip.ID)
	data.PublicIP = types.StringPointerValue(pip.PublicIP)
	data.Description = types.StringPointerValue(pip.Description)

	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *NetworkPublicIPResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data NetworkPublicIPModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.networkPIP.Delete(ctx, data.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(tfutil.ParseSDKError(err))
		return
	}
}

func (r *NetworkPublicIPResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError("Update is not supported for public IP", "")
}

func (r *NetworkPublicIPResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
}
