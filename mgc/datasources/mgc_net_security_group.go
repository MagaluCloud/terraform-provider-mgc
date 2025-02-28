package datasources

import (
	"context"

	netSDK "github.com/MagaluCloud/mgc-sdk-go/network"

	"github.com/MagaluCloud/terraform-provider-mgc/mgc/tfutil"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type NetworkSecurityGroupModel struct {
	CreatedAt   types.String                    `tfsdk:"created_at"`
	Description types.String                    `tfsdk:"description"`
	Error       types.String                    `tfsdk:"error"`
	ExternalId  types.String                    `tfsdk:"external_id"`
	Id          types.String                    `tfsdk:"id"`
	IsDefault   types.Bool                      `tfsdk:"is_default"`
	Name        types.String                    `tfsdk:"name"`
	ProjectType types.String                    `tfsdk:"project_type"`
	Rules       []NetworkSecurityGroupRuleModel `tfsdk:"rules"`
	Status      types.String                    `tfsdk:"status"`
	TenantId    types.String                    `tfsdk:"tenant_id"`
	Updated     types.String                    `tfsdk:"updated"`
	VpcId       types.String                    `tfsdk:"vpc_id"`
}

type NetworkSecurityGroupRuleModel struct {
	CreatedAt       types.String `tfsdk:"created_at"`
	Direction       types.String `tfsdk:"direction"`
	Error           types.String `tfsdk:"error"`
	Ethertype       types.String `tfsdk:"ethertype"`
	Id              types.String `tfsdk:"id"`
	PortRangeMax    types.Int64  `tfsdk:"port_range_max"`
	PortRangeMin    types.Int64  `tfsdk:"port_range_min"`
	Protocol        types.String `tfsdk:"protocol"`
	RemoteGroupId   types.String `tfsdk:"remote_group_id"`
	RemoteIpPrefix  types.String `tfsdk:"remote_ip_prefix"`
	SecurityGroupId types.String `tfsdk:"security_group_id"`
	Status          types.String `tfsdk:"status"`
}

type NetworkSecurityGroupResource struct {
	networkSecurityGroups netSDK.SecurityGroupService
}

func NewDataSourceNetworkSecurityGroup() datasource.DataSource {
	return &NetworkSecurityGroupResource{}
}

func (r *NetworkSecurityGroupResource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Network Security Group",
		Attributes: map[string]schema.Attribute{
			"created_at": schema.StringAttribute{
				Description: "The creation timestamp of the security group.",
				Computed:    true,
			},
			"description": schema.StringAttribute{
				Description: "The description of the security group.",
				Optional:    true,
			},
			"error": schema.StringAttribute{
				Description: "Error message, if any.",
				Computed:    true,
			},
			"external_id": schema.StringAttribute{
				Description: "The external ID of the security group.",
				Computed:    true,
			},
			"id": schema.StringAttribute{
				Description: "The ID of the security group.",
				Required:    true,
			},
			"is_default": schema.BoolAttribute{
				Description: "Indicates if this is the default security group.",
				Computed:    true,
			},
			"name": schema.StringAttribute{
				Description: "The name of the security group.",
				Computed:    true,
			},
			"project_type": schema.StringAttribute{
				Description: "The project type of the security group.",
				Computed:    true,
			},
			"rules": schema.ListNestedAttribute{
				Description: "The rules of the security group.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"created_at": schema.StringAttribute{
							Description: "The creation timestamp of the rule.",
							Computed:    true,
						},
						"direction": schema.StringAttribute{
							Description: "The direction of the rule.",
							Computed:    true,
						},
						"error": schema.StringAttribute{
							Description: "Error message, if any.",
							Computed:    true,
						},
						"ethertype": schema.StringAttribute{
							Description: "The ethertype of the rule.",
							Computed:    true,
						},
						"id": schema.StringAttribute{
							Description: "The ID of the rule.",
							Computed:    true,
						},
						"port_range_max": schema.Int64Attribute{
							Description: "The maximum port range of the rule.",
							Computed:    true,
						},
						"port_range_min": schema.Int64Attribute{
							Description: "The minimum port range of the rule.",
							Computed:    true,
						},
						"protocol": schema.StringAttribute{
							Description: "The protocol of the rule.",
							Computed:    true,
						},
						"remote_group_id": schema.StringAttribute{
							Description: "The remote group ID of the rule.",
							Computed:    true,
						},
						"remote_ip_prefix": schema.StringAttribute{
							Description: "The remote IP prefix of the rule.",
							Computed:    true,
						},
						"security_group_id": schema.StringAttribute{
							Description: "The security group ID of the rule.",
							Computed:    true,
						},
						"status": schema.StringAttribute{
							Description: "The status of the rule.",
							Computed:    true,
						},
					},
				},
			},
			"status": schema.StringAttribute{
				Description: "The status of the security group.",
				Computed:    true,
			},
			"tenant_id": schema.StringAttribute{
				Description: "The tenant ID of the security group.",
				Computed:    true,
			},
			"updated": schema.StringAttribute{
				Description: "The last update timestamp of the security group.",
				Computed:    true,
			},
			"vpc_id": schema.StringAttribute{
				Description: "The VPC ID of the security group.",
				Computed:    true,
			},
		},
	}
}

