package resources

import (
	"context"

	netSDK "github.com/MagaluCloud/mgc-sdk-go/network"

	"github.com/MagaluCloud/terraform-provider-mgc/mgc/tfutil"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type NetworkSecurityGroupRuleModel struct {
	Id              types.String `tfsdk:"id"`
	Description     types.String `tfsdk:"description"`
	Direction       types.String `tfsdk:"direction"`
	Ethertype       types.String `tfsdk:"ethertype"`
	PortRangeMax    types.Int64  `tfsdk:"port_range_max"`
	PortRangeMin    types.Int64  `tfsdk:"port_range_min"`
	Protocol        types.String `tfsdk:"protocol"`
	RemoteIpPrefix  types.String `tfsdk:"remote_ip_prefix"`
	SecurityGroupId types.String `tfsdk:"security_group_id"`
}

type NetworkSecurityGroupsRulesResource struct {
	networkRules netSDK.RuleService
}

func NewNetworkSecurityGroupsRulesResource() resource.Resource {
	return &NetworkSecurityGroupsRulesResource{}
}

func (r *NetworkSecurityGroupsRulesResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Network Security Group Rule",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The ID of the security group rule",
				Computed:    true,
			},
			"description": schema.StringAttribute{
				Description: "Description of the security group rule. Example: 'Allow incoming SSH traffic'",
				Optional:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"direction": schema.StringAttribute{
				Description: "Direction of traffic flow. Allowed values: 'ingress' or 'egress'. Example: 'ingress'",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.OneOf("ingress", "egress"),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"ethertype": schema.StringAttribute{
				Description: "Network protocol version. Allowed values: 'IPv4' or 'IPv6'. Example: 'IPv4'",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.OneOf("IPv4", "IPv6"),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"port_range_max": schema.Int64Attribute{
				Description: "Maximum port number in the range. Valid values: 1-65535. Example: 22",
				Optional:    true,
				Validators: []validator.Int64{
					int64validator.Between(1, 65535),
				},
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
			"port_range_min": schema.Int64Attribute{
				Description: "Minimum port number in the range. Valid values: 1-65535. Example: 22",
				Optional:    true,
				Validators: []validator.Int64{
					int64validator.Between(1, 65535),
				},
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
			"protocol": schema.StringAttribute{
				Description: "IP protocol. Allowed values: tcp, udp, icmp, icmpv6. Example: 'tcp'",
				Optional:    true,
				Validators: []validator.String{
					stringvalidator.OneOf("tcp", "udp", "icmp", "icmpv6"),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"remote_ip_prefix": schema.StringAttribute{
				Description: "CIDR notation of remote IPv4 and IPv6 range. Example: '192.168.1.0/24', '0.0.0.0/0', '::/0' or '2001:db8::/32'",
				Optional:    true,
				Validators: []validator.String{
					tfutil.CidrValidator{},
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"security_group_id": schema.StringAttribute{
				Description: "ID of the security group to which this rule will be added. Example: 'sg-0123456789abcdef0'",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
		},
	}
}

func (r *NetworkSecurityGroupsRulesResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	dataConfig, ok := req.ProviderData.(tfutil.DataConfig)
	if !ok {
		resp.Diagnostics.AddError("Failed to get provider data", "Failed to get provider data")
		return
	}

	r.networkRules = netSDK.New(&dataConfig.CoreConfig).Rules()
}

func (r *NetworkSecurityGroupsRulesResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_network_security_groups_rules"
}

func (r *NetworkSecurityGroupsRulesResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data NetworkSecurityGroupRuleModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	created, err := r.networkRules.Create(ctx, data.SecurityGroupId.ValueString(), netSDK.RuleCreateRequest{
		Description:    data.Description.ValueStringPointer(),
		Direction:      data.Direction.ValueStringPointer(),
		EtherType:      data.Ethertype.ValueString(),
		PortRangeMax:   tfutil.ConvertInt64PointerToIntPointer(data.PortRangeMax.ValueInt64Pointer()),
		PortRangeMin:   tfutil.ConvertInt64PointerToIntPointer(data.PortRangeMin.ValueInt64Pointer()),
		Protocol:       data.Protocol.ValueStringPointer(),
		RemoteIPPrefix: data.RemoteIpPrefix.ValueStringPointer(),
	})
	if err != nil {
		resp.Diagnostics.AddError(tfutil.ParseSDKError(err))
		return
	}

	data.Id = types.StringValue(created)
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *NetworkSecurityGroupsRulesResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data NetworkSecurityGroupRuleModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	rule, err := r.networkRules.Get(ctx, data.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(tfutil.ParseSDKError(err))
		return
	}

	data.Description = types.StringPointerValue(rule.Description)
	data.Direction = types.StringPointerValue(rule.Direction)
	data.Ethertype = types.StringPointerValue(rule.EtherType)
	data.PortRangeMax = types.Int64PointerValue(tfutil.ConvertIntPointerToInt64Pointer(rule.PortRangeMax))
	data.PortRangeMin = types.Int64PointerValue(tfutil.ConvertIntPointerToInt64Pointer(rule.PortRangeMin))
	data.Protocol = types.StringPointerValue(rule.Protocol)
	data.RemoteIpPrefix = types.StringPointerValue(rule.RemoteIPPrefix)
	data.SecurityGroupId = types.StringPointerValue(rule.SecurityGroupID)

	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *NetworkSecurityGroupsRulesResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data NetworkSecurityGroupRuleModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.networkRules.Delete(ctx, data.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(tfutil.ParseSDKError(err))
		return
	}
}

func (r *NetworkSecurityGroupsRulesResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError("Update is not supported for security group rules", "")
}

func (r *NetworkSecurityGroupsRulesResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
}
