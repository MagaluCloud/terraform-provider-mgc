package kubernetes

import (
	"context"
	"testing"
	"time"

	k8sSDK "github.com/MagaluCloud/mgc-sdk-go/kubernetes"
	"github.com/MagaluCloud/terraform-provider-mgc/mgc/utils"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
)

func pointer[T any](v T) *T {
	return &v
}

func TestConvertToNodePoolToTFModel(t *testing.T) {
	now := time.Now()
	nowStr := utils.ConvertTimeToRFC3339(&now)

	testCases := []struct {
		name     string
		region   string
		input    *k8sSDK.NodePool
		expected NodePool
	}{
		{
			name:     "should return empty struct when input is nil",
			region:   "br-sao-1",
			input:    nil,
			expected: NodePool{},
		},
		{
			name:   "should convert a fully populated node pool",
			region: "br-sao-1",
			input: &k8sSDK.NodePool{
				ID:        "np-12345",
				Name:      "my-full-nodepool",
				Replicas:  3,
				CreatedAt: &now,
				UpdatedAt: &now,
				AutoScale: &k8sSDK.AutoScale{
					MaxReplicas: pointer(5),
					MinReplicas: pointer(2),
				},
				InstanceTemplate: k8sSDK.InstanceTemplate{
					Flavor: k8sSDK.Flavor{Name: "c1.medium"},
				},
				Taints: &[]k8sSDK.Taint{
					{Key: "app", Value: "blue", Effect: "NoSchedule"},
					{Key: "storage", Value: "ssd", Effect: "NoExecute"},
				},
				MaxPodsPerNode:    pointer(110),
				AvailabilityZones: &[]string{"a", "b"},
			},
			expected: NodePool{
				ID:          types.StringValue("np-12345"),
				Name:        types.StringValue("my-full-nodepool"),
				Flavor:      types.StringValue("c1.medium"),
				Replicas:    types.Int64Value(3),
				MaxReplicas: types.Int64Value(5),
				MinReplicas: types.Int64Value(2),
				CreatedAt:   types.StringPointerValue(nowStr),
				UpdatedAt:   types.StringPointerValue(nowStr),
				Taints: &[]Taint{
					{Key: types.StringValue("app"), Value: types.StringValue("blue"), Effect: types.StringValue("NoSchedule")},
					{Key: types.StringValue("storage"), Value: types.StringValue("ssd"), Effect: types.StringValue("NoExecute")},
				},
				MaxPodsPerNode: types.Int64Value(110),
				AvailabilityZones: &[]types.String{
					types.StringValue("br-sao-1-a"),
					types.StringValue("br-sao-1-b"),
				},
			},
		},
		{
			name:   "should handle partially populated node pool with nil values",
			region: "us-east-1",
			input: &k8sSDK.NodePool{
				ID:        "np-67890",
				Name:      "my-partial-nodepool",
				Replicas:  1,
				CreatedAt: &now,
				UpdatedAt: nil,
				AutoScale: &k8sSDK.AutoScale{
					MaxReplicas: nil,
					MinReplicas: pointer(1),
				},
				InstanceTemplate: k8sSDK.InstanceTemplate{
					Flavor: k8sSDK.Flavor{Name: "t2.micro"},
				},
				Taints:            nil,
				MaxPodsPerNode:    nil,
				AvailabilityZones: nil,
			},
			expected: NodePool{
				ID:                types.StringValue("np-67890"),
				Name:              types.StringValue("my-partial-nodepool"),
				Flavor:            types.StringValue("t2.micro"),
				Replicas:          types.Int64Value(1),
				MinReplicas:       types.Int64Value(1),
				CreatedAt:         types.StringPointerValue(nowStr),
				UpdatedAt:         types.StringNull(),
				MaxReplicas:       types.Int64Null(),
				Taints:            nil,
				MaxPodsPerNode:    types.Int64Null(),
				AvailabilityZones: nil,
			},
		},
		{
			name:   "should handle empty slices for taints and availability zones",
			region: "br-sao-1",
			input: &k8sSDK.NodePool{
				ID:                "np-abcde",
				Name:              "empty-slice-pool",
				Replicas:          1,
				Taints:            &[]k8sSDK.Taint{},
				AvailabilityZones: &[]string{},
			},
			expected: NodePool{
				ID:                types.StringValue("np-abcde"),
				Name:              types.StringValue("empty-slice-pool"),
				Replicas:          types.Int64Value(1),
				Flavor:            types.StringNull(),
				CreatedAt:         types.StringNull(),
				UpdatedAt:         types.StringNull(),
				MaxReplicas:       types.Int64Null(),
				MinReplicas:       types.Int64Null(),
				MaxPodsPerNode:    types.Int64Null(),
				Taints:            &[]Taint{},
				AvailabilityZones: &[]types.String{},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := ConvertToNodePoolToTFModel(tc.input, tc.region)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestConvertToNodePoolToTFModelNewFields(t *testing.T) {
	now := time.Now()
	nowStr := utils.ConvertTimeToRFC3339(&now)

	testCases := []struct {
		name     string
		region   string
		input    *k8sSDK.NodePool
		expected NodePool
	}{
		{
			name:   "should convert labels and security groups correctly",
			region: "br-se1",
			input: &k8sSDK.NodePool{
				ID:       "np-labels-sg",
				Name:     "labels-sg-pool",
				Replicas: 2,
				Labels: map[string]string{
					"env":  "production",
					"tier": "frontend",
					"app":  "web",
				},
				SecurityGroups: &[]string{"sg-web", "sg-app", "sg-db"},
				InstanceTemplate: k8sSDK.InstanceTemplate{
					Flavor: k8sSDK.Flavor{Name: "c1.large"},
				},
				CreatedAt: &now,
			},
			expected: NodePool{
				ID:                types.StringValue("np-labels-sg"),
				Name:              types.StringValue("labels-sg-pool"),
				Flavor:            types.StringValue("c1.large"),
				Replicas:          types.Int64Value(2),
				CreatedAt:         types.StringPointerValue(nowStr),
				UpdatedAt:         types.StringNull(),
				MaxReplicas:       types.Int64Null(),
				MinReplicas:       types.Int64Null(),
				MaxPodsPerNode:    types.Int64Null(),
				Taints:            nil,
				AvailabilityZones: nil,
			},
		},
		{
			name:   "should handle empty labels and security groups",
			region: "us-west-2",
			input: &k8sSDK.NodePool{
				ID:             "np-empty-fields",
				Name:           "empty-fields-pool",
				Replicas:       1,
				Labels:         map[string]string{},
				SecurityGroups: &[]string{},
				InstanceTemplate: k8sSDK.InstanceTemplate{
					Flavor: k8sSDK.Flavor{Name: "t2.small"},
				},
			},
			expected: NodePool{
				ID:                types.StringValue("np-empty-fields"),
				Name:              types.StringValue("empty-fields-pool"),
				Flavor:            types.StringValue("t2.small"),
				Replicas:          types.Int64Value(1),
				CreatedAt:         types.StringNull(),
				UpdatedAt:         types.StringNull(),
				MaxReplicas:       types.Int64Null(),
				MinReplicas:       types.Int64Null(),
				MaxPodsPerNode:    types.Int64Null(),
				Taints:            nil,
				AvailabilityZones: nil,
			},
		},
		{
			name:   "should handle nil labels and security groups",
			region: "eu-west-1",
			input: &k8sSDK.NodePool{
				ID:             "np-nil-fields",
				Name:           "nil-fields-pool",
				Replicas:       3,
				Labels:         nil,
				SecurityGroups: nil,
				InstanceTemplate: k8sSDK.InstanceTemplate{
					Flavor: k8sSDK.Flavor{Name: "m5.xlarge"},
				},
			},
			expected: NodePool{
				ID:                types.StringValue("np-nil-fields"),
				Name:              types.StringValue("nil-fields-pool"),
				Flavor:            types.StringValue("m5.xlarge"),
				Replicas:          types.Int64Value(3),
				CreatedAt:         types.StringNull(),
				UpdatedAt:         types.StringNull(),
				MaxReplicas:       types.Int64Null(),
				MinReplicas:       types.Int64Null(),
				MaxPodsPerNode:    types.Int64Null(),
				Taints:            nil,
				AvailabilityZones: nil,
			},
		},
		{
			name:   "should use instance template flavor and ignore direct flavor field",
			region: "ap-south-1",
			input: &k8sSDK.NodePool{
				ID:       "np-flavor-template",
				Name:     "flavor-template-pool",
				Replicas: 1,
				Flavor:   "direct-flavor-name",
				InstanceTemplate: k8sSDK.InstanceTemplate{
					Flavor: k8sSDK.Flavor{Name: "template-flavor-name"},
				},
			},
			expected: NodePool{
				ID:                types.StringValue("np-flavor-template"),
				Name:              types.StringValue("flavor-template-pool"),
				Flavor:            types.StringValue("template-flavor-name"),
				Replicas:          types.Int64Value(1),
				CreatedAt:         types.StringNull(),
				UpdatedAt:         types.StringNull(),
				MaxReplicas:       types.Int64Null(),
				MinReplicas:       types.Int64Null(),
				MaxPodsPerNode:    types.Int64Null(),
				Taints:            nil,
				AvailabilityZones: nil,
			},
		},
		{
			name:   "should use instance template flavor when available",
			region: "ca-central-1",
			input: &k8sSDK.NodePool{
				ID:       "np-flavor-template-only",
				Name:     "flavor-template-only-pool",
				Replicas: 1,
				Flavor:   "",
				InstanceTemplate: k8sSDK.InstanceTemplate{
					Flavor: k8sSDK.Flavor{Name: "template-flavor"},
				},
			},
			expected: NodePool{
				ID:                types.StringValue("np-flavor-template-only"),
				Name:              types.StringValue("flavor-template-only-pool"),
				Flavor:            types.StringValue("template-flavor"),
				Replicas:          types.Int64Value(1),
				CreatedAt:         types.StringNull(),
				UpdatedAt:         types.StringNull(),
				MaxReplicas:       types.Int64Null(),
				MinReplicas:       types.Int64Null(),
				MaxPodsPerNode:    types.Int64Null(),
				Taints:            nil,
				AvailabilityZones: nil,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := ConvertToNodePoolToTFModel(tc.input, tc.region)

			// Basic field assertions
			assert.Equal(t, tc.expected.ID, result.ID)
			assert.Equal(t, tc.expected.Name, result.Name)
			assert.Equal(t, tc.expected.Flavor, result.Flavor)
			assert.Equal(t, tc.expected.Replicas, result.Replicas)
			assert.Equal(t, tc.expected.CreatedAt, result.CreatedAt)
			assert.Equal(t, tc.expected.UpdatedAt, result.UpdatedAt)
			assert.Equal(t, tc.expected.MaxReplicas, result.MaxReplicas)
			assert.Equal(t, tc.expected.MinReplicas, result.MinReplicas)
			assert.Equal(t, tc.expected.MaxPodsPerNode, result.MaxPodsPerNode)
			assert.Equal(t, tc.expected.Taints, result.Taints)
			assert.Equal(t, tc.expected.AvailabilityZones, result.AvailabilityZones)

			// Validate labels conversion
			if len(tc.input.Labels) > 0 {
				assert.False(t, result.Labels.IsNull())
				var resultLabels map[string]string
				diags := result.Labels.ElementsAs(context.Background(), &resultLabels, false)
				assert.False(t, diags.HasError())
				assert.Equal(t, tc.input.Labels, resultLabels)
			} else {
				// Labels should be null or empty map
				if !result.Labels.IsNull() {
					var resultLabels map[string]string
					diags := result.Labels.ElementsAs(context.Background(), &resultLabels, false)
					assert.False(t, diags.HasError())
					assert.Empty(t, resultLabels)
				}
			}

			// Validate security groups conversion
			if tc.input.SecurityGroups != nil && len(*tc.input.SecurityGroups) > 0 {
				assert.False(t, result.SecurityGroups.IsNull())
				var resultSGs []string
				diags := result.SecurityGroups.ElementsAs(context.Background(), &resultSGs, false)
				assert.False(t, diags.HasError())
				assert.Equal(t, *tc.input.SecurityGroups, resultSGs)
			} else {
				// SecurityGroups should be null or empty
				if !result.SecurityGroups.IsNull() {
					var resultSGs []string
					diags := result.SecurityGroups.ElementsAs(context.Background(), &resultSGs, false)
					assert.False(t, diags.HasError())
					assert.Empty(t, resultSGs)
				}
			}
		})
	}
}

func TestConvertToNodePoolToTFModelEdgeCases(t *testing.T) {
	t.Run("should handle complex labels with special characters", func(t *testing.T) {
		input := &k8sSDK.NodePool{
			ID:       "np-complex",
			Name:     "complex-pool",
			Replicas: 1,
			Labels: map[string]string{
				"app.kubernetes.io/name":    "myapp",
				"app.kubernetes.io/version": "1.0.0",
				"environment":               "prod",
				"special-chars":             "value!@#$%^&*()",
				"unicode":                   "测试",
				"empty-value":               "",
			},
		}

		result := ConvertToNodePoolToTFModel(input, "br-se1")

		assert.False(t, result.Labels.IsNull())
		var resultLabels map[string]string
		diags := result.Labels.ElementsAs(context.Background(), &resultLabels, false)
		assert.False(t, diags.HasError())
		assert.Equal(t, input.Labels, resultLabels)
	})

	t.Run("should_handle_security_groups_correctly", func(t *testing.T) {
		securityGroups := []string{"sg-1", "sg-2", "sg-3"}
		input := &k8sSDK.NodePool{
			ID:             "np-sg-test",
			Name:           "sg-test-pool",
			Replicas:       1,
			SecurityGroups: &securityGroups,
			InstanceTemplate: k8sSDK.InstanceTemplate{
				Flavor: k8sSDK.Flavor{Name: "test-flavor"},
			},
		}

		result := ConvertToNodePoolToTFModel(input, "us-east-1")

		assert.False(t, result.SecurityGroups.IsNull())
		var resultSGs []string
		diags := result.SecurityGroups.ElementsAs(context.Background(), &resultSGs, false)
		assert.False(t, diags.HasError())

		// Sets don't preserve order or duplicates, so check that all unique values are present
		assert.Len(t, resultSGs, 3)
		assert.Contains(t, resultSGs, "sg-1")
		assert.Contains(t, resultSGs, "sg-2")
		assert.Contains(t, resultSGs, "sg-3")
	})

	t.Run("should_handle_nil_maps_correctly", func(t *testing.T) {
		input := &k8sSDK.NodePool{
			ID:       "np-nil-map",
			Name:     "nil-map-pool",
			Replicas: 1,
			Labels:   nil, // nil map should be handled gracefully
			InstanceTemplate: k8sSDK.InstanceTemplate{
				Flavor: k8sSDK.Flavor{Name: "test-flavor"},
			},
		}

		result := ConvertToNodePoolToTFModel(input, "us-east-1")

		// For nil maps, len() returns 0, so Labels should be null or empty
		if !result.Labels.IsNull() {
			var resultLabels map[string]string
			diags := result.Labels.ElementsAs(context.Background(), &resultLabels, false)
			assert.False(t, diags.HasError())
			assert.Empty(t, resultLabels)
		}
	})

}
