package network

import (
	"context"
	"fmt"
	"time"

	netSDK "github.com/MagaluCloud/mgc-sdk-go/network"
	"github.com/MagaluCloud/terraform-provider-mgc/mgc/utils"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type NetworkVpcsPeeringDataSourceModel struct {
	ID             types.String `tfsdk:"id"`
	Name           types.String `tfsdk:"name"`
	Description    types.String `tfsdk:"description"`
	Status         types.String `tfsdk:"status"`
	RequesterVpcID types.String `tfsdk:"requester_vpc_id"`
	AccepterVpcID  types.String `tfsdk:"accepter_vpc_id"`
	CreatedAt      types.String `tfsdk:"created_at"`
	UpdatedAt      types.String `tfsdk:"updated_at"`
}

type NetworkVpcsPeeringDatasource struct {
	networkPeering netSDK.VpcsPeeringsService
}

func NewDataSourceNetworkVpcsPeering() datasource.DataSource {
	return &NetworkVpcsPeeringDatasource{}
}

func (r *NetworkVpcsPeeringDatasource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_network_vpcs_peering"
}

func (r *NetworkVpcsPeeringDatasource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	dataConfig, ok := req.ProviderData.(utils.DataConfig)
	if !ok {
		resp.Diagnostics.AddError("Failed to get provider data", "Failed to get provider data")
		return
	}

	r.networkPeering = netSDK.New(dataConfig.CoreFor(utils.ServiceNetwork)).VpcsPeerings()
}

func (r *NetworkVpcsPeeringDatasource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Network VPC Peering",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The ID of the peering.",
				Required:    true,
			},
			"name": schema.StringAttribute{
				Description: "The name of the peering.",
				Computed:    true,
			},
			"description": schema.StringAttribute{
				Description: "The description to help identify the peering.",
				Computed:    true,
			},
			"status": schema.StringAttribute{
				Description: "Current status of the peering.",
				Computed:    true,
			},
			"requester_vpc_id": schema.StringAttribute{
				Description: "ID of the VPC that requested the peering.",
				Computed:    true,
			},
			"accepter_vpc_id": schema.StringAttribute{
				Description: "ID of the VPC that received the peering invitation.",
				Computed:    true,
			},
			"created_at": schema.StringAttribute{
				Description: "Timestamp of the peering creation.",
				Computed:    true,
			},
			"updated_at": schema.StringAttribute{
				Description: "Timestamp of the last peering update.",
				Computed:    true,
			},
		},
	}
}

func (r *NetworkVpcsPeeringDatasource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	data := &NetworkVpcsPeeringDataSourceModel{}
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	peeringID := data.ID.ValueString()

	peering, err := r.networkPeering.Get(ctx, peeringID)
	if err != nil {
		if isPeeringNotFound(err) {
			resp.Diagnostics.AddError(
				"VPC peering not found",
				fmt.Sprintf("There is no VPC peering with the ID %s.", peeringID),
			)
			return
		}
		resp.Diagnostics.AddError(utils.ParseSDKError(err))
		return
	}

	tfModel := convertSDKVpcsPeeringToDataSourceModel(*peering)
	resp.Diagnostics.Append(resp.State.Set(ctx, &tfModel)...)
}

// convertSDKVpcsPeeringToDataSourceModel maps a listed peering onto the data
// source model. The API describes the two sides as members with a role; the
// model exposes them as the two VPC ids, mirroring the resource schema.
func convertSDKVpcsPeeringToDataSourceModel(peering netSDK.VpcsPeering) NetworkVpcsPeeringDataSourceModel {
	tfModel := NetworkVpcsPeeringDataSourceModel{
		ID:          types.StringValue(peering.ID),
		Name:        types.StringValue(peering.Name),
		Description: types.StringPointerValue(peering.Description),
		Status:      types.StringValue(string(peering.Status)),
		CreatedAt:   types.StringPointerValue(utils.ConvertTimeToRFC3339((*time.Time)(peering.CreatedAt))),
		UpdatedAt:   types.StringPointerValue(utils.ConvertTimeToRFC3339((*time.Time)(peering.Updated))),
	}

	for _, member := range peering.Members {
		switch member.DirectRole {
		case netSDK.VpcsPeeringDirectRoleRequester:
			tfModel.RequesterVpcID = types.StringValue(member.VpcID)
		case netSDK.VpcsPeeringDirectRoleAccepter:
			tfModel.AccepterVpcID = types.StringValue(member.VpcID)
		}
	}

	return tfModel
}
