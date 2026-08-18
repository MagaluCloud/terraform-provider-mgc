package network

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"
	"time"

	clientSDK "github.com/MagaluCloud/mgc-sdk-go/client"
	netSDK "github.com/MagaluCloud/mgc-sdk-go/network"

	"github.com/MagaluCloud/terraform-provider-mgc/mgc/utils"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type mockVpcsPeeringsService struct {
	mock.Mock
	netSDK.VpcsPeeringsService
}

func (m *mockVpcsPeeringsService) Create(ctx context.Context, req netSDK.VpcsPeeringsCreateRequest) (*netSDK.VpcsPeeringsCreateResponse, error) {
	args := m.Called(ctx, req)
	result := args.Get(0)
	if result == nil {
		return nil, args.Error(1)
	}
	return result.(*netSDK.VpcsPeeringsCreateResponse), args.Error(1)
}

func (m *mockVpcsPeeringsService) Get(ctx context.Context, peeringID string) (*netSDK.VpcsPeering, error) {
	args := m.Called(ctx, peeringID)
	result := args.Get(0)
	if result == nil {
		return nil, args.Error(1)
	}
	return result.(*netSDK.VpcsPeering), args.Error(1)
}

func (m *mockVpcsPeeringsService) Delete(ctx context.Context, peeringID string) error {
	args := m.Called(ctx, peeringID)
	return args.Error(0)
}

func peeringTestSchema() schema.Schema {
	r := &NetworkVpcsPeeringResource{}
	resp := &resource.SchemaResponse{}
	r.Schema(context.Background(), resource.SchemaRequest{}, resp)
	return resp.Schema
}

func notFoundError() error {
	return &clientSDK.HTTPError{
		StatusCode: http.StatusNotFound,
		Status:     "404 Not Found",
		Body:       []byte(`{"error":"not found"}`),
		Response:   &http.Response{Header: http.Header{}},
	}
}

func sdkPeering(status netSDK.VpcsPeeringStatus) *netSDK.VpcsPeering {
	description := "a description"
	return &netSDK.VpcsPeering{
		ID:          "peering-123",
		Name:        "my-peering",
		Description: &description,
		Status:      status,
		Members: []netSDK.VpcsPeeringMember{
			{ID: "member-1", VpcID: "vpc-requester", DirectRole: netSDK.VpcsPeeringDirectRoleRequester},
			{ID: "member-2", VpcID: "vpc-accepter", DirectRole: netSDK.VpcsPeeringDirectRoleAccepter},
		},
	}
}

func TestNewNetworkVpcsPeeringResource(t *testing.T) {
	t.Parallel()

	r := NewNetworkVpcsPeeringResource()
	require.NotNil(t, r)
	require.IsType(t, &NetworkVpcsPeeringResource{}, r)

	assert.NotSame(t, r, NewNetworkVpcsPeeringResource())
}

func TestNetworkVpcsPeeringResource_Metadata(t *testing.T) {
	t.Parallel()

	r := &NetworkVpcsPeeringResource{}
	resp := &resource.MetadataResponse{}

	r.Metadata(context.Background(), resource.MetadataRequest{ProviderTypeName: "mgc"}, resp)

	assert.Equal(t, "mgc_network_vpcs_peering", resp.TypeName)
}

func TestNetworkVpcsPeeringResource_Configure(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		providerData any
		expectError  bool
	}{
		{name: "nil provider data", providerData: nil, expectError: false},
		{name: "invalid provider data type", providerData: "invalid", expectError: true},
		{name: "valid provider data", providerData: utils.DataConfig{ApiKey: "test-key", Region: "test-region"}, expectError: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			r := &NetworkVpcsPeeringResource{}
			resp := &resource.ConfigureResponse{}

			r.Configure(context.Background(), resource.ConfigureRequest{ProviderData: tt.providerData}, resp)

			assert.Equal(t, tt.expectError, resp.Diagnostics.HasError())
		})
	}
}

