package network

import (
	"context"
	"errors"
	"testing"

	netSDK "github.com/MagaluCloud/mgc-sdk-go/network"

	"github.com/MagaluCloud/terraform-provider-mgc/mgc/internal/testutils"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type mockPublicIPService struct {
	mock.Mock
	netSDK.PublicIPService
}

func (m *mockPublicIPService) Get(ctx context.Context, id string) (*netSDK.PublicIPResponse, error) {
	args := m.Called(ctx, id)
	if result := args.Get(0); result != nil {
		return result.(*netSDK.PublicIPResponse), args.Error(1)
	}
	return nil, args.Error(1)
}

type mockPublicIPVPCService struct {
	mock.Mock
	netSDK.VPCService
}

func (m *mockPublicIPVPCService) CreatePublicIP(ctx context.Context, vpcID string, req netSDK.PublicIPCreateRequest) (string, error) {
	args := m.Called(ctx, vpcID, req)
	return args.String(0), args.Error(1)
}

func publicIPState(t *testing.T, r *NetworkPublicIPResource, data NetworkPublicIPModel) tfsdk.State {
	t.Helper()
	schemaResp := testutils.GetResourceTestSchema(t, r)
	state := tfsdk.State{Schema: schemaResp.Schema}
	diags := state.Set(context.Background(), data)
	require.False(t, diags.HasError(), "%v", diags)
	return state
}

func TestNetworkPublicIPResource_Read_SetsVPCIdAfterImport(t *testing.T) {
	pipSvc := &mockPublicIPService{}
	pipSvc.On("Get", mock.Anything, "pip-1").Return(&netSDK.PublicIPResponse{
		ID:       ptrTo("pip-1"),
		VPCID:    ptrTo("vpc-1"),
		PublicIP: ptrTo("203.0.113.10"),
	}, nil)
	r := &NetworkPublicIPResource{networkPIP: pipSvc}

	// ImportState only sets the id, so vpc_id is null until Read fills it.
	state := publicIPState(t, r, NetworkPublicIPModel{
		Id:          types.StringValue("pip-1"),
		PublicIP:    types.StringNull(),
		Description: types.StringNull(),
		VPCId:       types.StringNull(),
	})
	resp := &resource.ReadResponse{State: state}

	r.Read(context.Background(), resource.ReadRequest{State: state}, resp)

	require.False(t, resp.Diagnostics.HasError(), "%v", resp.Diagnostics)
	var got NetworkPublicIPModel
	require.False(t, resp.State.Get(context.Background(), &got).HasError())
	assert.Equal(t, types.StringValue("vpc-1"), got.VPCId)
	assert.Equal(t, types.StringValue("203.0.113.10"), got.PublicIP)
	pipSvc.AssertExpectations(t)
}

func TestNetworkPublicIPResource_Create_GetErrorKeepsCreatedID(t *testing.T) {
	vpcSvc := &mockPublicIPVPCService{}
	vpcSvc.On("CreatePublicIP", mock.Anything, "vpc-1", mock.Anything).Return("pip-new", nil)
	pipSvc := &mockPublicIPService{}
	pipSvc.On("Get", mock.Anything, "pip-new").Return(nil, errors.New("api error"))
	r := &NetworkPublicIPResource{networkPIP: pipSvc, networkVpc: vpcSvc}

	schemaResp := testutils.GetResourceTestSchema(t, r)
	plan := tfsdk.Plan{Schema: schemaResp.Schema}
	require.False(t, plan.Set(context.Background(), NetworkPublicIPModel{
		Id:          types.StringUnknown(),
		PublicIP:    types.StringUnknown(),
		Description: types.StringNull(),
		VPCId:       types.StringValue("vpc-1"),
	}).HasError())
	resp := &resource.CreateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}

	require.NotPanics(t, func() {
		r.Create(context.Background(), resource.CreateRequest{Plan: plan}, resp)
	})

	require.True(t, resp.Diagnostics.HasError())
	var got NetworkPublicIPModel
	require.False(t, resp.State.Get(context.Background(), &got).HasError())
	assert.Equal(t, types.StringValue("pip-new"), got.Id)
	assert.Equal(t, types.StringValue("vpc-1"), got.VPCId)
}
