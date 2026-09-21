package network

import (
	"context"
	"encoding/json"
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

func (m *mockVpcsPeeringsService) ListAll(ctx context.Context, opts *netSDK.ListAllVpcsPeeringsOptions) ([]netSDK.VpcsPeering, error) {
	args := m.Called(ctx, opts)
	result := args.Get(0)
	if result == nil {
		return nil, args.Error(1)
	}
	return result.([]netSDK.VpcsPeering), args.Error(1)
}

// sdkPeeringFromJSON builds the fixture through the SDK's own unmarshaling, the
// only way to fill the timestamps from outside the SDK module (their type is
// internal) — and what the converter sees is exactly what the wire produces.
func sdkPeeringFromJSON(t *testing.T, raw string) netSDK.VpcsPeering {
	t.Helper()

	var peering netSDK.VpcsPeering
	require.NoError(t, json.Unmarshal([]byte(raw), &peering))
	return peering
}

func fullSDKPeering(t *testing.T) netSDK.VpcsPeering {
	return sdkPeeringFromJSON(t, `{
		"id": "peering-1",
		"name": "peering-prod-to-db",
		"description": "a description",
		"status": "created",
		"created_at": "2026-07-23T20:21:08.861443",
		"updated": "2026-07-24T17:47:13.459860",
		"members": [
			{"id": "member-1", "vpc_id": "vpc-requester", "direct_role": "requester"},
			{"id": "member-2", "vpc_id": "vpc-accepter", "direct_role": "accepter"}
		]
	}`)
}

func peeringsTestSchema() dschema.Schema {
	d := &NetworkVpcsPeeringsDatasource{}
	resp := &datasource.SchemaResponse{}
	d.Schema(context.Background(), datasource.SchemaRequest{}, resp)
	return resp.Schema
}

// dataSourceTestConfig builds a tfsdk.Config from a model. Config has no Set,
// so the raw value is assembled through a throwaway State with the same schema.
func dataSourceTestConfig(t *testing.T, schema dschema.Schema, model any) tfsdk.Config {
	t.Helper()

	state := tfsdk.State{Schema: schema}
	diags := state.Set(context.Background(), model)
	require.False(t, diags.HasError(), diags)
	return tfsdk.Config{Schema: schema, Raw: state.Raw}
}

func TestConvertSDKVpcsPeeringToDataSourceModel(t *testing.T) {
	t.Parallel()

	t.Run("full peering", func(t *testing.T) {
		t.Parallel()

		tfModel := convertSDKVpcsPeeringToDataSourceModel(fullSDKPeering(t))

		assert.Equal(t, "peering-1", tfModel.ID.ValueString())
		assert.Equal(t, "peering-prod-to-db", tfModel.Name.ValueString())
		assert.Equal(t, "a description", tfModel.Description.ValueString())
		assert.Equal(t, "created", tfModel.Status.ValueString())
		assert.Equal(t, "vpc-requester", tfModel.RequesterVpcID.ValueString())
		assert.Equal(t, "vpc-accepter", tfModel.AccepterVpcID.ValueString())
		assert.Equal(t, "2026-07-23T20:21:08Z", tfModel.CreatedAt.ValueString())
		assert.Equal(t, "2026-07-24T17:47:13Z", tfModel.UpdatedAt.ValueString())
	})

	t.Run("optional fields absent stay null", func(t *testing.T) {
		t.Parallel()

		tfModel := convertSDKVpcsPeeringToDataSourceModel(sdkPeeringFromJSON(t,
			`{"id": "peering-2", "name": "minimal", "status": "pending", "members": []}`))

		assert.True(t, tfModel.Description.IsNull())
		assert.True(t, tfModel.CreatedAt.IsNull())
		assert.True(t, tfModel.UpdatedAt.IsNull())
		assert.True(t, tfModel.RequesterVpcID.IsNull())
		assert.True(t, tfModel.AccepterVpcID.IsNull())
	})

	t.Run("unknown member role is ignored", func(t *testing.T) {
		t.Parallel()

		tfModel := convertSDKVpcsPeeringToDataSourceModel(sdkPeeringFromJSON(t, `{
			"id": "peering-3", "name": "odd", "status": "pending",
			"members": [{"id": "member-1", "vpc_id": "vpc-x", "direct_role": "observer"}]
		}`))

		assert.True(t, tfModel.RequesterVpcID.IsNull())
		assert.True(t, tfModel.AccepterVpcID.IsNull())
	})
}

func TestNetworkVpcsPeeringsDatasource_Read(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		configVpcID   types.String
		expectedOpts  *netSDK.ListAllVpcsPeeringsOptions
		peerings      []netSDK.VpcsPeering
		expectedIDs   []string
		expectedError bool
	}{
		{
			name:         "all tenant peerings",
			configVpcID:  types.StringNull(),
			expectedOpts: &netSDK.ListAllVpcsPeeringsOptions{},
			peerings: []netSDK.VpcsPeering{
				fullSDKPeering(t),
				sdkPeeringFromJSON(t, `{"id": "peering-2", "name": "minimal", "status": "pending", "members": []}`),
			},
			expectedIDs: []string{"peering-1", "peering-2"},
		},
		{
			name:         "vpc filter is forwarded to the API",
			configVpcID:  types.StringValue("vpc-1"),
			expectedOpts: &netSDK.ListAllVpcsPeeringsOptions{VpcID: "vpc-1"},
			peerings:     []netSDK.VpcsPeering{fullSDKPeering(t)},
			expectedIDs:  []string{"peering-1"},
		},
		{
			// The API keeps answering deleted peerings for a while; a listing
			// that shows them would disagree with the resource, which treats
			// deleted as gone.
			name:         "soft-deleted peerings are dropped",
			configVpcID:  types.StringNull(),
			expectedOpts: &netSDK.ListAllVpcsPeeringsOptions{},
			peerings: []netSDK.VpcsPeering{
				fullSDKPeering(t),
				sdkPeeringFromJSON(t, `{"id": "peering-gone", "name": "gone", "status": "deleted", "members": []}`),
			},
			expectedIDs: []string{"peering-1"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockSvc := &mockVpcsPeeringsService{}
			mockSvc.On("ListAll", mock.Anything, tt.expectedOpts).Return(tt.peerings, nil)

			d := &NetworkVpcsPeeringsDatasource{networkPeering: mockSvc}

			config := dataSourceTestConfig(t, peeringsTestSchema(), NetworkVpcsPeeringsDataSourceModel{VpcID: tt.configVpcID})

			resp := &datasource.ReadResponse{State: tfsdk.State{Schema: peeringsTestSchema()}}
			d.Read(context.Background(), datasource.ReadRequest{Config: config}, resp)

			require.False(t, resp.Diagnostics.HasError(), resp.Diagnostics)

			var state NetworkVpcsPeeringsDataSourceModel
			resp.State.Get(context.Background(), &state)

			ids := make([]string, 0, len(state.Items))
			for _, item := range state.Items {
				ids = append(ids, item.ID.ValueString())
			}
			assert.Equal(t, tt.expectedIDs, ids)
			mockSvc.AssertExpectations(t)
		})
	}
}

func TestNetworkVpcsPeeringsDatasource_ReadError(t *testing.T) {
	t.Parallel()

	mockSvc := &mockVpcsPeeringsService{}
	mockSvc.On("ListAll", mock.Anything, mock.Anything).Return(nil, errors.New("sdk error"))

	d := &NetworkVpcsPeeringsDatasource{networkPeering: mockSvc}

	config := dataSourceTestConfig(t, peeringsTestSchema(), NetworkVpcsPeeringsDataSourceModel{VpcID: types.StringNull()})

	resp := &datasource.ReadResponse{State: tfsdk.State{Schema: peeringsTestSchema()}}
	d.Read(context.Background(), datasource.ReadRequest{Config: config}, resp)

	require.True(t, resp.Diagnostics.HasError())
}
