package resources

import (
	"context"

	netSDK "github.com/MagaluCloud/mgc-sdk-go/network"

	"github.com/MagaluCloud/terraform-provider-mgc/mgc/tfutil"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type NetworkVPCInterfaceModel struct {
	Id    types.String `tfsdk:"id"`
	VpcId types.String `tfsdk:"vpc_id"`
	Name  types.String `tfsdk:"name"`
}

type NetworkVPCInterfaceResource struct {
	networkVpcsPorts netSDK.VPCService
	networkPorts     netSDK.PortService
	region           string
}

func NewNetworkVPCInterfaceResource() resource.Resource {
	return &NetworkVPCInterfaceResource{}
}

func (r *NetworkVPCInterfaceResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_network_vpcs_interfaces"
}

func (r *NetworkVPCInterfaceResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	dataConfig, ok := req.ProviderData.(tfutil.DataConfig)
	if !ok {
		resp.Diagnostics.AddError("Failed to get provider data", "Failed to get provider data")
		return
	}

	r.networkVpcsPorts = netSDK.New(&dataConfig.CoreConfig).VPCs()
	r.networkPorts = netSDK.New(&dataConfig.CoreConfig).Ports()
	r.region = dataConfig.Region
}

func (r *NetworkVPCInterfaceResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Network VPC Interface",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The ID of the VPC Interface",
				Computed:    true,
			},
			"vpc_id": schema.StringAttribute{
				Description: "The ID of the VPC",
				Required:    true,
			},
			"name": schema.StringAttribute{
				Description: "The name of the VPC Interface",
				Required:    true,
			},
		},
	}
}

func (r *NetworkVPCInterfaceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var model NetworkVPCInterfaceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	vpcInterface, err := r.networkPorts.Get(ctx, model.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(tfutil.ParseSDKError(err))
		return
	}

	model.Name = types.StringPointerValue(vpcInterface.Name)
	model.VpcId = types.StringPointerValue(vpcInterface.VPCID)

	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}

func (r *NetworkVPCInterfaceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var model NetworkVPCInterfaceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	defaultRules := false
	createdVPCInterface, err := r.networkVpcsPorts.CreatePort(ctx, model.VpcId.ValueString(), netSDK.PortCreateRequest{
		Name:   model.Name.ValueString(),
		HasPIP: &defaultRules,
		HasSG:  &defaultRules,
	}, netSDK.PortCreateOptions{})
	if err != nil {
		resp.Diagnostics.AddError(tfutil.ParseSDKError(err))
		return
	}

	model.Id = types.StringValue(createdVPCInterface)
	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}

func (r *NetworkVPCInterfaceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError("Update is not supported for VPC Interfaces", "")
}

func (r *NetworkVPCInterfaceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var model NetworkVPCInterfaceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.networkPorts.Delete(ctx, model.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(tfutil.ParseSDKError(err))
		return
	}
}

func (r *NetworkVPCInterfaceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.Set(ctx, &NetworkVPCInterfaceModel{
		Id: types.StringValue(req.ID),
	})...)
}
