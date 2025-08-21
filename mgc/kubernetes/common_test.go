package kubernetes

import (
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