// Every configurable attribute must require replace: the peering API has no update endpoint.
func TestNetworkVpcsPeeringResource_SchemaRequiresReplace(t *testing.T) {
	t.Parallel()

	attrs := peeringTestSchema().Attributes

	for _, name := range []string{"id", "name", "description", "requester_vpc_id", "accepter_vpc_id", "status", "created_at", "updated_at"} {
		require.Contains(t, attrs, name)
	}

	for _, name := range []string{"name", "description", "requester_vpc_id", "accepter_vpc_id"} {
		attr, ok := attrs[name].(schema.StringAttribute)
		require.True(t, ok, name)
		require.Len(t, attr.PlanModifiers, 1, "%s must require replace", name)
	}

	// created_at holds its known value across plans; updated_at must refresh from the API.
	createdAt, ok := attrs["created_at"].(schema.StringAttribute)
	require.True(t, ok)
	assert.Len(t, createdAt.PlanModifiers, 1, "created_at should keep its known value")

	updatedAt, ok := attrs["updated_at"].(schema.StringAttribute)
	require.True(t, ok)
	assert.Empty(t, updatedAt.PlanModifiers, "updated_at must not use the prior state")
}

func TestNetworkVpcsPeeringResource_Create(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		mockSetup      func(*mockVpcsPeeringsService)
		expectError    bool
		expectedStatus string
	}{
		{
			name: "waits until completed",
			mockSetup: func(m *mockVpcsPeeringsService) {
				m.On("Create", mock.Anything, mock.Anything).Return(
					&netSDK.VpcsPeeringsCreateResponse{ID: "peering-123", Status: netSDK.VpcsPeeringStatusPending}, nil)
				m.On("Get", mock.Anything, "peering-123").Return(
					sdkPeering(netSDK.VpcsPeeringStatusCompleted), nil)
			},
			expectedStatus: "completed",
		},
		{
			name: "created is also a terminal status",
			mockSetup: func(m *mockVpcsPeeringsService) {
				m.On("Create", mock.Anything, mock.Anything).Return(
					&netSDK.VpcsPeeringsCreateResponse{ID: "peering-123", Status: netSDK.VpcsPeeringStatusPending}, nil)
				m.On("Get", mock.Anything, "peering-123").Return(
					sdkPeering(netSDK.VpcsPeeringStatusCreated), nil)
			},
			expectedStatus: "created",
		},
		{
			name: "create fails",
			mockSetup: func(m *mockVpcsPeeringsService) {
				m.On("Create", mock.Anything, mock.Anything).Return(nil, errors.New("sdk error"))
			},
			expectError: true,
		},
		{
			name: "provisioning ends in error status",
			mockSetup: func(m *mockVpcsPeeringsService) {
				m.On("Create", mock.Anything, mock.Anything).Return(
					&netSDK.VpcsPeeringsCreateResponse{ID: "peering-123", Status: netSDK.VpcsPeeringStatusPending}, nil)
				m.On("Get", mock.Anything, "peering-123").Return(
					sdkPeering(netSDK.VpcsPeeringStatusError), nil)
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockSvc := &mockVpcsPeeringsService{}
			tt.mockSetup(mockSvc)
			r := &NetworkVpcsPeeringResource{networkPeering: mockSvc}

			plan := tfsdk.Plan{Schema: peeringTestSchema()}
			plan.Set(context.Background(), NetworkVpcsPeeringModel{
				Name:           types.StringValue("my-peering"),
				Description:    types.StringValue("a description"),
				RequesterVpcID: types.StringValue("vpc-requester"),
				AccepterVpcID:  types.StringValue("vpc-accepter"),
			})

			resp := &resource.CreateResponse{State: tfsdk.State{Schema: peeringTestSchema()}}
			r.Create(context.Background(), resource.CreateRequest{Plan: plan}, resp)

			if tt.expectError {
				require.True(t, resp.Diagnostics.HasError())
				return
			}

			require.False(t, resp.Diagnostics.HasError())

			var state NetworkVpcsPeeringModel
			resp.State.Get(context.Background(), &state)
			assert.Equal(t, "peering-123", state.ID.ValueString())
			assert.Equal(t, tt.expectedStatus, state.Status.ValueString())
			assert.Equal(t, "my-peering", state.Name.ValueString())
			assert.Equal(t, "a description", state.Description.ValueString())
		})
	}
}

