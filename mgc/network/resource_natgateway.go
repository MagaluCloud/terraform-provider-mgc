package network

import (
	"context"

	"github.com/MagaluCloud/mgc-sdk-go/network"
	"github.com/MagaluCloud/terraform-provider-mgc/mgc/utils"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource              = &natGatewayResource{}
	_ resource.ResourceWithConfigure = &natGatewayResource{}
)

func NewNetworkNatGatewayResource() resource.Resource {
	return &natGatewayResource{}
}

type natGatewayResource struct {
	sdkNetwork network.NatGatewayService
	region     string
}

type natGatewayResourceModel struct {
	ID               types.String `tfsdk:"id"`
	Name             types.String `tfsdk:"name"`
	Description      types.String `tfsdk:"description"`
	VPCID            types.String `tfsdk:"vpc_id"`
	AvailabilityZone types.String `tfsdk:"availability_zone"`
}

func (r *natGatewayResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_network_nat_gateway"
}

func (r *natGatewayResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a NAT Gateway resource.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The ID of the NAT Gateway.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Description: "The name of the NAT Gateway.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"description": schema.StringAttribute{
				Description: "The description of the NAT Gateway.",
				Optional:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"vpc_id": schema.StringAttribute{
				Description: "The ID of the VPC where the NAT Gateway will be created.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"availability_zone": schema.StringAttribute{
				Description: "The availability zone of the NAT Gateway.",
				Computed:    true,
				Optional:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
		},
	}
}

func (r *natGatewayResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	dataConfig, ok := req.ProviderData.(utils.DataConfig)
	if !ok {
		resp.Diagnostics.AddError("Failed to get provider data", "Failed to get provider data")
		return
	}

	r.sdkNetwork = network.New(&dataConfig.CoreConfig).NatGateways()
	r.region = dataConfig.Region
}

func (r *natGatewayResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan natGatewayResourceModel
	diags := req.Config.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var azparsed string
	if !plan.AvailabilityZone.IsUnknown() {
		az, err := utils.ConvertAvailabilityZoneToXZone(plan.AvailabilityZone.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Invalid Availability Zone", err.Error())
			return
		}
		azparsed = az
	}

	natGateway, err := r.sdkNetwork.Create(ctx, network.CreateNatGatewayRequest{
		Name:        plan.Name.ValueString(),
		Description: plan.Description.ValueStringPointer(),
		VPCID:       plan.VPCID.ValueString(),
		Zone:        azparsed,
	})
	if err != nil {
		resp.Diagnostics.AddError(utils.ParseSDKError(err))
		return
	}

	plan.ID = types.StringValue(natGateway)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

func (r *natGatewayResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state natGatewayResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	natGateway, err := r.sdkNetwork.Get(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(utils.ParseSDKError(err))
		return
	}

	state.Name = types.StringValue(*natGateway.Name)
	state.Description = types.StringValue(*natGateway.Description)
	state.VPCID = types.StringValue(*natGateway.VPCID)
	if natGateway.Zone != nil {
		state.AvailabilityZone = types.StringValue(utils.ConvertXZoneToAvailabilityZone(r.region, *natGateway.Zone))
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *natGatewayResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError("Update is not supported for NAT Gateway", "")
}

func (r *natGatewayResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state natGatewayResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.sdkNetwork.Delete(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(utils.ParseSDKError(err))
		return
	}
}
