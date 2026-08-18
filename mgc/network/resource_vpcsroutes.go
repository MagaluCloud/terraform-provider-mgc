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
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

const (
	RoutePoolingTimeout = 100 * time.Minute

	// Target type values accepted by the routes API (TargetSchema.type).
	routeTargetTypePortID     = "port_id"
	routeTargetTypeVpcPeering = "vpc_peering"
)

type NetworkVpcsRouteModel struct {
	ID              types.String `tfsdk:"id"`
	VpcID           types.String `tfsdk:"vpc_id"`
	PortID          types.String `tfsdk:"port_id"`
	PeeringID       types.String `tfsdk:"peering_id"`
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

	r.networkRoute = netSDK.New(dataConfig.CoreFor(utils.ServiceNetwork)).VpcsRoutes()
}

func (r *NetworkVpcsRouteResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Adds a route to a VPC's route table.\n\n" +
			"To let two peered VPCs reach each other, create one route on each VPC with the " +
			"destinations crossed: on each side set `peering_id` to the VPC peering and " +
			"`cidr_destination` to the CIDR of a subnet in the other VPC.\n\n" +
			"The peering must be in status `completed` before its routes take effect, and after a " +
			"route is created it can take a few minutes before connectivity is actually available.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The ID of the route.",
				Computed:    true,
			},
			"vpc_id": schema.StringAttribute{
				Description: "ID of the VPC being configured (the source side) whose route table receives this route.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"port_id": schema.StringAttribute{
				Description: "ID of the port used as the next hop for this route. Exactly one of `port_id` or `peering_id` must be set.",
				Optional:    true,
				Validators: []validator.String{
					routeTargetExactlyOneOf(),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"peering_id": schema.StringAttribute{
				Description: "ID of the VPC peering used as the next hop for this route. Exactly one of `port_id` or `peering_id` must be set.",
				Optional:    true,
				Validators: []validator.String{
					routeTargetExactlyOneOf(),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"cidr_destination": schema.StringAttribute{
				Description: "Destination CIDR block matched by this route. For a peering route, use the CIDR of a subnet in the other VPC.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"description": schema.StringAttribute{
				Description: "The description to help identify the route.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					stringplanmodifier.RequiresReplace(),
				},
			},
			"next_hop": schema.StringAttribute{
				Description: "Resolved next hop for the route, derived from the target.",
				Computed:    true,
			},
			"type": schema.StringAttribute{
				Description: "Type of the route, as defined by the networking service.",
				Computed:    true,
			},
			"status": schema.StringAttribute{
				Description: "Current status of the route.",
				Computed:    true,
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
		CIDRDestination: data.CIDRDestination.ValueString(),
		Description:     data.Description.ValueStringPointer(),
		Targets:         routeTargets(data),
	})
	if err != nil {
		resp.Diagnostics.AddError(utils.ParseSDKError(err))
		return
	}

	route, err := r.WaitUntilRouteStatusMatches(ctx, vpcID, createdRoute.ID, string(netSDK.RouteStatusCreated))
	if err != nil {
		resp.Diagnostics.AddError(utils.ParseSDKError(err))
		if fetched, getErr := r.networkRoute.Get(ctx, vpcID, createdRoute.ID); getErr == nil && fetched != nil {
			resp.Diagnostics.Append(resp.State.Set(ctx, convertSDKRouteResultToTerraformNetworkVpcsRouteModel(fetched))...)
		}
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
		if httpErr, ok := err.(*clientSDK.HTTPError); ok && httpErr.StatusCode == http.StatusNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
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

	_, err = r.WaitUntilRouteStatusMatches(ctx, data.VpcID.ValueString(), data.ID.ValueString(), string(netSDK.RouteStatusDeleted))
	if err != nil {
		switch e := err.(type) {
		case *clientSDK.HTTPError:
			if e.StatusCode == http.StatusNotFound {
				return
			} else {
				resp.Diagnostics.AddError(utils.ParseSDKError(err))
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

func (r *NetworkVpcsRouteResource) WaitUntilRouteStatusMatches(ctx context.Context, vpcID, routeID string, expectedStatus ...string) (*netSDK.VpcsRoute, error) {
	var result *netSDK.VpcsRoute
	var err error

	time.Sleep(5 * time.Second)
	for startTime := time.Now(); time.Since(startTime) < RoutePoolingTimeout; {
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
		time.Sleep(30 * time.Second)
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
		PortID:          routeTargetOrNull(sdkResult.PortID),
		PeeringID:       routeTargetOrNull(sdkResult.VPCPeeringID),
		CIDRDestination: types.StringValue(sdkResult.CIDRDestination),
		NextHop:         types.StringValue(sdkResult.NextHop),
		Type:            types.StringValue(sdkResult.Type),
		Status:          types.StringValue(string(sdkResult.Status)),
	}
	tfModel.Description = types.StringValue(sdkResult.Description)

	return tfModel
}

// routeTargetExactlyOneOf ties the next-hop attributes together: the API takes
// exactly one target per route. Each attribute carries it so the constraint is
// visible on both, and the validator deduplicates the merged self path.
func routeTargetExactlyOneOf() validator.String {
	return stringvalidator.ExactlyOneOf(
		path.MatchRoot("port_id"),
		path.MatchRoot("peering_id"),
	)
}

// routeTargets maps whichever next-hop attribute is set to the API target pair.
// The schema guarantees exactly one of them is configured.
func routeTargets(data NetworkVpcsRouteModel) netSDK.TargetsRequest {
	if !data.PeeringID.IsNull() {
		return netSDK.TargetsRequest{ID: data.PeeringID.ValueString(), Type: routeTargetTypeVpcPeering}
	}
	return netSDK.TargetsRequest{ID: data.PortID.ValueString(), Type: routeTargetTypePortID}
}

// routeTargetOrNull maps the SDK zero value back to null: the API answers with
// the target fields flattened and the one that does not apply to the route
// comes back null, which is also what the config holds.
func routeTargetOrNull(id string) types.String {
	if id == "" {
		return types.StringNull()
	}
	return types.StringValue(id)
}