// A create that fails during polling must still leave the id in the state, so the resource
// is tainted instead of orphaned in the cloud.
func TestNetworkVpcsPeeringResource_CreatePersistsStateBeforePolling(t *testing.T) {
	t.Parallel()

	mockSvc := &mockVpcsPeeringsService{}
	mockSvc.On("Create", mock.Anything, mock.Anything).Return(
		&netSDK.VpcsPeeringsCreateResponse{ID: "peering-123", Status: netSDK.VpcsPeeringStatusPending}, nil)
	mockSvc.On("Get", mock.Anything, "peering-123").Return(
		sdkPeering(netSDK.VpcsPeeringStatusError), nil)

	r := &NetworkVpcsPeeringResource{networkPeering: mockSvc}

	plan := tfsdk.Plan{Schema: peeringTestSchema()}
	plan.Set(context.Background(), NetworkVpcsPeeringModel{
		Name:           types.StringValue("my-peering"),
		RequesterVpcID: types.StringValue("vpc-requester"),
		AccepterVpcID:  types.StringValue("vpc-accepter"),
	})

	resp := &resource.CreateResponse{State: tfsdk.State{Schema: peeringTestSchema()}}
	r.Create(context.Background(), resource.CreateRequest{Plan: plan}, resp)

	require.True(t, resp.Diagnostics.HasError())

	var state NetworkVpcsPeeringModel
	resp.State.Get(context.Background(), &state)
	assert.Equal(t, "peering-123", state.ID.ValueString())
}

// A peering on a just-created VPC can fail until the VPC is ready, so createWithRetry
// retries after each backoff. Zero backoffs keep the test instant.
func TestNetworkVpcsPeeringResource_CreateWithRetry(t *testing.T) {
	t.Parallel()

	req := netSDK.VpcsPeeringsCreateRequest{Name: "my-peering"}

	t.Run("succeeds once the vpc is ready", func(t *testing.T) {
		t.Parallel()

		mockSvc := &mockVpcsPeeringsService{}
		mockSvc.On("Create", mock.Anything, mock.Anything).Return(nil, errors.New("vpc not ready")).Twice()
		mockSvc.On("Create", mock.Anything, mock.Anything).Return(
			&netSDK.VpcsPeeringsCreateResponse{ID: "peering-123", Status: netSDK.VpcsPeeringStatusPending}, nil).Once()

		r := &NetworkVpcsPeeringResource{networkPeering: mockSvc, createBackoffs: []time.Duration{0, 0, 0}}

		got, err := r.createWithRetry(context.Background(), req)

		require.NoError(t, err)
		assert.Equal(t, "peering-123", got.ID)
		mockSvc.AssertNumberOfCalls(t, "Create", 3)
	})

	t.Run("gives up after the initial attempt plus every retry", func(t *testing.T) {
		t.Parallel()

		mockSvc := &mockVpcsPeeringsService{}
		mockSvc.On("Create", mock.Anything, mock.Anything).Return(nil, errors.New("vpc not ready"))

		r := &NetworkVpcsPeeringResource{networkPeering: mockSvc, createBackoffs: []time.Duration{0, 0, 0}}

		_, err := r.createWithRetry(context.Background(), req)

		require.Error(t, err)
		// One immediate attempt plus one per backoff.
		mockSvc.AssertNumberOfCalls(t, "Create", 4)
	})

	t.Run("a cancelled context stops the backoff", func(t *testing.T) {
		t.Parallel()

		mockSvc := &mockVpcsPeeringsService{}
		mockSvc.On("Create", mock.Anything, mock.Anything).Return(nil, errors.New("vpc not ready"))

		r := &NetworkVpcsPeeringResource{networkPeering: mockSvc, createBackoffs: []time.Duration{time.Hour}}

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		_, err := r.createWithRetry(ctx, req)

		require.ErrorIs(t, err, context.Canceled)
		mockSvc.AssertNumberOfCalls(t, "Create", 1)
	})
}

