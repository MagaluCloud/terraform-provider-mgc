package kubernetes

import (
	"testing"
	"time"

	k8sSDK "github.com/MagaluCloud/mgc-sdk-go/kubernetes"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
)

func TestConvertSDKCreateResultToTerraformCreateClusterModel(t *testing.T) {
	t.Run("should return nil for nil input", func(t *testing.T) {
		result := convertSDKCreateResultToTerraformCreateClusterModel(nil)
		assert.Nil(t, result)
	})

	t.Run("should handle minimal SDK input correctly", func(t *testing.T) {
		sdkResult := &k8sSDK.Cluster{
			ID:      "cluster-123",
			Name:    "test-cluster",
			Version: "1.28.0",
		}

		result := convertSDKCreateResultToTerraformCreateClusterModel(sdkResult)

		assert.NotNil(t, result)
		assert.Equal(t, types.StringValue("cluster-123"), result.ID)
		assert.Equal(t, types.StringValue("test-cluster"), result.Name)
		assert.Equal(t, types.StringValue("1.28.0"), result.Version)
		assert.True(t, result.Description.IsNull())
		assert.True(t, result.Region.IsNull())
		assert.True(t, result.UpdatedAt.IsNull())
		assert.True(t, result.MachineTypesSource.IsNull())
		assert.True(t, result.PlatformVersion.IsNull())
	})

	t.Run("should convert full SDK input with all new fields", func(t *testing.T) {
		now := time.Now()
		region := "br-se1"
		description := "Production cluster"
		servicesIPv4CIDR := "10.96.0.0/12"
		clusterIPv4CIDR := "10.244.0.0/16"
		machineTypesSource := k8sSDK.MachineTypesSourceExternal

		sdkResult := &k8sSDK.Cluster{
			ID:                 "cluster-456",
			Name:               "prod-cluster",
			Version:            "1.29.0",
			Description:        &description,
			Region:             &region,
			CreatedAt:          &now,
			UpdatedAt:          &now,
			ServicesIpV4CIDR:   &servicesIPv4CIDR,
			ClusterIPv4CIDR:    &clusterIPv4CIDR,
			MachineTypesSource: &machineTypesSource,
			Platform: &k8sSDK.Platform{
				Version: "2.1.0",
			},
			AllowedCIDRs: &[]string{"192.168.1.0/24", "10.0.0.0/8"},
		}

		result := convertSDKCreateResultToTerraformCreateClusterModel(sdkResult)
		expectedTime := types.StringValue(now.Format(time.RFC3339))

		assert.NotNil(t, result)
		assert.Equal(t, types.StringValue("cluster-456"), result.ID)
		assert.Equal(t, types.StringValue("prod-cluster"), result.Name)
		assert.Equal(t, types.StringValue("1.29.0"), result.Version)
		assert.Equal(t, types.StringValue("Production cluster"), result.Description)
		assert.Equal(t, types.StringValue("br-se1"), result.Region)
		assert.Equal(t, expectedTime, result.CreatedAt)
		assert.Equal(t, expectedTime, result.UpdatedAt)
		assert.Equal(t, types.StringValue("10.96.0.0/12"), result.ServicesIpV4CIDR)
		assert.Equal(t, types.StringValue("10.244.0.0/16"), result.ClusterIPv4CIDR)
		assert.Equal(t, types.StringValue("external"), result.MachineTypesSource)
		assert.Equal(t, types.StringValue("2.1.0"), result.PlatformVersion)
		assert.Len(t, result.AllowedCidrs, 2)
		assert.Equal(t, types.StringValue("192.168.1.0/24"), result.AllowedCidrs[0])
		assert.Equal(t, types.StringValue("10.0.0.0/8"), result.AllowedCidrs[1])
	})

	t.Run("should handle machine types source internal", func(t *testing.T) {
		machineTypesSource := k8sSDK.MachineTypesSourceInternal
		sdkResult := &k8sSDK.Cluster{
			ID:                 "cluster-789",
			Name:               "internal-cluster",
			Version:            "1.27.0",
			MachineTypesSource: &machineTypesSource,
		}

		result := convertSDKCreateResultToTerraformCreateClusterModel(sdkResult)

		assert.NotNil(t, result)
		assert.Equal(t, types.StringValue("internal"), result.MachineTypesSource)
	})

	t.Run("should handle empty description and allowed CIDRs", func(t *testing.T) {
		emptyDescription := ""
		emptyCIDRs := []string{}

		sdkResult := &k8sSDK.Cluster{
			ID:           "cluster-empty",
			Name:         "empty-cluster",
			Version:      "1.28.0",
			Description:  &emptyDescription,
			AllowedCIDRs: &emptyCIDRs,
		}

		result := convertSDKCreateResultToTerraformCreateClusterModel(sdkResult)

		assert.NotNil(t, result)
		assert.True(t, result.Description.IsNull())
		assert.Nil(t, result.AllowedCidrs)
	})

	t.Run("should handle nil platform", func(t *testing.T) {
		sdkResult := &k8sSDK.Cluster{
			ID:       "cluster-no-platform",
			Name:     "no-platform-cluster",
			Version:  "1.28.0",
			Platform: nil,
		}

		result := convertSDKCreateResultToTerraformCreateClusterModel(sdkResult)

		assert.NotNil(t, result)
		assert.True(t, result.PlatformVersion.IsNull())
	})
}

