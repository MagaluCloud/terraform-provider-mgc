package datasources

import (
	"context"

	netSDK "github.com/MagaluCloud/mgc-sdk-go/network"

	"github.com/MagaluCloud/terraform-provider-mgc/mgc/tfutil"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type NetworkSecurityGroupsModel struct {
	Items []NetworkSecurityGroupsModelItem `tfsdk:"items"`
}

type NetworkSecurityGroupsModelItem struct {
	Description types.String `tfsdk:"description"`
	Error       types.String `tfsdk:"error"`
	Id          types.String `tfsdk:"id"`
	IsDefault   types.Bool   `tfsdk:"is_default"`
	Name        types.String `tfsdk:"name"`
	Status      types.String `tfsdk:"status"`
	VpcId       types.String `tfsdk:"vpc_id"`
}

type NetworkSecurityGroupsResource struct {
	networkSecurityGroups netSDK.SecurityGroupService
}

func NewDataSourceNetworkSecurityGroups() datasource.DataSource {
	return &NetworkSecurityGroupsResource{}
}

func (r *NetworkSecurityGroupsResource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Network Security Groups",
		Attributes: map[string]schema.Attribute{
			"items": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"description": schema.StringAttribute{
							Description: "The description of the security group.",
							Optional:    true,
						},
						"error": schema.StringAttribute{
							Description: "Error message, if any.",
							Computed:    true,
						},
						"id": schema.StringAttribute{
							Description: "The ID of the security group.",
							Computed:    true,
						},
						"is_default": schema.BoolAttribute{
							Description: "Indicates if this is the default security group.",
							Computed:    true,
						},
						"name": schema.StringAttribute{
							Description: "The name of the security group.",
							Computed:    true,
						},
						"status": schema.StringAttribute{
							Description: "The status of the security group.",
							Computed:    true,
						},
						"vpc_id": schema.StringAttribute{
							Description: "The VPC ID of the security group.",
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

func (r *NetworkSecurityGroupsResource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_network_security_groups"
}

func (r *NetworkSecurityGroupsResource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (r *NetworkSecurityGroupsResource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data NetworkSecurityGroupsModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	securityGroupFound, err := r.networkSecurityGroups.List(ctx)
	if err != nil {
		resp.Diagnostics.AddError(tfutil.ParseSDKError(err))
		return
	}

	for _, securityGroup := range securityGroupFound {
		data.Items = append(data.Items, NetworkSecurityGroupsModelItem{
			Description: types.StringPointerValue(securityGroup.Description),
			Error:       types.StringPointerValue(securityGroup.Error),
			Id:          types.StringPointerValue(securityGroup.ID),
			IsDefault:   types.BoolPointerValue(securityGroup.IsDefault),
			Name:        types.StringPointerValue(securityGroup.Name),
			Status:      types.StringValue(securityGroup.Status),
			VpcId:       types.StringPointerValue(securityGroup.VPCID),
		})
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
