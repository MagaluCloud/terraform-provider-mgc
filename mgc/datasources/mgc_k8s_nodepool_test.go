package datasources

import (
	"context"
	"reflect"
	"testing"
	"time"

	sdkK8s "github.com/MagaluCloud/mgc-sdk-go/kubernetes"

	"github.com/MagaluCloud/terraform-provider-mgc/mgc/tfutil"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

func TestConvertGetResultToFlattened(t *testing.T) {
	ctx := context.Background()
	testTime := time.Date(2025, 8, 11, 14, 56, 0, 0, time.UTC)
	testTimeString := testTime.Format(time.RFC3339)

	maxPods := 110
	minReplicas := 1
	maxReplicas := 5

	expectedLabels, _ := types.MapValueFrom(ctx, types.StringType, map[string]string{"env": "prod", "tier": "frontend"})
	expectedEmptyLabels, _ := types.MapValueFrom(ctx, types.StringType, map[string]string{})

	testCases := []struct {
		name           string
		inputSDK       *sdkK8s.NodePool
		inputClusterID string
		inputRegion    string
		expectedResult FlattenedGetResult
		expectPanic    bool
	}{
		{
			name:           "success: nil input returns empty struct",
			inputSDK:       nil,
			inputClusterID: "cluster-123",
			inputRegion:    "br-sao-1",
			expectedResult: FlattenedGetResult{},
		},
		{
			name: "success: full data mapping",
			inputSDK: &sdkK8s.NodePool{
				ID:        "nodepool-uuid-123",
				Name:      "my-nodepool",
				CreatedAt: &testTime,
				UpdatedAt: &testTime,
				Replicas:  3,
				AutoScale: &sdkK8s.AutoScale{
					MinReplicas: &minReplicas,
					MaxReplicas: &maxReplicas,
				},
				InstanceTemplate: sdkK8s.InstanceTemplate{
					DiskSize:  50,
					DiskType:  "premium",
					NodeImage: "ubuntu-22.04",
					Flavor: sdkK8s.Flavor{
						ID:   "flavor-id-abc",
						Name: "c1.medium",
						RAM:  4096,
						Size: 50,
						VCPU: 2,
					},
				},
				Labels:            map[string]string{"env": "prod", "tier": "frontend"},
				SecurityGroups:    &[]string{"sg-1", "sg-2"},
				Status:            sdkK8s.Status{State: "ACTIVE", Messages: []string{"Nodepool is active"}},
				Taints:            &[]sdkK8s.Taint{{Effect: "NoSchedule", Key: "app", Value: "critical"}},
				Zone:              &[]string{"zone-a"},
				AvailabilityZones: &[]string{"a"},
				MaxPodsPerNode:    &maxPods,
			},
			inputClusterID: "cluster-abc",
			inputRegion:    "br-sao-1",
			expectedResult: FlattenedGetResult{
				ID:                         types.StringValue("nodepool-uuid-123"),
				ClusterID:                  types.StringValue("cluster-abc"),
				Name:                       types.StringValue("my-nodepool"),
				CreatedAt:                  types.StringValue(testTimeString),
				UpdatedAt:                  types.StringValue(testTimeString),
				Replicas:                   types.Int64Value(3),
				AutoScaleMaxReplicas:       types.Int64Value(5),
				AutoScaleMinReplicas:       types.Int64Value(1),
				InstanceTemplateDiskSize:   types.Int64Value(50),
				InstanceTemplateDiskType:   types.StringValue("premium"),
				InstanceTemplateNodeImage:  types.StringValue("ubuntu-22.04"),
				InstanceTemplateFlavorID:   types.StringValue("flavor-id-abc"),
				InstanceTemplateFlavorName: types.StringValue("c1.medium"),
				InstanceTemplateFlavorRam:  types.Int64Value(4096),
				InstanceTemplateFlavorSize: types.Int64Value(50),
				InstanceTemplateFlavorVcpu: types.Int64Value(2),
				Labels:                     expectedLabels,
				SecurityGroups:             []types.String{types.StringValue("sg-1"), types.StringValue("sg-2")},
				StatusState:                types.StringValue("ACTIVE"),
				StatusMessages:             []types.String{types.StringValue("Nodepool is active")},
				Taints:                     []tfutil.Taint{{Effect: types.StringValue("NoSchedule"), Key: types.StringValue("app"), Value: types.StringValue("critical")}},
				Zone:                       []types.String{types.StringValue("zone-a")},
				AvailabilityZones:          []types.String{types.StringValue("br-sao-1-a")},
				MaxPodsPerNode:             types.Int64Value(110),
			},
		},
		{
			name: "success: minimal data with nils and empty slices",
			inputSDK: &sdkK8s.NodePool{
				ID:                "nodepool-uuid-456",
				Name:              "minimal-nodepool",
				CreatedAt:         nil,
				UpdatedAt:         nil,
				Replicas:          1,
				AutoScale:         nil,
				InstanceTemplate:  sdkK8s.InstanceTemplate{},
				Labels:            map[string]string{},
				SecurityGroups:    &[]string{},
				Status:            sdkK8s.Status{State: "CREATING", Messages: []string{}},
				Taints:            &[]sdkK8s.Taint{},
				Zone:              &[]string{},
				AvailabilityZones: &[]string{},
				MaxPodsPerNode:    &maxPods,
			},
			inputClusterID: "cluster-def",
			inputRegion:    "br-sao-1",
			expectedResult: FlattenedGetResult{
				ID:                         types.StringValue("nodepool-uuid-456"),
				ClusterID:                  types.StringValue("cluster-def"),
				Name:                       types.StringValue("minimal-nodepool"),
				CreatedAt:                  types.StringNull(),
				UpdatedAt:                  types.StringNull(),
				Replicas:                   types.Int64Value(1),
				AutoScaleMaxReplicas:       types.Int64Null(),
				AutoScaleMinReplicas:       types.Int64Null(),
				InstanceTemplateDiskSize:   types.Int64Value(0),
				InstanceTemplateDiskType:   basetypes.NewStringValue(""),
				InstanceTemplateNodeImage:  basetypes.NewStringValue(""),
				InstanceTemplateFlavorID:   basetypes.NewStringValue(""),
				InstanceTemplateFlavorName: basetypes.NewStringValue(""),
				InstanceTemplateFlavorRam:  types.Int64Value(0),
				InstanceTemplateFlavorSize: types.Int64Value(0),
				InstanceTemplateFlavorVcpu: types.Int64Value(0),
				Labels:                     expectedEmptyLabels,
				SecurityGroups:             nil,
				StatusState:                types.StringValue("CREATING"),
				StatusMessages:             []types.String{types.StringNull()},
				Taints:                     nil,
				Zone:                       nil,
				AvailabilityZones:          nil,
				MaxPodsPerNode:             types.Int64Value(110),
			},
		},
		{
			name: "BUG: panics on nil MaxPodsPerNode",
			inputSDK: &sdkK8s.NodePool{
				ID:             "nodepool-uuid-789",
				MaxPodsPerNode: nil,
			},
			expectPanic: true,
		},
		{
			name: "BUG: panics on StatusMessages slice with more than 1 item",
			inputSDK: &sdkK8s.NodePool{
				ID:             "nodepool-uuid-101",
				Status:         sdkK8s.Status{Messages: []string{"msg1", "msg2"}},
				MaxPodsPerNode: &maxPods,
			},
			expectPanic: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			defer func() {
				r := recover()
				if tc.expectPanic {
					if r == nil {
						t.Errorf("Expected a panic, but did not get one")
					}
				} else {
					if r != nil {
						t.Errorf("Did not expect a panic, but got one: %v", r)
					}
				}
			}()

			flattened, err := ConvertGetResultToFlattened(ctx, tc.inputSDK, tc.inputClusterID, tc.inputRegion)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if tc.expectPanic {
				return
			}

			if !reflect.DeepEqual(flattened, tc.expectedResult) {
				t.Errorf("Result does not match expected value.\nGot: %#v\nWant: %#v", flattened, tc.expectedResult)
			}
		})
	}
}
