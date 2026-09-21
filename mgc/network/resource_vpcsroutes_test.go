package network

import (
	"context"
	"errors"
	"testing"

	netSDK "github.com/MagaluCloud/mgc-sdk-go/network"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type mockVpcsRoutesService struct {
	mock.Mock
	netSDK.VpcsRoutesService
}

func (m *mockVpcsRoutesService) Create(ctx context.Context, vpcID string, req netSDK.VpcsRoutesCreateRequest) (*netSDK.VpcsRoutesCreateResponse, error) {
	args := m.Called(ctx, vpcID, req)
	result := args.Get(0)
	if result == nil {
		return nil, args.Error(1)
	}
	return result.(*netSDK.VpcsRoutesCreateResponse), args.Error(1)
}

func (m *mockVpcsRoutesService) Get(ctx context.Context, vpcID, routeID string) (*netSDK.VpcsRoute, error) {
	args := m.Called(ctx, vpcID, routeID)
	result := args.Get(0)
	if result == nil {
		return nil, args.Error(1)
	}
	return result.(*netSDK.VpcsRoute), args.Error(1)
}

func routeTestSchema() schema.Schema {
	r := &NetworkVpcsRouteResource{}
	resp := &resource.SchemaResponse{}
	r.Schema(context.Background(), resource.SchemaRequest{}, resp)
	return resp.Schema
}

// sdkRoute answers a GET with the flat shape the API uses: the field that does
// not match the route's target comes back null, which the SDK maps to "".
func sdkRoute(portID, peeringID string) *netSDK.VpcsRoute {
	return &netSDK.VpcsRoute{
		VpcsRouteDetail: netSDK.VpcsRouteDetail{
			ID:              "route-123",
			PortID:          portID,
			VPCPeeringID:    peeringID,
			CIDRDestination: "10.0.0.0/16",
			Description:     "a description",
			NextHop:         "192.168.0.1",
			Type:            "static",
			Status:          netSDK.RouteStatusCreated,
		},
		VpcID: "vpc-1",
	}
}

// The API takes exactly one target per route (targets is an object, not a
// list), so the two next-hop attributes must be optional, mutually exclusive
// and immutable.
func TestNetworkVpcsRouteResource_SchemaNextHopAttributes(t *testing.T) {
	t.Parallel()

	attrs := routeTestSchema().Attributes

	for _, name := range []string{"port_id", "peering_id"} {
		attr, ok := attrs[name].(schema.StringAttribute)
		require.True(t, ok, name)

		assert.True(t, attr.Optional, "%s must be optional", name)
		assert.False(t, attr.Required, "%s must not be required", name)
		assert.Len(t, attr.PlanModifiers, 1, "%s must require replace", name)

		require.Len(t, attr.Validators, 1, "%s must carry the exclusivity validator", name)
		description := attr.Validators[0].MarkdownDescription(context.Background())
		assert.Contains(t, description, "port_id", name)
		assert.Contains(t, description, "peering_id", name)
	}
}

func TestNetworkVpcsRouteResource_Create(t *testing.T) {
	t.Parallel()

	description := "a description"

	tests := []struct {
		name            string
		plan            NetworkVpcsRouteModel
		expectedRequest netSDK.VpcsRoutesCreateRequest
		routeFromAPI    *netSDK.VpcsRoute
	}{
		{
			name: "port target",
			plan: NetworkVpcsRouteModel{
				VpcID:           types.StringValue("vpc-1"),
				PortID:          types.StringValue("port-1"),
				CIDRDestination: types.StringValue("10.0.0.0/16"),
				Description:     types.StringValue(description),
			},
			expectedRequest: netSDK.VpcsRoutesCreateRequest{
				CIDRDestination: "10.0.0.0/16",
				Description:     &description,
				Targets:         netSDK.TargetsRequest{ID: "port-1", Type: "port_id"},
			},
			routeFromAPI: sdkRoute("port-1", ""),
		},
		{
			name: "vpc peering target",
			plan: NetworkVpcsRouteModel{
				VpcID:           types.StringValue("vpc-1"),
				PeeringID:       types.StringValue("peering-1"),
				CIDRDestination: types.StringValue("10.0.0.0/16"),
				Description:     types.StringValue(description),
			},
			expectedRequest: netSDK.VpcsRoutesCreateRequest{
				CIDRDestination: "10.0.0.0/16",
				Description:     &description,
				Targets:         netSDK.TargetsRequest{ID: "peering-1", Type: "vpc_peering"},
			},
			routeFromAPI: sdkRoute("", "peering-1"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockSvc := &mockVpcsRoutesService{}
			mockSvc.On("Create", mock.Anything, "vpc-1", tt.expectedRequest).Return(
				&netSDK.VpcsRoutesCreateResponse{ID: "route-123", Status: netSDK.RouteStatusPending}, nil)
			mockSvc.On("Get", mock.Anything, "vpc-1", "route-123").Return(tt.routeFromAPI, nil)

			r := &NetworkVpcsRouteResource{networkRoute: mockSvc}

			plan := tfsdk.Plan{Schema: routeTestSchema()}
			plan.Set(context.Background(), tt.plan)

			resp := &resource.CreateResponse{State: tfsdk.State{Schema: routeTestSchema()}}
			r.Create(context.Background(), resource.CreateRequest{Plan: plan}, resp)

			require.False(t, resp.Diagnostics.HasError(), resp.Diagnostics)

			var state NetworkVpcsRouteModel
			resp.State.Get(context.Background(), &state)
			assert.Equal(t, "route-123", state.ID.ValueString())
			assert.Equal(t, tt.plan.PortID, state.PortID)
			assert.Equal(t, tt.plan.PeeringID, state.PeeringID)
			assert.Equal(t, "created", state.Status.ValueString())
			mockSvc.AssertExpectations(t)
		})
	}
}

func TestNetworkVpcsRouteResource_CreateFails(t *testing.T) {
	t.Parallel()

	mockSvc := &mockVpcsRoutesService{}
	mockSvc.On("Create", mock.Anything, "vpc-1", mock.Anything).Return(nil, errors.New("sdk error"))

	r := &NetworkVpcsRouteResource{networkRoute: mockSvc}

	plan := tfsdk.Plan{Schema: routeTestSchema()}
	plan.Set(context.Background(), NetworkVpcsRouteModel{
		VpcID:           types.StringValue("vpc-1"),
		PortID:          types.StringValue("port-1"),
		CIDRDestination: types.StringValue("10.0.0.0/16"),
	})

	resp := &resource.CreateResponse{State: tfsdk.State{Schema: routeTestSchema()}}
	r.Create(context.Background(), resource.CreateRequest{Plan: plan}, resp)

	require.True(t, resp.Diagnostics.HasError())
}

// Read mirrors the API: whichever target field the route does not use must land
// in the state as null, exactly what the config holds. "" there would be
// reported as drift.
func TestNetworkVpcsRouteResource_Read(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		routeFromAPI    *netSDK.VpcsRoute
		expectedPort    types.String
		expectedPeering types.String
	}{
		{
			name:            "port route keeps peering_id null",
			routeFromAPI:    sdkRoute("port-1", ""),
			expectedPort:    types.StringValue("port-1"),
			expectedPeering: types.StringNull(),
		},
		{
			name:            "peering route keeps port_id null",
			routeFromAPI:    sdkRoute("", "peering-1"),
			expectedPort:    types.StringNull(),
			expectedPeering: types.StringValue("peering-1"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockSvc := &mockVpcsRoutesService{}
			mockSvc.On("Get", mock.Anything, "vpc-1", "route-123").Return(tt.routeFromAPI, nil)

			r := &NetworkVpcsRouteResource{networkRoute: mockSvc}

			state := tfsdk.State{Schema: routeTestSchema()}
			state.Set(context.Background(), NetworkVpcsRouteModel{
				ID:    types.StringValue("route-123"),
				VpcID: types.StringValue("vpc-1"),
			})

			resp := &resource.ReadResponse{State: state}
			r.Read(context.Background(), resource.ReadRequest{State: state}, resp)

			require.False(t, resp.Diagnostics.HasError(), resp.Diagnostics)

			var read NetworkVpcsRouteModel
			resp.State.Get(context.Background(), &read)
			assert.Equal(t, tt.expectedPort, read.PortID)
			assert.Equal(t, tt.expectedPeering, read.PeeringID)
		})
	}
}

