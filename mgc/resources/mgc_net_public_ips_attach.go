package resources

import (
	"context"
	"strings"

	netSDK "github.com/MagaluCloud/mgc-sdk-go/network"

	"github.com/MagaluCloud/terraform-provider-mgc/mgc/tfutil"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type NetworkPublicIAttachPModel struct {
	PublicIpID  types.String `tfsdk:"public_ip_id"`
	InterfaceID types.String `tfsdk:"interface_id"`
}

type NetworkPublicIPAttachResource struct {
	networkPIP netSDK.PublicIPService
}

func NewNetworkPublicIPAttachResource() resource.Resource {
	return &NetworkPublicIPAttachResource{}
}

func (r *NetworkPublicIPAttachResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_network_public_ips_attach"
}

func (r *NetworkPublicIPAttachResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *NetworkPublicIPAttachResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Network Public IP Attach",
		Attributes: map[string]schema.Attribute{
			"interface_id": schema.StringAttribute{
				Description: "Interface ID",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"public_ip_id": schema.StringAttribute{
				Description: "Public IP ID",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
		},
	}
}

func (r *NetworkPublicIPAttachResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var model NetworkPublicIAttachPModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.networkPIP.AttachToPort(ctx, model.PublicIpID.ValueString(), model.InterfaceID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(tfutil.ParseSDKError(err))
		return
	}

	resp.State.Set(ctx, &model)
}

func (r *NetworkPublicIPAttachResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var model NetworkPublicIAttachPModel
	resp.Diagnostics.Append(req.State.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.networkPIP.DetachFromPort(ctx, model.PublicIpID.ValueString(), model.InterfaceID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(tfutil.ParseSDKError(err))
		return
	}
}

func (r *NetworkPublicIPAttachResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError("Update not supported", "Update not supported")
}

func (r *NetworkPublicIPAttachResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var model NetworkPublicIAttachPModel
	resp.Diagnostics.Append(req.State.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	pip, err := r.networkPIP.Get(ctx, model.PublicIpID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(tfutil.ParseSDKError(err))
		return
	}

	model.InterfaceID = types.StringPointerValue(pip.PortID)
	resp.State.Set(ctx, &model)
}

func (r *NetworkPublicIPAttachResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.Split(req.ID, ",")
	if len(parts) != 2 {
		resp.Diagnostics.AddError("Invalid ID", "ID must be in the format public_ip_id,interface_id")
		return
	}

	resp.State.Set(ctx, &NetworkPublicIAttachPModel{
		PublicIpID:  types.StringValue(parts[0]),
		InterfaceID: types.StringValue(parts[1]),
	})
}
