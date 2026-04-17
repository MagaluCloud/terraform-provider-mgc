package kubernetes

import (
	"testing"
	"time"

	k8sSDK "github.com/MagaluCloud/mgc-sdk-go/kubernetes"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
)

func TestConvertSDKCreateResultToTerraformCreateClusterModel(t *testing.T) {
	t.Run("should return nil for nil input", func(t *testing.T) {
		result := convertSDKCreateResultToTerraformCreateClsuterModel(nil)
		assert.Nil(t, result)
	})

	t.Run("should handle minimal SDK input correctly", func(t *testing.T) {
		sdkResult := &k8sSDK.Cluster{
			ID:      "cluster-123",
			Name:    "test-cluster",
			Version: "1.28.0",
		}

		result := convertSDKCreateResultToTerraformCreateClsuterModel(sdkResult)

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

		result := convertSDKCreateResultToTerraformCreateClsuterModel(sdkResult)
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

		result := convertSDKCreateResultToTerraformCreateClsuterModel(sdkResult)

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

		result := convertSDKCreateResultToTerraformCreateClsuterModel(sdkResult)

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

		result := convertSDKCreateResultToTerraformCreateClsuterModel(sdkResult)

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
		tfModel := convertSDKCreateResultToTerraformCreateClsuterModel(sdkCluster)

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

		tfModel := convertSDKCreateResultToTerraformCreateClsuterModel(sdkCluster)

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

				tfModel := convertSDKCreateResultToTerraformCreateClsuterModel(sdkCluster)
				assert.Equal(t, types.StringValue(tc.expected), tfModel.MachineTypesSource)
			})
		}
	})
}

func TestBuildPatchClusterRequest(t *testing.T) {
	t.Run("should request version upgrade when plan version differs from state version", func(t *testing.T) {
		state := KubernetesClusterCreateResourceModel{
			Version:      types.StringValue("v1.28.0"),
			AllowedCidrs: []types.String{types.StringValue("10.0.0.0/8")},
		}
		plan := KubernetesClusterCreateResourceModel{
			Version:      types.StringValue("v1.29.0"),
			AllowedCidrs: []types.String{types.StringValue("10.0.0.0/8")},
		}

		patch, versionChanged := buildPatchClusterRequest(state, plan)

		assert.True(t, versionChanged, "version change must be reported so Update can poll until running")
		assert.NotNil(t, patch.Version)
		assert.Equal(t, "v1.29.0", *patch.Version)
	})

	t.Run("should not include version in patch when plan version equals state version", func(t *testing.T) {
		state := KubernetesClusterCreateResourceModel{
			Version: types.StringValue("v1.28.0"),
		}
		plan := KubernetesClusterCreateResourceModel{
			Version: types.StringValue("v1.28.0"),
		}

		patch, versionChanged := buildPatchClusterRequest(state, plan)

		assert.False(t, versionChanged)
		assert.Nil(t, patch.Version, "no upgrade should be triggered when version did not change")
	})

	t.Run("should not include version in patch when plan version is unknown", func(t *testing.T) {
		state := KubernetesClusterCreateResourceModel{
			Version: types.StringValue("v1.28.0"),
		}
		plan := KubernetesClusterCreateResourceModel{
			Version: types.StringUnknown(),
		}

		patch, versionChanged := buildPatchClusterRequest(state, plan)

		assert.False(t, versionChanged)
		assert.Nil(t, patch.Version)
	})

	t.Run("should not include version in patch when plan version is null", func(t *testing.T) {
		state := KubernetesClusterCreateResourceModel{
			Version: types.StringValue("v1.28.0"),
		}
		plan := KubernetesClusterCreateResourceModel{
			Version: types.StringNull(),
		}

		patch, versionChanged := buildPatchClusterRequest(state, plan)

		assert.False(t, versionChanged)
		assert.Nil(t, patch.Version)
	})

	t.Run("should always carry allowed_cidrs from plan", func(t *testing.T) {
		state := KubernetesClusterCreateResourceModel{
			Version:      types.StringValue("v1.28.0"),
			AllowedCidrs: []types.String{types.StringValue("10.0.0.0/8")},
		}
		plan := KubernetesClusterCreateResourceModel{
			Version: types.StringValue("v1.28.0"),
			AllowedCidrs: []types.String{
				types.StringValue("192.168.0.0/16"),
				types.StringValue("172.16.0.0/12"),
			},
		}

		patch, versionChanged := buildPatchClusterRequest(state, plan)

		assert.False(t, versionChanged)
		assert.NotNil(t, patch.AllowedCIDRs)
		assert.Equal(t, []string{"192.168.0.0/16", "172.16.0.0/12"}, *patch.AllowedCIDRs)
	})

	t.Run("should combine version upgrade and allowed_cidrs change in the same patch", func(t *testing.T) {
		state := KubernetesClusterCreateResourceModel{
			Version:      types.StringValue("v1.28.0"),
			AllowedCidrs: []types.String{types.StringValue("10.0.0.0/8")},
		}
		plan := KubernetesClusterCreateResourceModel{
			Version:      types.StringValue("v1.30.1"),
			AllowedCidrs: []types.String{types.StringValue("192.168.0.0/16")},
		}

		patch, versionChanged := buildPatchClusterRequest(state, plan)

		assert.True(t, versionChanged)
		assert.NotNil(t, patch.Version)
		assert.Equal(t, "v1.30.1", *patch.Version)
		assert.NotNil(t, patch.AllowedCIDRs)
		assert.Equal(t, []string{"192.168.0.0/16"}, *patch.AllowedCIDRs)
	})

	t.Run("should carry empty allowed_cidrs slice when plan clears the list", func(t *testing.T) {
		state := KubernetesClusterCreateResourceModel{
			Version:      types.StringValue("v1.28.0"),
			AllowedCidrs: []types.String{types.StringValue("10.0.0.0/8")},
		}
		plan := KubernetesClusterCreateResourceModel{
			Version:      types.StringValue("v1.28.0"),
			AllowedCidrs: nil,
		}

		patch, _ := buildPatchClusterRequest(state, plan)

		assert.NotNil(t, patch.AllowedCIDRs)
		assert.Empty(t, *patch.AllowedCIDRs)
	})
}