// Right after the create the peering may not be visible yet. A 404 while waiting for it to
// provision is eventual consistency, not a failure, so the waiter must keep polling — here
// it only gives up because the context expires, not because of the 404.
func TestNetworkVpcsPeeringResource_WaiterRetriesNotFoundOnCreate(t *testing.T) {
	t.Parallel()

	mockSvc := &mockVpcsPeeringsService{}
	mockSvc.On("Get", mock.Anything, "peering-123").Return(nil, notFoundError())

	r := &NetworkVpcsPeeringResource{networkPeering: mockSvc}

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, err := r.waitUntilPeeringStatusMatches(ctx, "peering-123",
		netSDK.VpcsPeeringStatusCreated,
		netSDK.VpcsPeeringStatusCompleted,
	)

	require.Error(t, err)
	assert.ErrorIs(t, err, context.DeadlineExceeded)
	mockSvc.AssertNumberOfCalls(t, "Get", 1)
}

// The same 404, while waiting for the delete, means the peering is gone: success.
func TestNetworkVpcsPeeringResource_WaiterTreatsNotFoundAsDeleted(t *testing.T) {
	t.Parallel()

	mockSvc := &mockVpcsPeeringsService{}
	mockSvc.On("Get", mock.Anything, "peering-123").Return(nil, notFoundError())

	r := &NetworkVpcsPeeringResource{networkPeering: mockSvc}

	result, err := r.waitUntilPeeringStatusMatches(context.Background(), "peering-123", netSDK.VpcsPeeringStatusDeleted)

	require.NoError(t, err)
	assert.Nil(t, result)
}

func TestNetworkVpcsPeeringResource_Read(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		mockSetup     func(*mockVpcsPeeringsService)
		expectError   bool
		expectRemoved bool
	}{
		{
			name: "refreshes status and members",
			mockSetup: func(m *mockVpcsPeeringsService) {
				m.On("Get", mock.Anything, "peering-123").Return(
					sdkPeering(netSDK.VpcsPeeringStatusCreated), nil)
			},
		},
		{
			name: "not found removes from state",
			mockSetup: func(m *mockVpcsPeeringsService) {
				m.On("Get", mock.Anything, "peering-123").Return(nil, notFoundError())
			},
			expectRemoved: true,
		},
		{
			name: "soft deleted removes from state",
			mockSetup: func(m *mockVpcsPeeringsService) {
				m.On("Get", mock.Anything, "peering-123").Return(
					sdkPeering(netSDK.VpcsPeeringStatusDeleted), nil)
			},
			expectRemoved: true,
		},
		{
			name: "generic error surfaces",
			mockSetup: func(m *mockVpcsPeeringsService) {
				m.On("Get", mock.Anything, "peering-123").Return(nil, errors.New("boom"))
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockSvc := &mockVpcsPeeringsService{}
			tt.mockSetup(mockSvc)
			r := &NetworkVpcsPeeringResource{networkPeering: mockSvc}

			state := tfsdk.State{Schema: peeringTestSchema()}
			state.Set(context.Background(), NetworkVpcsPeeringModel{
				ID:             types.StringValue("peering-123"),
				Name:           types.StringValue("my-peering"),
				Description:    types.StringValue("a description"),
				RequesterVpcID: types.StringValue("vpc-requester"),
				AccepterVpcID:  types.StringValue("vpc-accepter"),
				Status:         types.StringValue("pending"),
			})

			resp := &resource.ReadResponse{State: state}
			r.Read(context.Background(), resource.ReadRequest{State: state}, resp)

			if tt.expectError {
				require.True(t, resp.Diagnostics.HasError())
				return
			}
			require.False(t, resp.Diagnostics.HasError())

			if tt.expectRemoved {
				assert.True(t, resp.State.Raw.IsNull(), "resource must be removed from the state")
				return
			}

			var got NetworkVpcsPeeringModel
			resp.State.Get(context.Background(), &got)
			assert.Equal(t, "created", got.Status.ValueString())
			assert.Equal(t, "my-peering", got.Name.ValueString())
			assert.Equal(t, "a description", got.Description.ValueString())
		})
	}
}