func TestConvertStringSliceToTypesStringSlice(t *testing.T) {
	t.Run("should convert empty slice", func(t *testing.T) {
		input := []string{}
		result := convertStringSliceToTypesStringSlice(input)
		assert.Empty(t, result)
	})

	t.Run("should convert slice with values", func(t *testing.T) {
		input := []string{"value1", "value2", "value3"}
		result := convertStringSliceToTypesStringSlice(input)

		assert.Len(t, result, 3)
		assert.Equal(t, types.StringValue("value1"), result[0])
		assert.Equal(t, types.StringValue("value2"), result[1])
		assert.Equal(t, types.StringValue("value3"), result[2])
	})

	t.Run("should handle slice with empty strings", func(t *testing.T) {
		input := []string{"", "value", ""}
		result := convertStringSliceToTypesStringSlice(input)

		assert.Len(t, result, 3)
		assert.Equal(t, types.StringValue(""), result[0])
		assert.Equal(t, types.StringValue("value"), result[1])
		assert.Equal(t, types.StringValue(""), result[2])
	})
}

func TestCreateAllowedCidrs(t *testing.T) {
	t.Run("should return nil for empty slice", func(t *testing.T) {
		input := []types.String{}
		result := createAllowedCidrs(input)
		assert.Nil(t, result)
	})

	t.Run("should convert types.String slice to string slice", func(t *testing.T) {
		input := []types.String{
			types.StringValue("192.168.1.0/24"),
			types.StringValue("10.0.0.0/8"),
		}
		result := createAllowedCidrs(input)

		assert.NotNil(t, result)
		assert.Len(t, *result, 2)
		assert.Equal(t, "192.168.1.0/24", (*result)[0])
		assert.Equal(t, "10.0.0.0/8", (*result)[1])
	})

	t.Run("should handle nil input", func(t *testing.T) {
		var input []types.String
		result := createAllowedCidrs(input)
		assert.Nil(t, result)
	})
}

