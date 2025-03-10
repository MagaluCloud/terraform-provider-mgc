package resources

import (
	"context"
	"slices"
	"strings"

	netSDK "github.com/MagaluCloud/mgc-sdk-go/network"

	"github.com/MagaluCloud/terraform-provider-mgc/mgc/tfutil"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type NetworkSecurityGroupsAttachModel struct {
	SecurityGroupID types.String `tfsdk:"security_group_id"`
	InterfaceID     types.String `tfsdk:"interface_id"`
}

type NetworkSecurityGroupsAttachResource struct {
	networkPorts netSDK.PortService
}

func NewNetworkSecurityGroupsAttachResource() resource.Resource {
	return &NetworkSecurityGroupsAttachResource{}
}

func (r *NetworkSecurityGroupsAttachResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_network_security_groups_attach"
}

func (r *NetworkSecurityGroupsAttachResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	dataConfig, ok := req.ProviderData.(tfutil.DataConfig)
	if !ok {
		resp.Diagnostics.AddError("Failed to get provider data", "Failed to get provider data")
		return
	}

	r.networkPorts = netSDK.New(&dataConfig.CoreConfig).Ports()
}

func (r *NetworkSecurityGroupsAttachResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Network Security Group Attach",
		Attributes: map[string]schema.Attribute{
			"interface_id": schema.StringAttribute{
				Description: "The ID of the Network Interface",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"security_group_id": schema.StringAttribute{
				Description: "The ID of the Security Group",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
		},
	}
}

func (r *NetworkSecurityGroupsAttachResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	data := NetworkSecurityGroupsAttachModel{}
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	interfaceResponse, err := r.networkPorts.Get(ctx, data.InterfaceID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(tfutil.ParseSDKError(err))
		return
	}

	if interfaceResponse.SecurityGroups == nil {
		resp.Diagnostics.AddError("Interface without security groups", "Security Group is nil")
		return
	}

	if !slices.Contains(*interfaceResponse.SecurityGroups, data.SecurityGroupID.ValueString()) {
		resp.Diagnostics.AddError("Security Group not attach to interface", "Security Group not found in interface")
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *NetworkSecurityGroupsAttachResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data NetworkSecurityGroupsAttachModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.networkPorts.AttachSecurityGroup(ctx, data.InterfaceID.ValueString(), data.SecurityGroupID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(tfutil.ParseSDKError(err))
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *NetworkSecurityGroupsAttachResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError("Update is not supported for Network Security Groups Attach", "")
}

func (r *NetworkSecurityGroupsAttachResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data NetworkSecurityGroupsAttachModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.networkPorts.DetachSecurityGroup(ctx, data.InterfaceID.ValueString(), data.SecurityGroupID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to detach Security Group", err.Error())
		return
	}
}

func (r *NetworkSecurityGroupsAttachResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	input := strings.Split(req.ID, ",")
	if len(input) != 2 {
		resp.Diagnostics.AddError("Invalid ID", "ID must be in the format security_group_id,interface_id")
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &NetworkSecurityGroupsAttachModel{
		SecurityGroupID: types.StringValue(input[0]),
		InterfaceID:     types.StringValue(input[1]),
	})...)
}