func (r *NetworkSecurityGroupResource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_network_security_group"
}

func (r *NetworkSecurityGroupResource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	dataConfig, ok := req.ProviderData.(tfutil.DataConfig)
	if !ok {
		resp.Diagnostics.AddError("Failed to get provider data", "Failed to get provider data")
		return
	}

	r.networkSecurityGroups = netSDK.New(&dataConfig.CoreConfig).SecurityGroups()
}

func (r *NetworkSecurityGroupResource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data NetworkSecurityGroupModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	securityGroupFound, err := r.networkSecurityGroups.Get(ctx, data.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(tfutil.ParseSDKError(err))
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, securityGroupSdkModelToTerraform(securityGroupFound))...)
}

func securityGroupSdkModelToTerraform(result *netSDK.SecurityGroupDetailResponse) NetworkSecurityGroupModel {
	a := NetworkSecurityGroupModel{
		Rules:       securityGroupRulesSdkModelToTerraform(result.Rules),
		Description: types.StringPointerValue(result.Description),
		Error:       types.StringPointerValue(result.Error),
		ExternalId:  types.StringPointerValue(result.ExternalID),
		Id:          types.StringPointerValue(result.ID),
		IsDefault:   types.BoolPointerValue(result.IsDefault),
		Name:        types.StringPointerValue(result.Name),
		ProjectType: types.StringPointerValue(result.ProjectType),
		Status:      types.StringValue(result.Status),
		TenantId:    types.StringPointerValue(result.TenantID),
		VpcId:       types.StringPointerValue(result.VPCID),
	}
	if result.CreatedAt.String() != "" {
		a.CreatedAt = types.StringValue(result.CreatedAt.String())
	}
	if result.Updated.String() != "" {
		a.Updated = types.StringValue(result.Updated.String())
	}

	return a
}

func securityGroupRulesSdkModelToTerraform(rules *[]netSDK.RuleResponse) []NetworkSecurityGroupRuleModel {
	if rules == nil {
		return []NetworkSecurityGroupRuleModel{}
	}

	var terraformRules []NetworkSecurityGroupRuleModel
	for _, rule := range *rules {
		terraformRules = append(terraformRules, NetworkSecurityGroupRuleModel{
			CreatedAt:       types.StringValue(rule.CreatedAt.String()),
			Direction:       types.StringPointerValue(rule.Direction),
			Error:           types.StringPointerValue(rule.Error),
			Ethertype:       types.StringPointerValue(rule.EtherType),
			Id:              types.StringPointerValue(rule.ID),
			PortRangeMax:    types.Int64PointerValue(tfutil.ConvertIntPointerToInt64Pointer(rule.PortRangeMax)),
			PortRangeMin:    types.Int64PointerValue(tfutil.ConvertIntPointerToInt64Pointer(rule.PortRangeMin)),
			Protocol:        types.StringPointerValue(rule.Protocol),
			RemoteGroupId:   types.StringPointerValue(rule.RemoteGroupID),
			RemoteIpPrefix:  types.StringPointerValue(rule.RemoteIPPrefix),
			SecurityGroupId: types.StringPointerValue(rule.SecurityGroupID),
			Status:          types.StringValue(rule.Status),
		})
	}
	return terraformRules
}