// A rename outside Terraform must land in the state as drift, but a description the API
// omits is kept: flatten only overwrites the description with a non-empty value.
func TestNetworkVpcsPeeringResource_ReadRefreshesNameKeepsDescription(t *testing.T) {
	t.Parallel()

	renamed := sdkPeering(netSDK.VpcsPeeringStatusCreated)
	renamed.Name = "renamed-outside-terraform"
	renamed.Description = nil

	mockSvc := &mockVpcsPeeringsService{}
	mockSvc.On("Get", mock.Anything, "peering-123").Return(renamed, nil)

	r := &NetworkVpcsPeeringResource{networkPeering: mockSvc}

	state := tfsdk.State{Schema: peeringTestSchema()}
	state.Set(context.Background(), NetworkVpcsPeeringModel{
		ID:          types.StringValue("peering-123"),
		Name:        types.StringValue("my-peering"),
		Description: types.StringValue("a description"),
	})

	resp := &resource.ReadResponse{State: state}
	r.Read(context.Background(), resource.ReadRequest{State: state}, resp)

	require.False(t, resp.Diagnostics.HasError())

	var got NetworkVpcsPeeringModel
	resp.State.Get(context.Background(), &got)
	assert.Equal(t, "renamed-outside-terraform", got.Name.ValueString())
	assert.Equal(t, "a description", got.Description.ValueString())
}

func TestNetworkVpcsPeeringResource_Delete(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		mockSetup   func(*mockVpcsPeeringsService)
		expectError bool
	}{
		{
			name: "waits until deleted",
			mockSetup: func(m *mockVpcsPeeringsService) {
				m.On("Delete", mock.Anything, "peering-123").Return(nil)
				m.On("Get", mock.Anything, "peering-123").Return(
					sdkPeering(netSDK.VpcsPeeringStatusDeleted), nil)
			},
		},
		{
			name: "already gone is a successful delete",
			mockSetup: func(m *mockVpcsPeeringsService) {
				m.On("Delete", mock.Anything, "peering-123").Return(notFoundError())
			},
		},
		{
			name: "not found while polling is a successful delete",
			mockSetup: func(m *mockVpcsPeeringsService) {
				m.On("Delete", mock.Anything, "peering-123").Return(nil)
				m.On("Get", mock.Anything, "peering-123").Return(nil, notFoundError())
			},
		},
		{
			name: "other errors surface",
			mockSetup: func(m *mockVpcsPeeringsService) {
				m.On("Delete", mock.Anything, "peering-123").Return(errors.New("boom"))
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockSvc := &mockVpcsPeeringsService{}
			tt.mockSetup(mockSvc)
			r := &NetworkVpcsPeeringResource{networkPeering: mockSvc}

			state := tfsdk.State{Schema: peeringTestSchema()}
			state.Set(context.Background(), NetworkVpcsPeeringModel{ID: types.StringValue("peering-123")})

			resp := &resource.DeleteResponse{State: state}
			r.Delete(context.Background(), resource.DeleteRequest{State: state}, resp)

			assert.Equal(t, tt.expectError, resp.Diagnostics.HasError())
		})
	}
}

