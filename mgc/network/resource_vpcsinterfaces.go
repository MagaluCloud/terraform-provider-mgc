package network

import (
	"context"

	netSDK "github.com/MagaluCloud/mgc-sdk-go/network"

	"github.com/MagaluCloud/terraform-provider-mgc/mgc/utils"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type NetworkVPCInterfaceModel struct {
	Id               types.String   `tfsdk:"id"`
	VpcId            types.String   `tfsdk:"vpc_id"`
	Name             types.String   `tfsdk:"name"`
	AvailabilityZone types.String   `tfsdk:"availability_zone"`
	SubnetsIds       []types.String `tfsdk:"subnet_ids"`
	AntiSpoofing     types.Bool     `tfsdk:"anti_spoofing"`
}

type NetworkVPCInterfaceResource struct {
	networkVpcsPorts netSDK.VPCService
	networkPorts     netSDK.PortService
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
	dataConfig, ok := req.ProviderData.(utils.DataConfig)
	if !ok {
		resp.Diagnostics.AddError("Failed to get provider data", "Failed to get provider data")
		return
	}

	r.networkVpcsPorts = netSDK.New(&dataConfig.CoreConfig).VPCs()
	r.networkPorts = netSDK.New(&dataConfig.CoreConfig).Ports()
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
				PlanModifiers: []planmodifier.String{
					utils.ReplaceIfChangeAndNotIsNotSetOnPlan{},
				},
			},
			"name": schema.StringAttribute{
				Description: "The name of the VPC Interface",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					utils.ReplaceIfChangeAndNotIsNotSetOnPlan{},
				},
			},
			"subnet_ids": schema.ListAttribute{
				Description: "The IDs of the subnets",
				Optional:    true,
				WriteOnly:   true,
				ElementType: types.StringType,
			},
			"availability_zone": schema.StringAttribute{
				Description: "The availability zone of the VPC Interface",
				Computed:    true,
				Optional:    true,
				PlanModifiers: []planmodifier.String{
					utils.ReplaceIfChangeAndNotIsNotSetOnPlan{},
				},
			},
			"anti_spoofing": schema.BoolAttribute{
				Description: "Activates (true) or deactivates (false) the IP Spoofing protection",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
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
		resp.Diagnostics.AddError(utils.ParseSDKError(err))
		return
	}

	model.Name = types.StringPointerValue(vpcInterface.Name)
	model.VpcId = types.StringPointerValue(vpcInterface.VPCID)
	model.AvailabilityZone = types.StringPointerValue(vpcInterface.Network.AvailabilityZone)
	model.AntiSpoofing = types.BoolPointerValue(vpcInterface.IPSpoofingGuard)

	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}

func (r *NetworkVPCInterfaceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var model NetworkVPCInterfaceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	defaultRules := false
	createNic := netSDK.PortCreateRequest{
		Name:   model.Name.ValueString(),
		HasPIP: &defaultRules,
		HasSG:  &defaultRules,
	}

	if model.SubnetsIds != nil {
		var subnets []string
		for _, subnetId := range model.SubnetsIds {
			subnets = append(subnets, subnetId.ValueString())
		}
		createNic.Subnets = &subnets
	}

	createdVPCInterface, err := r.networkVpcsPorts.CreatePort(ctx, model.VpcId.ValueString(), createNic,
		netSDK.PortCreateOptions{
			Zone: model.AvailabilityZone.ValueStringPointer(),
		})
	if err != nil {
		resp.Diagnostics.AddError(utils.ParseSDKError(err))
		return
	}

	err = r.networkPorts.Update(ctx, createdVPCInterface, netSDK.PortUpdateRequest{
		IPSpoofingGuard: model.AntiSpoofing.ValueBoolPointer(),
	})

	if err != nil {
		resp.Diagnostics.AddError(utils.ParseSDKError(err))
		return
	}

	createdVPCGet, err := r.networkPorts.Get(ctx, createdVPCInterface)
	if err != nil {
		resp.Diagnostics.AddError(utils.ParseSDKError(err))
		return
	}

	model.SubnetsIds = nil
	model.AvailabilityZone = types.StringPointerValue(createdVPCGet.Network.AvailabilityZone)
	model.Id = types.StringValue(createdVPCInterface)
	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}

func (r *NetworkVPCInterfaceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var planData NetworkVPCInterfaceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &planData)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var stateData NetworkVPCInterfaceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &stateData)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if planData.AntiSpoofing.ValueBool() != stateData.AntiSpoofing.ValueBool() {
		stateData.AntiSpoofing = planData.AntiSpoofing

		err := r.networkPorts.Update(ctx, stateData.Id.ValueString(), netSDK.PortUpdateRequest{
			IPSpoofingGuard: stateData.AntiSpoofing.ValueBoolPointer(),
		})

		if err != nil {
			resp.Diagnostics.AddError(utils.ParseSDKError(err))
			return
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &stateData)...)
}

func (r *NetworkVPCInterfaceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var model NetworkVPCInterfaceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.networkPorts.Delete(ctx, model.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(utils.ParseSDKError(err))
		return
	}
}

func (r *NetworkVPCInterfaceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.Set(ctx, &NetworkVPCInterfaceModel{
		Id: types.StringValue(req.ID),
	})...)
}