func TestClusterResourceValidation(t *testing.T) {
	t.Run("should validate new fields mapping", func(t *testing.T) {
		now := time.Now()
		region := "us-east-1"
		machineTypesSource := k8sSDK.MachineTypesSourceExternal

		// Create a comprehensive SDK cluster with all new fields
		sdkCluster := &k8sSDK.Cluster{
			ID:                 "validation-cluster",
			Name:               "validation-test",
			Version:            "1.29.0",
			Region:             &region,
			CreatedAt:          &now,
			UpdatedAt:          &now,
			MachineTypesSource: &machineTypesSource,
			Platform: &k8sSDK.Platform{
				Version: "3.0.0",
			},
			ServicesIpV4CIDR: stringPtr("10.96.0.0/12"),
			ClusterIPv4CIDR:  stringPtr("10.244.0.0/16"),
		}

		// Convert to Terraform model
		tfModel := convertSDKCreateResultToTerraformCreateClusterModel(sdkCluster)

		// Validate all new fields are correctly mapped
		assert.Equal(t, types.StringValue("us-east-1"), tfModel.Region)
		assert.Equal(t, types.StringValue(now.Format(time.RFC3339)), tfModel.UpdatedAt)
		assert.Equal(t, types.StringValue("external"), tfModel.MachineTypesSource)
		assert.Equal(t, types.StringValue("3.0.0"), tfModel.PlatformVersion)
		assert.Equal(t, types.StringValue("10.96.0.0/12"), tfModel.ServicesIpV4CIDR)
		assert.Equal(t, types.StringValue("10.244.0.0/16"), tfModel.ClusterIPv4CIDR)
	})

	t.Run("should handle edge cases for new fields", func(t *testing.T) {
		// Test with minimal cluster data
		sdkCluster := &k8sSDK.Cluster{
			ID:      "minimal-cluster",
			Name:    "minimal",
			Version: "1.28.0",
			// All optional fields are nil
		}

		tfModel := convertSDKCreateResultToTerraformCreateClusterModel(sdkCluster)

		// All new optional fields should be null
		assert.True(t, tfModel.Region.IsNull())
		assert.True(t, tfModel.UpdatedAt.IsNull())
		assert.True(t, tfModel.MachineTypesSource.IsNull())
		assert.True(t, tfModel.PlatformVersion.IsNull())
		assert.True(t, tfModel.ServicesIpV4CIDR.IsNull())
		assert.True(t, tfModel.ClusterIPv4CIDR.IsNull())
	})
}

func TestMachineTypesSourceEnum(t *testing.T) {
	t.Run("should handle all enum values", func(t *testing.T) {
		testCases := []struct {
			name      string
			enumValue k8sSDK.MachineTypesSource
			expected  string
		}{
			{
				name:      "external",
				enumValue: k8sSDK.MachineTypesSourceExternal,
				expected:  "external",
			},
			{
				name:      "internal",
				enumValue: k8sSDK.MachineTypesSourceInternal,
				expected:  "internal",
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				sdkCluster := &k8sSDK.Cluster{
					ID:                 "enum-test",
					Name:               "enum-cluster",
					Version:            "1.28.0",
					MachineTypesSource: &tc.enumValue,
				}

				tfModel := convertSDKCreateResultToTerraformCreateClusterModel(sdkCluster)
				assert.Equal(t, types.StringValue(tc.expected), tfModel.MachineTypesSource)
			})
		}
	})
}

func TestTimeConversionEdgeCases(t *testing.T) {
	t.Run("should handle zero time values", func(t *testing.T) {
		zeroTime := time.Time{}
		sdkCluster := &k8sSDK.Cluster{
			ID:        "zero-time-test",
			Name:      "zero-time",
			Version:   "1.28.0",
			CreatedAt: &zeroTime,
			UpdatedAt: &zeroTime,
		}

		tfModel := convertSDKCreateResultToTerraformCreateClusterModel(sdkCluster)

		// Should handle zero time gracefully
		assert.False(t, tfModel.CreatedAt.IsNull())
		assert.False(t, tfModel.UpdatedAt.IsNull())
	})

	t.Run("should handle future time values", func(t *testing.T) {
		futureTime := time.Date(2030, 12, 31, 23, 59, 59, 0, time.UTC)
		sdkCluster := &k8sSDK.Cluster{
			ID:        "future-time-test",
			Name:      "future-time",
			Version:   "1.28.0",
			CreatedAt: &futureTime,
			UpdatedAt: &futureTime,
		}

		tfModel := convertSDKCreateResultToTerraformCreateClusterModel(sdkCluster)

		expectedTime := types.StringValue(futureTime.Format(time.RFC3339))
		assert.Equal(t, expectedTime, tfModel.CreatedAt)
		assert.Equal(t, expectedTime, tfModel.UpdatedAt)
	})
}