func TestNetworkVpcsPeeringResource_UpdateIsNotSupported(t *testing.T) {
	t.Parallel()

	r := &NetworkVpcsPeeringResource{}
	resp := &resource.UpdateResponse{}

	r.Update(context.Background(), resource.UpdateRequest{}, resp)

	require.True(t, resp.Diagnostics.HasError())
}

func TestNetworkVpcsPeeringResource_ImportState(t *testing.T) {
	t.Parallel()

	t.Run("the peering id comes straight from the import command", func(t *testing.T) {
		t.Parallel()

		resp := &resource.ImportStateResponse{State: emptyPeeringState(t)}
		(&NetworkVpcsPeeringResource{}).ImportState(context.Background(),
			resource.ImportStateRequest{ID: "b0bd2376-42a1-433a-bc01-d3d5ab1bb143"}, resp)

		require.False(t, resp.Diagnostics.HasError(), resp.Diagnostics)

		var imported NetworkVpcsPeeringModel
		resp.State.Get(context.Background(), &imported)

		assert.Equal(t, "b0bd2376-42a1-433a-bc01-d3d5ab1bb143", imported.ID.ValueString())
	})

	t.Run("empty id is rejected", func(t *testing.T) {
		t.Parallel()

		resp := &resource.ImportStateResponse{State: emptyPeeringState(t)}
		(&NetworkVpcsPeeringResource{}).ImportState(context.Background(),
			resource.ImportStateRequest{ID: ""}, resp)

		require.True(t, resp.Diagnostics.HasError())
	})
}

// The import only carries the id; every other attribute has to come from the Read
// the framework runs right after it.
func TestNetworkVpcsPeeringResource_ReadAfterImportFillsEveryAttribute(t *testing.T) {
	t.Parallel()

	mockSvc := &mockVpcsPeeringsService{}
	mockSvc.On("Get", mock.Anything, "peering-123").Return(
		sdkPeering(netSDK.VpcsPeeringStatusPendingRouteTable), nil)

	r := &NetworkVpcsPeeringResource{networkPeering: mockSvc}

	importResp := &resource.ImportStateResponse{State: emptyPeeringState(t)}
	r.ImportState(context.Background(), resource.ImportStateRequest{ID: "peering-123"}, importResp)
	require.False(t, importResp.Diagnostics.HasError(), importResp.Diagnostics)

	readResp := &resource.ReadResponse{State: importResp.State}
	r.Read(context.Background(), resource.ReadRequest{State: importResp.State}, readResp)
	require.False(t, readResp.Diagnostics.HasError(), readResp.Diagnostics)

	var got NetworkVpcsPeeringModel
	readResp.State.Get(context.Background(), &got)

	assert.Equal(t, "peering-123", got.ID.ValueString())
	assert.Equal(t, "my-peering", got.Name.ValueString())
	assert.Equal(t, "a description", got.Description.ValueString())
	assert.Equal(t, "pending_route", got.Status.ValueString())
	assert.Equal(t, "vpc-requester", got.RequesterVpcID.ValueString())
	assert.Equal(t, "vpc-accepter", got.AccepterVpcID.ValueString())
	mockSvc.AssertExpectations(t)
}

// emptyPeeringState is what the framework hands ImportState: the schema with every
// attribute null.
func emptyPeeringState(t *testing.T) tfsdk.State {
	t.Helper()

	state := tfsdk.State{Schema: peeringTestSchema()}
	require.False(t, state.Set(context.Background(), NetworkVpcsPeeringModel{}).HasError())
	return state
}