func TestClusterNetworkFrom(t *testing.T) {
	t.Run("a cluster without an explicit network falls back to the default VPC subnets", func(t *testing.T) {
		request := clusterNetworkFrom(nil)

		assert.Nil(t, request, "no network payload should be sent so the API uses the default VPC")
	})

	t.Run("a cluster's network is described by the subnet IDs it should run on", func(t *testing.T) {
		network := &ClusterNetwork{
			SubnetIDs: types.SetValueMust(types.StringType, []attr.Value{
				types.StringValue("11111111-1111-1111-1111-111111111111"),
				types.StringValue("22222222-2222-2222-2222-222222222222"),
			}),
		}

		request := clusterNetworkFrom(network)

		assert.NotNil(t, request)
		assert.ElementsMatch(t, []string{
			"11111111-1111-1111-1111-111111111111",
			"22222222-2222-2222-2222-222222222222",
		}, request.SubnetIDs)
	})

	t.Run("the subnet IDs form an unordered set where duplicates are not allowed", func(t *testing.T) {
		network := &ClusterNetwork{
			SubnetIDs: types.SetValueMust(types.StringType, []attr.Value{
				types.StringValue("subnet-c"),
				types.StringValue("subnet-a"),
				types.StringValue("subnet-b"),
			}),
		}

		request := clusterNetworkFrom(network)

		assert.ElementsMatch(t, []string{"subnet-a", "subnet-b", "subnet-c"}, request.SubnetIDs)
	})
}

func TestConvertSDKClusterReadsNetworkSubnetIDs(t *testing.T) {
	t.Run("a cluster's network subnets are exposed as the set of their IDs", func(t *testing.T) {
		sdkResult := &k8sSDK.Cluster{
			ID:      "cluster-with-network",
			Name:    "with-network",
			Version: "1.29.0",
			Network: &k8sSDK.Network{
				VPCID: "vpc-xyz",
				Subnets: []k8sSDK.Subnet{
					{ID: "subnet-a", CIDR: "172.18.0.0/20", AvailabilityZone: "a"},
					{ID: "subnet-b", CIDR: "172.18.16.0/20", AvailabilityZone: "b"},
				},
			},
		}

		tfModel := convertSDKCreateResultToTerraformCreateClsuterModel(sdkResult)

		assert.NotNil(t, tfModel.Network, "the network block must be populated when the API returns subnets")
		ids := []string{}
		for _, e := range tfModel.Network.SubnetIDs.Elements() {
			ids = append(ids, e.(types.String).ValueString())
		}
		assert.ElementsMatch(t, []string{"subnet-a", "subnet-b"}, ids)
	})

	t.Run("a cluster without network information leaves the network block unset", func(t *testing.T) {
		sdkResult := &k8sSDK.Cluster{
			ID:      "cluster-no-network",
			Name:    "no-network",
			Version: "1.29.0",
		}

		tfModel := convertSDKCreateResultToTerraformCreateClsuterModel(sdkResult)

		assert.Nil(t, tfModel.Network)
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

		tfModel := convertSDKCreateResultToTerraformCreateClsuterModel(sdkCluster)

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

		tfModel := convertSDKCreateResultToTerraformCreateClsuterModel(sdkCluster)

		expectedTime := types.StringValue(futureTime.Format(time.RFC3339))
		assert.Equal(t, expectedTime, tfModel.CreatedAt)
		assert.Equal(t, expectedTime, tfModel.UpdatedAt)
	})
}
