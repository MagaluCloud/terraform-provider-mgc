package network

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"time"

	clientSDK "github.com/MagaluCloud/mgc-sdk-go/client"
	netSDK "github.com/MagaluCloud/mgc-sdk-go/network"

	"github.com/MagaluCloud/terraform-provider-mgc/mgc/utils"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

const (
	PeeringPollingTimeout  = 30 * time.Minute
	peeringPollingInterval = 10 * time.Second
)

type NetworkVpcsPeeringModel struct {
	ID             types.String `tfsdk:"id"`
	Name           types.String `tfsdk:"name"`
	Description    types.String `tfsdk:"description"`
	RequesterVpcID types.String `tfsdk:"requester_vpc_id"`
	AccepterVpcID  types.String `tfsdk:"accepter_vpc_id"`
	Status         types.String `tfsdk:"status"`
}

type NetworkVpcsPeeringResource struct {
	networkPeering netSDK.VpcsPeeringsService
}

func NewNetworkVpcsPeeringResource() resource.Resource {
	return &NetworkVpcsPeeringResource{}
}

func (r *NetworkVpcsPeeringResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_network_vpcs_peering"
}

func (r *NetworkVpcsPeeringResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *NetworkVpcsPeeringResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Network VPC Peering. The peering API has no update endpoint, so every " +
			"attribute change replaces the resource. Import is not supported yet.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The ID of the peering.",
				Computed:    true,
			},
			"name": schema.StringAttribute{
				Description: "The name of the peering. Alphanumeric characters and hyphens, up to 50 characters.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"description": schema.StringAttribute{
				Description: "The description to help identify the peering.",
				Optional:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"requester_vpc_id": schema.StringAttribute{
				Description: "ID of the VPC requesting the peering.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"accepter_vpc_id": schema.StringAttribute{
				Description: "ID of the VPC receiving the peering invitation.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"status": schema.StringAttribute{
				Description: "Current status of the peering.",
				Computed:    true,
			},
		},
	}
}

func (r *NetworkVpcsPeeringResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data NetworkVpcsPeeringModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	created, err := r.networkPeering.Create(ctx, netSDK.VpcsPeeringsCreateRequest{
		Name:        data.Name.ValueString(),
		Description: data.Description.ValueStringPointer(),
		VPCs: netSDK.VpcsPeeringsCreateVpcs{
			RequesterVpcID: data.RequesterVpcID.ValueString(),
			AccepterVpcID:  data.AccepterVpcID.ValueString(),
		},
	})
	if err != nil {
		resp.Diagnostics.AddError(utils.ParseSDKError(err))
		return
	}

	data.ID = types.StringValue(created.ID)
	data.Status = types.StringValue(string(created.Status))
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	peering, err := r.waitUntilPeeringStatusMatches(ctx, created.ID,
		netSDK.VpcsPeeringStatusPendingRouteTable,
		netSDK.VpcsPeeringStatusCreated,
	)
	if err != nil {
		resp.Diagnostics.AddError(utils.ParseSDKError(err))
		return
	}

	data = flattenVpcsPeering(data, peering)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *NetworkVpcsPeeringResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data NetworkVpcsPeeringModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	peering, err := r.networkPeering.Get(ctx, data.ID.ValueString())
	if err != nil {
		if isPeeringNotFound(err) {
			resp.Diagnostics.AddWarning(
				"VPC peering not found",
				fmt.Sprintf("VPC peering %s no longer exists and was removed from the state.", data.ID.ValueString()),
			)
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(utils.ParseSDKError(err))
		return
	}

	// Soft delete: the API keeps answering for a while with status "deleted".
	if peering.Status == netSDK.VpcsPeeringStatusDeleted {
		resp.Diagnostics.AddWarning(
			"VPC peering deleted",
			fmt.Sprintf("VPC peering %s was deleted and was removed from the state.", data.ID.ValueString()),
		)
		resp.State.RemoveResource(ctx)
		return
	}

	data = flattenVpcsPeering(data, peering)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *NetworkVpcsPeeringResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError("Update is not supported for VPC peering", "Every attribute change replaces the resource")
}

func (r *NetworkVpcsPeeringResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data NetworkVpcsPeeringModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	peeringID := data.ID.ValueString()

	if err := r.networkPeering.Delete(ctx, peeringID); err != nil {
		// A peering that is already gone is a successful delete.
		if isPeeringNotFound(err) {
			return
		}
		resp.Diagnostics.AddError(utils.ParseSDKError(err))
		return
	}

	if _, err := r.waitUntilPeeringStatusMatches(ctx, peeringID, netSDK.VpcsPeeringStatusDeleted); err != nil {
		resp.Diagnostics.AddError(utils.ParseSDKError(err))
	}
}

// waitUntilPeeringStatusMatches polls the peering until it reaches one of the expected
// statuses.
func (r *NetworkVpcsPeeringResource) waitUntilPeeringStatusMatches(ctx context.Context, peeringID string, expectedStatus ...netSDK.VpcsPeeringStatus) (*netSDK.VpcsPeering, error) {
	for startTime := time.Now(); time.Since(startTime) < PeeringPollingTimeout; {
		result, err := r.networkPeering.Get(ctx, peeringID)

		switch {
		case err == nil:
			if slices.Contains(expectedStatus, result.Status) {
				return result, nil
			}
			if result.Status == netSDK.VpcsPeeringStatusError {
				return result, errors.New("vpc peering provisioning failure")
			}
			tflog.Debug(ctx, fmt.Sprintf("current vpc peering status: [%s]", result.Status))

		// A 404 means the peering does not exist. When waiting for the delete that is the
		// end state; right after a create it is eventual consistency, so keep polling.
		case isPeeringNotFound(err):
			if slices.Contains(expectedStatus, netSDK.VpcsPeeringStatusDeleted) {
				return nil, nil
			}

		default:
			return nil, err
		}

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(peeringPollingInterval):
		}
	}

	return nil, errors.New("timeout waiting for vpc peering to provision")
}

// flattenVpcsPeering applies the peering the API returned on top of the current model.
func flattenVpcsPeering(tfData NetworkVpcsPeeringModel, peering *netSDK.VpcsPeering) NetworkVpcsPeeringModel {
	if peering == nil {
		return tfData
	}

	tfData.ID = types.StringValue(peering.ID)
	tfData.Name = types.StringValue(peering.Name)
	tfData.Description = types.StringPointerValue(peering.Description)
	tfData.Status = types.StringValue(string(peering.Status))

	// The API describes the two sides as members with a role; the schema exposes them as
	// the two ids the user wrote. A member the API omits keeps the configured value.
	for _, member := range peering.Members {
		switch member.DirectRole {
		case netSDK.VpcsPeeringDirectRoleRequester:
			tfData.RequesterVpcID = types.StringValue(member.VpcID)
		case netSDK.VpcsPeeringDirectRoleAccepter:
			tfData.AccepterVpcID = types.StringValue(member.VpcID)
		}
	}

	return tfData
}

func isPeeringNotFound(err error) bool {
	var httpErr *clientSDK.HTTPError
	if errors.As(err, &httpErr) {
		return httpErr.StatusCode == http.StatusNotFound
	}
	return false
}