func TestFlattenVpcsPeering(t *testing.T) {
	t.Parallel()

	base := NetworkVpcsPeeringModel{
		Name:           types.StringValue("my-peering"),
		Description:    types.StringValue("a description"),
		RequesterVpcID: types.StringValue("vpc-requester"),
		AccepterVpcID:  types.StringValue("vpc-accepter"),
	}

	t.Run("nil response keeps the model untouched", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, base, flattenVpcsPeering(base, nil))
	})

	// Regression: a response without the id must not wipe the one the create or the
	// import already put in the state — a state without id can never be read again.
	t.Run("id absent from the response keeps the known one", func(t *testing.T) {
		t.Parallel()

		known := base
		known.ID = types.StringValue("peering-123")

		got := flattenVpcsPeering(known, &netSDK.VpcsPeering{
			Name:   "my-peering",
			Status: netSDK.VpcsPeeringStatusCreated,
		})

		assert.Equal(t, "peering-123", got.ID.ValueString())
	})

	t.Run("mirrors the full peering, members mapped onto requester and accepter", func(t *testing.T) {
		t.Parallel()

		description := "another description"
		got := flattenVpcsPeering(base, &netSDK.VpcsPeering{
			ID:          "peering-123",
			Name:        "another-name",
			Description: &description,
			Status:      netSDK.VpcsPeeringStatusCreated,
			Members: []netSDK.VpcsPeeringMember{
				{ID: "member-1", VpcID: "vpc-other-accepter", DirectRole: netSDK.VpcsPeeringDirectRoleAccepter},
				{ID: "member-2", VpcID: "vpc-other-requester", DirectRole: netSDK.VpcsPeeringDirectRoleRequester},
			},
		})

		assert.Equal(t, "peering-123", got.ID.ValueString())
		assert.Equal(t, "created", got.Status.ValueString())
		assert.Equal(t, "vpc-other-requester", got.RequesterVpcID.ValueString())
		assert.Equal(t, "vpc-other-accepter", got.AccepterVpcID.ValueString())
		assert.Equal(t, "another-name", got.Name.ValueString())
		assert.Equal(t, "another description", got.Description.ValueString())
	})

	t.Run("absent description keeps the known one", func(t *testing.T) {
		t.Parallel()

		got := flattenVpcsPeering(base, &netSDK.VpcsPeering{
			ID:     "peering-123",
			Name:   "my-peering",
			Status: netSDK.VpcsPeeringStatusPending,
		})

		assert.Equal(t, "a description", got.Description.ValueString())
	})

	t.Run("empty members keep the configured vpc ids", func(t *testing.T) {
		t.Parallel()

		got := flattenVpcsPeering(base, &netSDK.VpcsPeering{
			ID:     "peering-123",
			Name:   "my-peering",
			Status: netSDK.VpcsPeeringStatusPending,
		})

		assert.Equal(t, "vpc-requester", got.RequesterVpcID.ValueString())
		assert.Equal(t, "vpc-accepter", got.AccepterVpcID.ValueString())
	})

	// The SDK timestamps use an internal type, so build the peering through JSON — the
	// only way to populate them from outside the SDK module.
	t.Run("maps the api timestamps to rfc3339", func(t *testing.T) {
		t.Parallel()

		var peering netSDK.VpcsPeering
		require.NoError(t, json.Unmarshal([]byte(
			`{"id":"peering-123","name":"my-peering","status":"created",`+
				`"created_at":"2024-01-01T00:00:00.000000","updated":"2024-02-02T12:30:00.000000"}`), &peering))

		got := flattenVpcsPeering(base, &peering)

		assert.Equal(t, "2024-01-01T00:00:00Z", got.CreatedAt.ValueString())
		assert.Equal(t, "2024-02-02T12:30:00Z", got.UpdatedAt.ValueString())
	})

	t.Run("absent timestamps become null", func(t *testing.T) {
		t.Parallel()

		got := flattenVpcsPeering(base, &netSDK.VpcsPeering{
			ID:     "peering-123",
			Name:   "my-peering",
			Status: netSDK.VpcsPeeringStatusPending,
		})

		assert.True(t, got.CreatedAt.IsNull())
		assert.True(t, got.UpdatedAt.IsNull())
	})
}