func TestNetworkVpcsRouteResource_ReadNotFoundRemovesState(t *testing.T) {
	t.Parallel()

	mockSvc := &mockVpcsRoutesService{}
	mockSvc.On("Get", mock.Anything, "vpc-1", "route-123").Return(nil, notFoundError())

	r := &NetworkVpcsRouteResource{networkRoute: mockSvc}

	state := tfsdk.State{Schema: routeTestSchema()}
	state.Set(context.Background(), NetworkVpcsRouteModel{
		ID:    types.StringValue("route-123"),
		VpcID: types.StringValue("vpc-1"),
	})

	resp := &resource.ReadResponse{State: state}
	r.Read(context.Background(), resource.ReadRequest{State: state}, resp)

	require.False(t, resp.Diagnostics.HasError())
	assert.True(t, resp.State.Raw.IsNull())
}

func TestConvertSDKRouteResultToTerraformNetworkVpcsRouteModel(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		route           *netSDK.VpcsRoute
		expectedPort    types.String
		expectedPeering types.String
	}{
		{
			name:            "port route",
			route:           sdkRoute("port-1", ""),
			expectedPort:    types.StringValue("port-1"),
			expectedPeering: types.StringNull(),
		},
		{
			name:            "peering route",
			route:           sdkRoute("", "peering-1"),
			expectedPort:    types.StringNull(),
			expectedPeering: types.StringValue("peering-1"),
		},
		{
			// System routes (e.g. the default one) may carry no target at all.
			name:            "route without target",
			route:           sdkRoute("", ""),
			expectedPort:    types.StringNull(),
			expectedPeering: types.StringNull(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tfModel := convertSDKRouteResultToTerraformNetworkVpcsRouteModel(tt.route)

			require.NotNil(t, tfModel)
			assert.Equal(t, tt.expectedPort, tfModel.PortID)
			assert.Equal(t, tt.expectedPeering, tfModel.PeeringID)
			assert.Equal(t, "route-123", tfModel.ID.ValueString())
			assert.Equal(t, "vpc-1", tfModel.VpcID.ValueString())
		})
	}

	t.Run("nil route", func(t *testing.T) {
		t.Parallel()
		assert.Nil(t, convertSDKRouteResultToTerraformNetworkVpcsRouteModel(nil))
	})
}
