package network

import (
	"context"
	"errors"
	"testing"

	netSDK "github.com/MagaluCloud/mgc-sdk-go/network"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	dschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func peeringTestDataSourceSchema() dschema.Schema {
	d := &NetworkVpcsPeeringDatasource{}
	resp := &datasource.SchemaResponse{}
	d.Schema(context.Background(), datasource.SchemaRequest{}, resp)
	return resp.Schema
}

func TestNetworkVpcsPeeringDatasource_Schema(t *testing.T) {
	t.Parallel()

	attrs := peeringTestDataSourceSchema().Attributes

	id, ok := attrs["id"].(dschema.StringAttribute)
	require.True(t, ok)
	assert.True(t, id.Required)

	for _, name := range []string{"name", "description", "status", "requester_vpc_id", "accepter_vpc_id", "created_at", "updated_at"} {
		attr, ok := attrs[name].(dschema.StringAttribute)
		require.True(t, ok, name)
		assert.True(t, attr.Computed, "%s must be computed", name)
	}
}

func TestNetworkVpcsPeeringDatasource_Read(t *testing.T) {
	t.Parallel()

	fullPeering := fullSDKPeering(t)

	tests := []struct {
		name        string
		peering     *netSDK.VpcsPeering
		getError    error
		expectError bool
	}{
		{
			name:    "peering found",
			peering: &fullPeering,
		},
		{
			name:        "peering not found",
			getError:    notFoundError(),
			expectError: true,
		},
		{
			// The resource removes a deleted peering from the state; the data
			// source has to agree that it no longer exists.
			name:        "soft-deleted peering counts as not found",
			peering:     sdkPeering(netSDK.VpcsPeeringStatusDeleted),
			expectError: true,
		},
		{
			name:        "get fails",
			getError:    errors.New("sdk error"),
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockSvc := &mockVpcsPeeringsService{}
			if tt.getError != nil {
				mockSvc.On("Get", mock.Anything, "peering-1").Return(nil, tt.getError)
			} else {
				mockSvc.On("Get", mock.Anything, "peering-1").Return(tt.peering, nil)
			}

			d := &NetworkVpcsPeeringDatasource{networkPeering: mockSvc}

			config := dataSourceTestConfig(t, peeringTestDataSourceSchema(), NetworkVpcsPeeringDataSourceModel{
				ID: types.StringValue("peering-1"),
			})

			resp := &datasource.ReadResponse{State: tfsdk.State{Schema: peeringTestDataSourceSchema()}}
			d.Read(context.Background(), datasource.ReadRequest{Config: config}, resp)

			if tt.expectError {
				require.True(t, resp.Diagnostics.HasError())
				return
			}

			require.False(t, resp.Diagnostics.HasError(), resp.Diagnostics)

			var state NetworkVpcsPeeringDataSourceModel
			resp.State.Get(context.Background(), &state)
			assert.Equal(t, "peering-1", state.ID.ValueString())
			assert.Equal(t, "peering-prod-to-db", state.Name.ValueString())
			assert.Equal(t, "vpc-requester", state.RequesterVpcID.ValueString())
			assert.Equal(t, "vpc-accepter", state.AccepterVpcID.ValueString())
			assert.Equal(t, "created", state.Status.ValueString())
			assert.Equal(t, "2026-07-23T20:21:08Z", state.CreatedAt.ValueString())
			mockSvc.AssertExpectations(t)
		})
	}
}
