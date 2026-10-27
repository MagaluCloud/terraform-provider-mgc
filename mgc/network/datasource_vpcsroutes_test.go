package network

import (
	"testing"

	netSDK "github.com/MagaluCloud/mgc-sdk-go/network"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConvertSDKListRouteResultToTerraformNetworkListVpcsRouteModel(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		route           *netSDK.VpcsRouteDetail
		expectedPort    types.String
		expectedPeering types.String
	}{
		{
			name:            "port route",
			route:           &sdkRoute("port-1", "").VpcsRouteDetail,
			expectedPort:    types.StringValue("port-1"),
			expectedPeering: types.StringNull(),
		},
		{
			name:            "peering route",
			route:           &sdkRoute("", "peering-1").VpcsRouteDetail,
			expectedPort:    types.StringNull(),
			expectedPeering: types.StringValue("peering-1"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tfModel := convertSDKListRouteResultToTerraformNetworkListVpcsRouteModel(tt.route)

			require.NotNil(t, tfModel)
			assert.Equal(t, tt.expectedPort, tfModel.PortID)
			assert.Equal(t, tt.expectedPeering, tfModel.PeeringID)
		})
	}

	t.Run("nil route", func(t *testing.T) {
		t.Parallel()
		assert.Nil(t, convertSDKListRouteResultToTerraformNetworkListVpcsRouteModel(nil))
	})
}
