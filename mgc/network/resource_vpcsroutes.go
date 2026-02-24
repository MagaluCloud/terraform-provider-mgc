package network

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"time"

	clientSDK "github.com/MagaluCloud/mgc-sdk-go/client"
	netSDK "github.com/MagaluCloud/mgc-sdk-go/network"

	"github.com/MagaluCloud/terraform-provider-mgc/mgc/utils"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

const (
	RoutePoolingTimeout = 100 * time.Minute
)

type NetworkVpcsRouteModel struct {
	ID              types.String `tfsdk:"id"`
	VpcID           types.String `tfsdk:"vpc_id"`
	PortID          types.String `tfsdk:"port_id"`
	CIDRDestination types.String `tfsdk:"cidr_destination"`
	Description     types.String `tfsdk:"description"`
	NextHop         types.String `tfsdk:"next_hop"`
	Type            types.String `tfsdk:"type"`
	Status          types.String `tfsdk:"status"`
}

type NetworkVpcsRouteResource struct {
	networkRoute netSDK.VpcsRoutesService
}

func NewNetworkVpcsRouteResource() resource.Resource {
	return &NetworkVpcsRouteResource{}
}

func (r *NetworkVpcsRouteResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_network_vpcs_route"
}

func (r *NetworkVpcsRouteResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	dataConfig, ok := req.ProviderData.(utils.DataConfig)
	if !ok {
		resp.Diagnostics.AddError("Failed to get provider data", "Failed to get provider data")
		return
	}

	r.networkRoute = netSDK.New(&dataConfig.CoreConfig).VpcsRoutes()
}

func (r *NetworkVpcsRouteResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Network Route",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The ID of the route.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"vpc_id": schema.StringAttribute{
				Description: "ID of the VPC where this route is associated.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"port_id": schema.StringAttribute{
				Description: "ID of the port used as the next hop for this route.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"cidr_destination": schema.StringAttribute{
				Description: "Destination CIDR block that defines the traffic matched by this route.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"description": schema.StringAttribute{
				Description: "The description to help identify the route.",
				Optional:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"next_hop": schema.StringAttribute{
				Description: "Resolved next hop for the route, derived from the associated port.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"type": schema.StringAttribute{
				Description: "Type of the route, as defined by the networking service.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"status": schema.StringAttribute{
				Description: "Current status of the route.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
		},
	}
}

func (r *NetworkVpcsRouteResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data NetworkVpcsRouteModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	vpcID := data.VpcID.ValueString()

	createdRoute, err := r.networkRoute.Create(ctx, vpcID, netSDK.VpcsRoutesCreateRequest{
		PortID:          data.PortID.ValueString(),
		CIDRDestination: data.CIDRDestination.ValueString(),
		Description:     data.Description.ValueStringPointer(),
	})
	if err != nil {
		resp.Diagnostics.AddError(utils.ParseSDKError(err))
		return
	}

	route, err := r.WaitUntilRouteSatusMatches(ctx, vpcID, createdRoute.ID, string(netSDK.RouteStatusCreated))
	if err != nil {
		resp.Diagnostics.AddError(utils.ParseSDKError(err))
		data.ID = types.StringValue(createdRoute.ID)
		resp.State.Set(ctx, &data)
		return
	}

	tfResult := convertSDKRouteResultToTerraformNetworkVpcsRouteModel(route)
	resp.Diagnostics.Append(resp.State.Set(ctx, &tfResult)...)
}

func (r *NetworkVpcsRouteResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data NetworkVpcsRouteModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	route, err := r.networkRoute.Get(ctx, data.VpcID.ValueString(), data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(utils.ParseSDKError(err))
		return
	}

	tfResult := convertSDKRouteResultToTerraformNetworkVpcsRouteModel(route)
	resp.Diagnostics.Append(resp.State.Set(ctx, &tfResult)...)
}

func (r *NetworkVpcsRouteResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data NetworkVpcsRouteModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.networkRoute.Delete(ctx, data.VpcID.ValueString(), data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(utils.ParseSDKError(err))
		return
	}

	_, err = r.WaitUntilRouteSatusMatches(ctx, data.ID.ValueString(), string(netSDK.RouteStatusDeleted))
	if err != nil {
		switch e := err.(type) {
		case *clientSDK.HTTPError:
			if e.StatusCode == http.StatusNotFound {
				return
			}
		default:
			resp.Diagnostics.AddError(utils.ParseSDKError(err))
			return
		}
	}
}

func (r *NetworkVpcsRouteResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError("Update is not supported for route", "")
}

func (r *NetworkVpcsRouteResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.Split(req.ID, ",")
	if len(parts) != 2 {
		resp.Diagnostics.AddError("Invalid import format", "Use `<vpc_id>,<route_id>`")
		return
	}

	resp.Diagnostics.Append(
		resp.State.SetAttribute(ctx, path.Root("vpc_id"), parts[0])...,
	)
	resp.Diagnostics.Append(
		resp.State.SetAttribute(ctx, path.Root("id"), parts[1])...,
	)
}

func (r *NetworkVpcsRouteResource) WaitUntilRouteSatusMatches(ctx context.Context, vpcID, routeID string, expectedStatus ...string) (*netSDK.VpcsRoute, error) {
	var result *netSDK.VpcsRoute
	var err error

	for startTime := time.Now(); time.Since(startTime) < RoutePoolingTimeout; {
		time.Sleep(1 * time.Minute)

		result, err = r.networkRoute.Get(ctx, vpcID, routeID)
		if err != nil {
			return nil, err
		}

		status := strings.ToLower(string(result.Status))

		if slices.Contains(expectedStatus, status) {
			return result, nil
		}
		if status == string(netSDK.RouteStatusError) {
			return result, errors.New("route provisioning failure")
		}

		tflog.Debug(ctx, fmt.Sprintf("current route status: [%s]", status))
	}

	return result, errors.New("timeout waiting for route to provision")
}

func convertSDKRouteResultToTerraformNetworkVpcsRouteModel(sdkResult *netSDK.VpcsRoute) *NetworkVpcsRouteModel {
	if sdkResult == nil {
		return nil
	}

	tfModel := &NetworkVpcsRouteModel{
		ID:              types.StringValue(sdkResult.ID),
		VpcID:           types.StringValue(sdkResult.VpcID),
		PortID:          types.StringValue(sdkResult.PortID),
		CIDRDestination: types.StringValue(sdkResult.CIDRDestination),
		NextHop:         types.StringValue(sdkResult.NextHop),
		Type:            types.StringValue(sdkResult.Type),
		Status:          types.StringValue(string(sdkResult.Status)),
	}

	var description *string
	if sdkResult.Description != "" {
		description = &sdkResult.Description
	}
	tfModel.Description = types.StringPointerValue(description)

	return tfModel
}
