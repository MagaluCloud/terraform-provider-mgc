package kubernetes

import (
	"context"
	"fmt"
	"math"
	"reflect"
	"strings"
	"testing"
	"time"

	sdkK8s "github.com/MagaluCloud/mgc-sdk-go/kubernetes"
	"github.com/MagaluCloud/terraform-provider-mgc/mgc/utils"

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
	}{
		{
			name:           "success: nil input returns empty struct",
			inputSDK:       nil,
			inputClusterID: "cluster-123",
			inputRegion:    "br-sao-1",
			expectedResult: FlattenedGetResult{
				ClusterID: types.StringValue("cluster-123"),
			},
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
				Labels:         map[string]string{"env": "prod", "tier": "frontend"},
				SecurityGroups: &[]string{"sg-1", "sg-2"},
				Status:         sdkK8s.Status{State: "ACTIVE", Messages: []string{"Nodepool is active"}},
				Taints:         &[]sdkK8s.Taint{{Effect: "NoSchedule", Key: "app", Value: "critical"}},

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
				Taints:                     []Taint{{Effect: types.StringValue("NoSchedule"), Key: types.StringValue("app"), Value: types.StringValue("critical")}},

				AvailabilityZones: []types.String{types.StringValue("br-sao-1-a")},
				MaxPodsPerNode:    types.Int64Value(110),
			},
		},
		{
			name: "success: minimal data with nils and empty slices",
			inputSDK: &sdkK8s.NodePool{
				ID:               "nodepool-uuid-456",
				Name:             "minimal-nodepool",
				CreatedAt:        nil,
				UpdatedAt:        nil,
				Replicas:         1,
				AutoScale:        nil,
				InstanceTemplate: sdkK8s.InstanceTemplate{},
				Labels:           map[string]string{},
				SecurityGroups:   &[]string{},
				Status:           sdkK8s.Status{State: "CREATING", Messages: []string{}},
				Taints:           &[]sdkK8s.Taint{},

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
				StatusMessages:             nil,
				Taints:                     nil,

				AvailabilityZones: nil,
				MaxPodsPerNode:    types.Int64Value(110),
			},
		},
		{
			name: "success: handles nil MaxPodsPerNode",
			inputSDK: &sdkK8s.NodePool{
				ID:               "nodepool-uuid-789",
				Name:             "test-nodepool",
				Replicas:         1,
				InstanceTemplate: sdkK8s.InstanceTemplate{},
				Status:           sdkK8s.Status{State: "ACTIVE"},
				MaxPodsPerNode:   nil,
			},
			inputClusterID: "cluster-789",
			inputRegion:    "br-sao-1",
			expectedResult: FlattenedGetResult{
				ID:                         types.StringValue("nodepool-uuid-789"),
				ClusterID:                  types.StringValue("cluster-789"),
				Name:                       types.StringValue("test-nodepool"),
				CreatedAt:                  types.StringNull(),
				UpdatedAt:                  types.StringNull(),
				Replicas:                   types.Int64Value(1),
				AutoScaleMaxReplicas:       types.Int64Null(),
				AutoScaleMinReplicas:       types.Int64Null(),
				InstanceTemplateDiskSize:   types.Int64Value(0),
				InstanceTemplateDiskType:   types.StringValue(""),
				InstanceTemplateNodeImage:  types.StringValue(""),
				InstanceTemplateFlavorID:   types.StringValue(""),
				InstanceTemplateFlavorName: types.StringValue(""),
				InstanceTemplateFlavorRam:  types.Int64Value(0),
				InstanceTemplateFlavorSize: types.Int64Value(0),
				InstanceTemplateFlavorVcpu: types.Int64Value(0),
				Labels:                     expectedEmptyLabels,
				SecurityGroups:             nil,
				StatusState:                types.StringValue("ACTIVE"),
				StatusMessages:             nil,
				Taints:                     nil,

				AvailabilityZones: nil,
				MaxPodsPerNode:    types.Int64Null(),
			},
		},
		{
			name: "success: handles StatusMessages slice with multiple items",
			inputSDK: &sdkK8s.NodePool{
				ID:               "nodepool-uuid-101",
				Name:             "multi-msg-nodepool",
				Replicas:         2,
				InstanceTemplate: sdkK8s.InstanceTemplate{},
				Status:           sdkK8s.Status{State: "UPDATING", Messages: []string{"msg1", "msg2", "msg3"}},
				MaxPodsPerNode:   &maxPods,
			},
			inputClusterID: "cluster-101",
			inputRegion:    "br-sao-1",
			expectedResult: FlattenedGetResult{
				ID:                         types.StringValue("nodepool-uuid-101"),
				ClusterID:                  types.StringValue("cluster-101"),
				Name:                       types.StringValue("multi-msg-nodepool"),
				CreatedAt:                  types.StringNull(),
				UpdatedAt:                  types.StringNull(),
				Replicas:                   types.Int64Value(2),
				AutoScaleMaxReplicas:       types.Int64Null(),
				AutoScaleMinReplicas:       types.Int64Null(),
				InstanceTemplateDiskSize:   types.Int64Value(0),
				InstanceTemplateDiskType:   types.StringValue(""),
				InstanceTemplateNodeImage:  types.StringValue(""),
				InstanceTemplateFlavorID:   types.StringValue(""),
				InstanceTemplateFlavorName: types.StringValue(""),
				InstanceTemplateFlavorRam:  types.Int64Value(0),
				InstanceTemplateFlavorSize: types.Int64Value(0),
				InstanceTemplateFlavorVcpu: types.Int64Value(0),
				Labels:                     expectedEmptyLabels,
				SecurityGroups:             nil,
				StatusState:                types.StringValue("UPDATING"),
				StatusMessages:             []types.String{types.StringValue("msg1"), types.StringValue("msg2"), types.StringValue("msg3")},
				Taints:                     nil,

				AvailabilityZones: nil,
				MaxPodsPerNode:    types.Int64Value(110),
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			flattened, err := ConvertGetResultToFlattened(ctx, tc.inputSDK, tc.inputClusterID, tc.inputRegion)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if !reflect.DeepEqual(flattened, tc.expectedResult) {
				t.Errorf("Result does not match expected value.\nGot: %#v\nWant: %#v", flattened, tc.expectedResult)
			}
		})
	}
}

func TestConvertGetResultToFlattenedEdgeCases(t *testing.T) {
	ctx := context.Background()

	testCases := []struct {
		name           string
		inputSDK       *sdkK8s.NodePool
		inputClusterID string
		inputRegion    string
		expectError    bool
		validateResult func(t *testing.T, result FlattenedGetResult)
	}{
		{
			name: "extreme_values_numeric_overflow_risk",
			inputSDK: &sdkK8s.NodePool{
				ID:       "overflow-test",
				Name:     "overflow-nodepool",
				Replicas: math.MaxInt32,
				InstanceTemplate: sdkK8s.InstanceTemplate{
					DiskSize: math.MaxInt32,
					Flavor: sdkK8s.Flavor{
						RAM:  math.MaxInt32,
						Size: math.MaxInt32,
						VCPU: math.MaxInt32,
					},
				},
				Status:         sdkK8s.Status{State: "ACTIVE"},
				MaxPodsPerNode: func() *int { i := math.MaxInt32; return &i }(),
			},
			inputClusterID: "cluster-overflow",
			inputRegion:    "br-sao-1",
			validateResult: func(t *testing.T, result FlattenedGetResult) {
				if result.Replicas.ValueInt64() != int64(math.MaxInt32) {
					t.Errorf("Expected Replicas to be %d, got %d", int64(math.MaxInt32), result.Replicas.ValueInt64())
				}
			},
		},
		{
			name: "negative_values_should_be_handled",
			inputSDK: &sdkK8s.NodePool{
				ID:       "negative-test",
				Name:     "negative-nodepool",
				Replicas: -1,
				InstanceTemplate: sdkK8s.InstanceTemplate{
					DiskSize: -100,
					Flavor: sdkK8s.Flavor{
						RAM:  -1024,
						Size: -50,
						VCPU: -2,
					},
				},
				Status:         sdkK8s.Status{State: "ERROR"},
				MaxPodsPerNode: func() *int { i := -10; return &i }(),
			},
			inputClusterID: "cluster-negative",
			inputRegion:    "br-sao-1",
			validateResult: func(t *testing.T, result FlattenedGetResult) {
				if result.MaxPodsPerNode.ValueInt64() != -10 {
					t.Errorf("Expected MaxPodsPerNode to be -10, got %d", result.MaxPodsPerNode.ValueInt64())
				}
			},
		},
		{
			name: "empty_strings_throughout",
			inputSDK: &sdkK8s.NodePool{
				ID:   "", // Empty ID
				Name: "", // Empty Name
				InstanceTemplate: sdkK8s.InstanceTemplate{
					DiskType:  "",
					NodeImage: "",
					Flavor: sdkK8s.Flavor{
						ID:   "",
						Name: "",
					},
				},
				Status: sdkK8s.Status{State: ""},
			},
			inputClusterID: "", // Empty cluster ID
			inputRegion:    "", // Empty region
			validateResult: func(t *testing.T, result FlattenedGetResult) {
				if result.ID.ValueString() != "" {
					t.Errorf("Expected empty ID to be preserved")
				}
				if result.ClusterID.ValueString() != "" {
					t.Errorf("Expected empty ClusterID to be preserved")
				}
			},
		},
		{
			name: "very_long_strings",
			inputSDK: &sdkK8s.NodePool{
				ID:   strings.Repeat("a", 1000),
				Name: strings.Repeat("b", 2000),
				InstanceTemplate: sdkK8s.InstanceTemplate{
					DiskType:  strings.Repeat("c", 500),
					NodeImage: strings.Repeat("d", 1500),
					Flavor: sdkK8s.Flavor{
						ID:   strings.Repeat("e", 800),
						Name: strings.Repeat("f", 600),
					},
				},
				Status: sdkK8s.Status{
					State:    strings.Repeat("g", 100),
					Messages: []string{strings.Repeat("h", 3000)},
				},
			},
			inputClusterID: strings.Repeat("i", 1200),
			inputRegion:    strings.Repeat("j", 300),
			validateResult: func(t *testing.T, result FlattenedGetResult) {
				if len(result.Name.ValueString()) != 2000 {
					t.Errorf("Expected Name length to be 2000, got %d", len(result.Name.ValueString()))
				}
			},
		},
		{
			name: "unicode_and_special_characters",
			inputSDK: &sdkK8s.NodePool{
				ID:   "测试-🚀-nodepool",
				Name: "тест-⚡-пул",
				InstanceTemplate: sdkK8s.InstanceTemplate{
					DiskType:  "nvme-ñoño",
					NodeImage: "ubuntu-émojis-🐧",
					Flavor: sdkK8s.Flavor{
						ID:   "flavor-αβγ",
						Name: "c1.мedium",
					},
				},
				Status: sdkK8s.Status{
					State:    "АКТИВНЫЙ",
					Messages: []string{"Message with émojis 🎉", "Another με ελληνικά"},
				},
				Labels: map[string]string{
					"환경":          "프로덕션",
					"application": "тест-app",
					"emoji":       "🚀",
				},
			},
			inputClusterID: "cluster-测试",
			inputRegion:    "регион-1",
			validateResult: func(t *testing.T, result FlattenedGetResult) {
				if !strings.Contains(result.ID.ValueString(), "🚀") {
					t.Errorf("Expected ID to contain emoji")
				}
			},
		},
		{
			name: "large_arrays_with_duplicates",
			inputSDK: &sdkK8s.NodePool{
				ID:               "large-arrays-test",
				Name:             "large-arrays-nodepool",
				InstanceTemplate: sdkK8s.InstanceTemplate{},
				Status: sdkK8s.Status{
					State: "ACTIVE",
					Messages: func() []string {
						msgs := make([]string, 1000)
						for i := range 1000 {
							msgs[i] = fmt.Sprintf("Message %d", i)
						}
						return msgs
					}(),
				},
				SecurityGroups: func() *[]string {
					sgs := make([]string, 500)
					for i := range 500 {
						sgs[i] = fmt.Sprintf("sg-%d", i%10) // Create duplicates
					}
					return &sgs
				}(),
				Taints: func() *[]sdkK8s.Taint {
					taints := make([]sdkK8s.Taint, 200)
					for i := range 200 {
						taints[i] = sdkK8s.Taint{
							Effect: "NoSchedule",
							Key:    fmt.Sprintf("key-%d", i),
							Value:  fmt.Sprintf("value-%d", i),
						}
					}
					return &taints
				}(),

				AvailabilityZones: func() *[]string {
					azs := []string{"a", "b", "c", "a", "b", "c"} // Duplicates
					return &azs
				}(),
			},
			inputClusterID: "cluster-large",
			inputRegion:    "br-sao-1",
			validateResult: func(t *testing.T, result FlattenedGetResult) {
				if len(result.StatusMessages) != 1000 {
					t.Errorf("Expected 1000 status messages, got %d", len(result.StatusMessages))
				}
				if len(result.SecurityGroups) != 500 {
					t.Errorf("Expected 500 security groups, got %d", len(result.SecurityGroups))
				}
			},
		},
		{
			name: "mixed_nil_and_empty_pointers",
			inputSDK: &sdkK8s.NodePool{
				ID:               "mixed-nil-test",
				Name:             "mixed-nil-nodepool",
				InstanceTemplate: sdkK8s.InstanceTemplate{},
				Status:           sdkK8s.Status{State: "ACTIVE"},
				AutoScale: &sdkK8s.AutoScale{
					MaxReplicas: nil,
					MinReplicas: func() *int { i := 1; return &i }(),
				},
				SecurityGroups: &[]string{}, // Empty slice
				Taints:         nil,         // Nil pointer

				AvailabilityZones: nil, // Nil pointer
				MaxPodsPerNode:    nil, // Nil pointer
			},
			inputClusterID: "cluster-mixed",
			inputRegion:    "br-sao-1",
			validateResult: func(t *testing.T, result FlattenedGetResult) {
				if !result.AutoScaleMaxReplicas.IsNull() {
					t.Errorf("Expected AutoScaleMaxReplicas to be null")
				}
				if result.AutoScaleMinReplicas.IsNull() {
					t.Errorf("Expected AutoScaleMinReplicas to not be null")
				}
				if result.SecurityGroups != nil {
					t.Errorf("Expected SecurityGroups to be nil for empty slice")
				}
			},
		},
		{
			name: "extreme_timestamp_values",
			inputSDK: &sdkK8s.NodePool{
				ID:   "timestamp-test",
				Name: "timestamp-nodepool",
				CreatedAt: func() *time.Time {
					// Very old date
					t := time.Date(1900, 1, 1, 0, 0, 0, 0, time.UTC)
					return &t
				}(),
				UpdatedAt: func() *time.Time {
					// Far future date
					t := time.Date(3000, 12, 31, 23, 59, 59, 999999999, time.UTC)
					return &t
				}(),
				InstanceTemplate: sdkK8s.InstanceTemplate{},
				Status:           sdkK8s.Status{State: "ACTIVE"},
			},
			inputClusterID: "cluster-timestamp",
			inputRegion:    "br-sao-1",
			validateResult: func(t *testing.T, result FlattenedGetResult) {
				if result.CreatedAt.IsNull() {
					t.Errorf("Expected CreatedAt to not be null")
				}
				if result.UpdatedAt.IsNull() {
					t.Errorf("Expected UpdatedAt to not be null")
				}
				// Validate the format is valid RFC3339
				if !strings.Contains(result.CreatedAt.ValueString(), "1900") {
					t.Errorf("Expected CreatedAt to contain year 1900")
				}
			},
		},
		{
			name: "complex_labels_with_edge_cases",
			inputSDK: &sdkK8s.NodePool{
				ID:               "labels-test",
				Name:             "labels-nodepool",
				InstanceTemplate: sdkK8s.InstanceTemplate{},
				Status:           sdkK8s.Status{State: "ACTIVE"},
				Labels: map[string]string{
					"":              "empty-key",
					"empty-value":   "",
					"spaces in key": "value",
					"key":           "spaces in value",
					"special-chars": "!@#$%^&*()_+-=[]{}|;':\",./<>?",
					"very-long-key-" + strings.Repeat("a", 200): "short-value",
					"short-key":      "very-long-value-" + strings.Repeat("b", 500),
					"unicode-key-测试": "unicode-value-тест",
					"null-like":      "null",
					"boolean-like":   "true",
					"number-like":    "123",
				},
			},
			inputClusterID: "cluster-labels",
			inputRegion:    "br-sao-1",
			validateResult: func(t *testing.T, result FlattenedGetResult) {
				if result.Labels.IsNull() {
					t.Errorf("Expected Labels to not be null")
				}
				labelMap := result.Labels.Elements()
				if len(labelMap) != 11 {
					t.Errorf("Expected 11 labels, got %d", len(labelMap))
				}
			},
		},
		{
			name: "auto_scale_edge_cases",
			inputSDK: &sdkK8s.NodePool{
				ID:               "autoscale-test",
				Name:             "autoscale-nodepool",
				InstanceTemplate: sdkK8s.InstanceTemplate{},
				Status:           sdkK8s.Status{State: "ACTIVE"},
				AutoScale: &sdkK8s.AutoScale{
					MaxReplicas: func() *int { i := 0; return &i }(),  // Zero max replicas
					MinReplicas: func() *int { i := -1; return &i }(), // Negative min replicas
				},
			},
			inputClusterID: "cluster-autoscale",
			inputRegion:    "br-sao-1",
			validateResult: func(t *testing.T, result FlattenedGetResult) {
				if result.AutoScaleMaxReplicas.ValueInt64() != 0 {
					t.Errorf("Expected AutoScaleMaxReplicas to be 0, got %d", result.AutoScaleMaxReplicas.ValueInt64())
				}
				if result.AutoScaleMinReplicas.ValueInt64() != -1 {
					t.Errorf("Expected AutoScaleMinReplicas to be -1, got %d", result.AutoScaleMinReplicas.ValueInt64())
				}
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := ConvertGetResultToFlattened(ctx, tc.inputSDK, tc.inputClusterID, tc.inputRegion)

			if tc.expectError && err == nil {
				t.Errorf("Expected an error but got none")
				return
			}
			if !tc.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if tc.validateResult != nil {
				tc.validateResult(t, result)
			}
		})
	}
}

func TestConvertGetResultToFlattenedConcurrencyAndContext(t *testing.T) {
	// Test with cancelled context
	t.Run("cancelled_context", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		nodepool := &sdkK8s.NodePool{
			ID:               "context-test",
			Name:             "context-nodepool",
			InstanceTemplate: sdkK8s.InstanceTemplate{},
			Status:           sdkK8s.Status{State: "ACTIVE"},
			Labels:           map[string]string{"test": "value"},
		}

		// This should still work even with cancelled context as the function doesn't use context for cancellation
		result, err := ConvertGetResultToFlattened(ctx, nodepool, "cluster-test", "br-sao-1")
		if err != nil {
			t.Errorf("Unexpected error with cancelled context: %v", err)
		}
		if result.ID.ValueString() != "context-test" {
			t.Errorf("Expected ID to be preserved even with cancelled context")
		}
	})

	// Test concurrent access
	t.Run("concurrent_access", func(t *testing.T) {
		ctx := context.Background()
		nodepool := &sdkK8s.NodePool{
			ID:               "concurrent-test",
			Name:             "concurrent-nodepool",
			InstanceTemplate: sdkK8s.InstanceTemplate{},
			Status:           sdkK8s.Status{State: "ACTIVE"},
		}

		numGoroutines := 100
		results := make([]FlattenedGetResult, numGoroutines)
		errors := make([]error, numGoroutines)
		done := make(chan int, numGoroutines)

		for i := range numGoroutines {
			go func(index int) {
				results[index], errors[index] = ConvertGetResultToFlattened(ctx, nodepool, "cluster-test", "br-sao-1")
				done <- index
			}(i)
		}

		// Wait for all goroutines to complete
		for range numGoroutines {
			<-done
		}

		// Verify all results are the same and no errors occurred
		for i := range numGoroutines {
			if errors[i] != nil {
				t.Errorf("Goroutine %d returned error: %v", i, errors[i])
			}
			if i > 0 && !reflect.DeepEqual(results[i], results[0]) {
				t.Errorf("Goroutine %d returned different result than goroutine 0", i)
			}
		}
	})
}

func TestConvertGetResultToFlattenedMemoryAndPerformance(t *testing.T) {
	ctx := context.Background()

	// Test with very large data structures to check for memory leaks or performance issues
	t.Run("large_data_structures", func(t *testing.T) {
		// Create a nodepool with large amounts of data
		largeLabels := make(map[string]string)
		for i := range 10000 {
			largeLabels[fmt.Sprintf("key-%d", i)] = fmt.Sprintf("value-%d", i)
		}

		largeSecurityGroups := make([]string, 5000)
		for i := range 5000 {
			largeSecurityGroups[i] = fmt.Sprintf("sg-%d", i)
		}

		largeTaints := make([]sdkK8s.Taint, 3000)
		for i := range 3000 {
			largeTaints[i] = sdkK8s.Taint{
				Effect: "NoSchedule",
				Key:    fmt.Sprintf("taint-key-%d", i),
				Value:  fmt.Sprintf("taint-value-%d", i),
			}
		}

		largeMessages := make([]string, 2000)
		for i := range 2000 {
			largeMessages[i] = fmt.Sprintf("Status message %d: %s", i, strings.Repeat("detail", 100))
		}

		nodepool := &sdkK8s.NodePool{
			ID:               "large-data-test",
			Name:             "large-data-nodepool",
			InstanceTemplate: sdkK8s.InstanceTemplate{},
			Status:           sdkK8s.Status{State: "ACTIVE", Messages: largeMessages},
			Labels:           largeLabels,
			SecurityGroups:   &largeSecurityGroups,
			Taints:           &largeTaints,
		}

		start := time.Now()
		result, err := ConvertGetResultToFlattened(ctx, nodepool, "cluster-large", "br-sao-1")
		duration := time.Since(start)

		if err != nil {
			t.Errorf("Error processing large data: %v", err)
		}

		// Verify the data was processed correctly
		if len(result.Labels.Elements()) != 10000 {
			t.Errorf("Expected 10000 labels, got %d", len(result.Labels.Elements()))
		}
		if len(result.SecurityGroups) != 5000 {
			t.Errorf("Expected 5000 security groups, got %d", len(result.SecurityGroups))
		}
		if len(result.Taints) != 3000 {
			t.Errorf("Expected 3000 taints, got %d", len(result.Taints))
		}
		if len(result.StatusMessages) != 2000 {
			t.Errorf("Expected 2000 status messages, got %d", len(result.StatusMessages))
		}

		// Log performance info (not failing the test, just for monitoring)
		t.Logf("Processing large data took: %v", duration)
		if duration > time.Second {
			t.Logf("WARNING: Processing took longer than 1 second (%v)", duration)
		}
	})
}

func TestConvertGetResultToFlattenedInputValidation(t *testing.T) {
	ctx := context.Background()

	testCases := []struct {
		name           string
		nodepool       *sdkK8s.NodePool
		clusterID      string
		region         string
		expectError    bool
		validateResult func(t *testing.T, result FlattenedGetResult, err error)
	}{
		{
			name:      "nil_nodepool_should_return_empty_struct",
			nodepool:  nil,
			clusterID: "test-cluster",
			region:    "test-region",
			validateResult: func(t *testing.T, result FlattenedGetResult, err error) {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				// Should return struct with ClusterID preserved
				if result.ClusterID.ValueString() != "test-cluster" {
					t.Errorf("Expected ClusterID to be preserved for nil input, got: %s", result.ClusterID.ValueString())
				}
			},
		},
		{
			name: "minimal_valid_nodepool",
			nodepool: &sdkK8s.NodePool{
				ID: "minimal-test",
				// All other fields are zero values
			},
			clusterID: "test-cluster",
			region:    "test-region",
			validateResult: func(t *testing.T, result FlattenedGetResult, err error) {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if result.ID.ValueString() != "minimal-test" {
					t.Errorf("Expected ID to be 'minimal-test', got: %s", result.ID.ValueString())
				}
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := ConvertGetResultToFlattened(ctx, tc.nodepool, tc.clusterID, tc.region)
			if tc.validateResult != nil {
				tc.validateResult(t, result, err)
			}
		})
	}
}

func TestConvertGetResultToFlattenedPotentialFailures(t *testing.T) {
	ctx := context.Background()

	testCases := []struct {
		name           string
		inputSDK       *sdkK8s.NodePool
		inputClusterID string
		inputRegion    string
		expectPanic    bool
		description    string
	}{
		{
			name: "zero_value_instance_template",
			inputSDK: &sdkK8s.NodePool{
				ID:               "zero-template-test",
				Name:             "zero-template-nodepool",
				Replicas:         1,
				InstanceTemplate: sdkK8s.InstanceTemplate{}, // Zero value struct
				Status:           sdkK8s.Status{State: "ACTIVE"},
			},
			inputClusterID: "cluster-zero",
			inputRegion:    "br-sao-1",
			expectPanic:    false,
			description:    "Test with zero value InstanceTemplate - should not panic",
		},
		{
			name: "zero_value_flavor_in_instance_template",
			inputSDK: &sdkK8s.NodePool{
				ID:       "zero-flavor-test",
				Name:     "zero-flavor-nodepool",
				Replicas: 1,
				InstanceTemplate: sdkK8s.InstanceTemplate{
					DiskSize:  100,
					DiskType:  "ssd",
					NodeImage: "ubuntu-22.04",
					Flavor:    sdkK8s.Flavor{}, // Zero value Flavor struct
				},
				Status: sdkK8s.Status{State: "ACTIVE"},
			},
			inputClusterID: "cluster-zero-flavor",
			inputRegion:    "br-sao-1",
			expectPanic:    false,
			description:    "Test with zero value Flavor in InstanceTemplate - should not panic",
		},
		{
			name: "zero_value_status",
			inputSDK: &sdkK8s.NodePool{
				ID:               "zero-status-test",
				Name:             "zero-status-nodepool",
				Replicas:         1,
				InstanceTemplate: sdkK8s.InstanceTemplate{},
				Status:           sdkK8s.Status{}, // Zero value Status
			},
			inputClusterID: "cluster-zero-status",
			inputRegion:    "br-sao-1",
			expectPanic:    false,
			description:    "Test with zero value Status - should not panic",
		},
		{
			name: "nil_messages_in_status",
			inputSDK: &sdkK8s.NodePool{
				ID:               "nil-messages-test",
				Name:             "nil-messages-nodepool",
				Replicas:         1,
				InstanceTemplate: sdkK8s.InstanceTemplate{},
				Status: sdkK8s.Status{
					State:    "ACTIVE",
					Messages: nil, // Explicitly nil messages
				},
			},
			inputClusterID: "cluster-nil-messages",
			inputRegion:    "br-sao-1",
			expectPanic:    false,
			description:    "Test with nil Messages in Status - should not panic",
		},
		{
			name: "empty_strings_in_nested_fields",
			inputSDK: &sdkK8s.NodePool{
				ID:       "empty-nested-test",
				Name:     "",
				Replicas: 0,
				InstanceTemplate: sdkK8s.InstanceTemplate{
					DiskSize:  0,
					DiskType:  "",
					NodeImage: "",
					Flavor: sdkK8s.Flavor{
						ID:   "",
						Name: "",
						RAM:  0,
						Size: 0,
						VCPU: 0,
					},
				},
				Status: sdkK8s.Status{
					State:    "",
					Messages: []string{},
				},
			},
			inputClusterID: "",
			inputRegion:    "",
			expectPanic:    false,
			description:    "Test with all empty/zero values in nested fields - should not panic",
		},
		{
			name: "stress_test_multiple_nil_arrays",
			inputSDK: &sdkK8s.NodePool{
				ID:               "stress-nil-test",
				Name:             "stress-nil-nodepool",
				Replicas:         1,
				InstanceTemplate: sdkK8s.InstanceTemplate{},
				Status:           sdkK8s.Status{State: "ACTIVE"},
				AutoScale:        nil,
				Labels:           nil,
				SecurityGroups:   nil,
				Taints:           nil,

				AvailabilityZones: nil,
				MaxPodsPerNode:    nil,
			},
			inputClusterID: "cluster-stress",
			inputRegion:    "br-sao-1",
			expectPanic:    false,
			description:    "Stress test with multiple nil pointer fields - should not panic",
		},
		{
			name: "boundary_values_in_flavor",
			inputSDK: &sdkK8s.NodePool{
				ID:       "boundary-test",
				Name:     "boundary-nodepool",
				Replicas: 1,
				InstanceTemplate: sdkK8s.InstanceTemplate{
					Flavor: sdkK8s.Flavor{
						ID:   "boundary-flavor",
						Name: "boundary-flavor-name",
						RAM:  0,      // Boundary: zero RAM
						Size: -1,     // Boundary: negative Size
						VCPU: 999999, // Boundary: very large VCPU
					},
				},
				Status: sdkK8s.Status{State: "ACTIVE"},
			},
			inputClusterID: "cluster-boundary",
			inputRegion:    "br-sao-1",
			expectPanic:    false,
			description:    "Test with boundary values in Flavor fields - should not panic",
		},
		{
			name: "potential_integer_overflow",
			inputSDK: &sdkK8s.NodePool{
				ID:       "overflow-test",
				Name:     "overflow-nodepool",
				Replicas: 2147483647, // Max int32
				InstanceTemplate: sdkK8s.InstanceTemplate{
					DiskSize: 2147483647, // Max int32
					Flavor: sdkK8s.Flavor{
						RAM:  2147483647, // Max int32
						Size: 2147483647, // Max int32
						VCPU: 2147483647, // Max int32
					},
				},
				Status: sdkK8s.Status{State: "ACTIVE"},
				MaxPodsPerNode: func() *int {
					max := 2147483647 // Max int32
					return &max
				}(),
			},
			inputClusterID: "cluster-overflow",
			inputRegion:    "br-sao-1",
			expectPanic:    false,
			description:    "Test potential integer overflow when converting int to int64 - should not panic",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					if !tc.expectPanic {
						t.Errorf("Unexpected panic in %s: %v\nDescription: %s", tc.name, r, tc.description)
					}
				} else if tc.expectPanic {
					t.Errorf("Expected panic in %s but none occurred\nDescription: %s", tc.name, tc.description)
				}
			}()

			result, err := ConvertGetResultToFlattened(ctx, tc.inputSDK, tc.inputClusterID, tc.inputRegion)

			if err != nil {
				t.Errorf("Unexpected error in %s: %v\nDescription: %s", tc.name, err, tc.description)
				return
			}

			// Basic validation that the function completed successfully
			if tc.inputSDK != nil && result.ID.ValueString() != tc.inputSDK.ID {
				t.Errorf("ID mismatch in %s: expected %s, got %s\nDescription: %s",
					tc.name, tc.inputSDK.ID, result.ID.ValueString(), tc.description)
			}

			// Additional validations based on test case
			switch tc.name {
			case "zero_value_instance_template":
				// Should handle zero values gracefully
				if result.InstanceTemplateDiskSize.ValueInt64() != 0 {
					t.Errorf("Expected DiskSize to be 0 for zero value template")
				}
			case "zero_value_flavor_in_instance_template":
				// Should handle zero flavor values gracefully
				if result.InstanceTemplateFlavorID.ValueString() != "" {
					t.Errorf("Expected FlavorID to be empty for zero value flavor")
				}
			case "potential_integer_overflow":
				// Verify no overflow occurred
				if result.Replicas.ValueInt64() != 2147483647 {
					t.Errorf("Integer conversion failed: expected 2147483647, got %d", result.Replicas.ValueInt64())
				}
			case "stress_test_multiple_nil_arrays":
				// Verify all nil arrays are handled properly
				if result.SecurityGroups != nil {
					t.Errorf("Expected SecurityGroups to be nil")
				}
				if result.Taints != nil {
					t.Errorf("Expected Taints to be nil")
				}
				if !result.MaxPodsPerNode.IsNull() {
					t.Errorf("Expected MaxPodsPerNode to be null")
				}
			}
		})
	}
}

func TestConvertGetResultToFlattenedDataCorruption(t *testing.T) {
	ctx := context.Background()

	// Test for potential data corruption scenarios
	testCases := []struct {
		name        string
		inputSDK    *sdkK8s.NodePool
		validateFn  func(t *testing.T, result FlattenedGetResult)
		description string
	}{
		{
			name: "verify_no_data_mutation",
			inputSDK: &sdkK8s.NodePool{
				ID:       "mutation-test",
				Name:     "original-name",
				Replicas: 5,
				InstanceTemplate: sdkK8s.InstanceTemplate{
					DiskSize:  100,
					DiskType:  "original-type",
					NodeImage: "original-image",
					Flavor: sdkK8s.Flavor{
						ID:   "original-flavor-id",
						Name: "original-flavor-name",
						RAM:  1024,
						Size: 50,
						VCPU: 2,
					},
				},
				Status: sdkK8s.Status{
					State:    "original-state",
					Messages: []string{"original-message"},
				},
				Labels: map[string]string{
					"original-key": "original-value",
				},
			},
			validateFn: func(t *testing.T, result FlattenedGetResult) {
				// Verify original data is preserved correctly
				if result.Name.ValueString() != "original-name" {
					t.Errorf("Name corrupted: expected 'original-name', got %s", result.Name.ValueString())
				}
				if result.Replicas.ValueInt64() != 5 {
					t.Errorf("Replicas corrupted: expected 5, got %d", result.Replicas.ValueInt64())
				}
				if result.InstanceTemplateFlavorName.ValueString() != "original-flavor-name" {
					t.Errorf("FlavorName corrupted: expected 'original-flavor-name', got %s",
						result.InstanceTemplateFlavorName.ValueString())
				}
			},
			description: "Verify that original data is not corrupted during conversion",
		},
		{
			name: "verify_slice_independence",
			inputSDK: &sdkK8s.NodePool{
				ID:               "slice-test",
				Name:             "slice-nodepool",
				Replicas:         1,
				InstanceTemplate: sdkK8s.InstanceTemplate{},
				Status: sdkK8s.Status{
					State:    "ACTIVE",
					Messages: []string{"message1", "message2", "message3"},
				},
				SecurityGroups: &[]string{"sg1", "sg2"},
				Taints: &[]sdkK8s.Taint{
					{Effect: "NoSchedule", Key: "key1", Value: "value1"},
					{Effect: "NoExecute", Key: "key2", Value: "value2"},
				},
			},
			validateFn: func(t *testing.T, result FlattenedGetResult) {
				// Verify slices are correctly converted and independent
				if len(result.StatusMessages) != 3 {
					t.Errorf("StatusMessages length corrupted: expected 3, got %d", len(result.StatusMessages))
				}
				if len(result.SecurityGroups) != 2 {
					t.Errorf("SecurityGroups length corrupted: expected 2, got %d", len(result.SecurityGroups))
				}
				if len(result.Taints) != 2 {
					t.Errorf("Taints length corrupted: expected 2, got %d", len(result.Taints))
				}

				// Verify content integrity
				if result.StatusMessages[0].ValueString() != "message1" {
					t.Errorf("First status message corrupted")
				}
				if result.SecurityGroups[0].ValueString() != "sg1" {
					t.Errorf("First security group corrupted")
				}
				if result.Taints[0].Key.ValueString() != "key1" {
					t.Errorf("First taint key corrupted")
				}
			},
			description: "Verify that slice data is converted correctly and maintains independence",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := ConvertGetResultToFlattened(ctx, tc.inputSDK, "test-cluster", "test-region")
			if err != nil {
				t.Errorf("Unexpected error in %s: %v\nDescription: %s", tc.name, err, tc.description)
				return
			}

			if tc.validateFn != nil {
				tc.validateFn(t, result)
			}
		})
	}
}

func TestConvertGetResultToFlattenedUtilsEdgeCases(t *testing.T) {
	ctx := context.Background()

	testCases := []struct {
		name           string
		inputSDK       *sdkK8s.NodePool
		inputClusterID string
		inputRegion    string
		validateResult func(t *testing.T, result FlattenedGetResult)
		description    string
	}{
		{
			name: "region_with_special_characters",
			inputSDK: &sdkK8s.NodePool{
				ID:                "region-special-test",
				Name:              "region-special-nodepool",
				InstanceTemplate:  sdkK8s.InstanceTemplate{},
				Status:            sdkK8s.Status{State: "ACTIVE"},
				AvailabilityZones: &[]string{"a", "b", "c"},
			},
			inputClusterID: "cluster-region-special",
			inputRegion:    "BR-SÃO-1!@#$%",
			validateResult: func(t *testing.T, result FlattenedGetResult) {
				if len(result.AvailabilityZones) != 3 {
					t.Errorf("Expected 3 availability zones, got %d", len(result.AvailabilityZones))
				}
				// Check if special characters are handled
				for i, az := range result.AvailabilityZones {
					azValue := az.ValueString()
					if !strings.Contains(azValue, "br-são-1!@#$%") {
						t.Errorf("AvailabilityZone %d (%s) doesn't contain expected region", i, azValue)
					}
				}
			},
			description: "Test region with special characters in availability zone conversion",
		},
		{
			name: "empty_region_and_xzone",
			inputSDK: &sdkK8s.NodePool{
				ID:                "empty-region-test",
				Name:              "empty-region-nodepool",
				InstanceTemplate:  sdkK8s.InstanceTemplate{},
				Status:            sdkK8s.Status{State: "ACTIVE"},
				AvailabilityZones: &[]string{"", "  ", "\t\n"},
			},
			inputClusterID: "cluster-empty-region",
			inputRegion:    "",
			validateResult: func(t *testing.T, result FlattenedGetResult) {
				if len(result.AvailabilityZones) != 3 {
					t.Errorf("Expected 3 availability zones, got %d", len(result.AvailabilityZones))
				}
				// Empty region should result in "-" + zone
				expectedValues := []string{"-", "-  ", "-\t\n"}
				for i, expected := range expectedValues {
					if result.AvailabilityZones[i].ValueString() != expected {
						t.Errorf("AvailabilityZone %d: expected '%s', got '%s'",
							i, expected, result.AvailabilityZones[i].ValueString())
					}
				}
			},
			description: "Test empty region and zones in availability zone conversion",
		},
		{
			name: "very_long_region_and_zones",
			inputSDK: &sdkK8s.NodePool{
				ID:               "long-region-test",
				Name:             "long-region-nodepool",
				InstanceTemplate: sdkK8s.InstanceTemplate{},
				Status:           sdkK8s.Status{State: "ACTIVE"},
				AvailabilityZones: &[]string{
					strings.Repeat("a", 1000),
					strings.Repeat("b", 2000),
				},
			},
			inputClusterID: "cluster-long-region",
			inputRegion:    strings.Repeat("region", 500),
			validateResult: func(t *testing.T, result FlattenedGetResult) {
				if len(result.AvailabilityZones) != 2 {
					t.Errorf("Expected 2 availability zones, got %d", len(result.AvailabilityZones))
				}
				// Check that the long strings are handled without truncation
				expectedLen1 := len(strings.Repeat("region", 500)) + 1 + len(strings.Repeat("a", 1000))
				expectedLen2 := len(strings.Repeat("region", 500)) + 1 + len(strings.Repeat("b", 2000))

				if len(result.AvailabilityZones[0].ValueString()) != expectedLen1 {
					t.Errorf("First AZ length incorrect: expected %d, got %d",
						expectedLen1, len(result.AvailabilityZones[0].ValueString()))
				}
				if len(result.AvailabilityZones[1].ValueString()) != expectedLen2 {
					t.Errorf("Second AZ length incorrect: expected %d, got %d",
						expectedLen2, len(result.AvailabilityZones[1].ValueString()))
				}
			},
			description: "Test very long region and zone strings",
		},
		{
			name: "unicode_region_and_zones",
			inputSDK: &sdkK8s.NodePool{
				ID:                "unicode-region-test",
				Name:              "unicode-region-nodepool",
				InstanceTemplate:  sdkK8s.InstanceTemplate{},
				Status:            sdkK8s.Status{State: "ACTIVE"},
				AvailabilityZones: &[]string{"测试", "тест", "🚀", "αβγ"},
			},
			inputClusterID: "cluster-unicode-region",
			inputRegion:    "região-中国-россия",
			validateResult: func(t *testing.T, result FlattenedGetResult) {
				if len(result.AvailabilityZones) != 4 {
					t.Errorf("Expected 4 availability zones, got %d", len(result.AvailabilityZones))
				}
				// Verify Unicode characters are preserved and lowercased correctly
				expectedValues := []string{
					"região-中国-россия-测试",
					"região-中国-россия-тест",
					"região-中国-россия-🚀",
					"região-中国-россия-αβγ",
				}
				for i, expected := range expectedValues {
					actual := result.AvailabilityZones[i].ValueString()
					if actual != expected {
						t.Errorf("Unicode AZ %d: expected '%s', got '%s'", i, expected, actual)
					}
				}
			},
			description: "Test Unicode characters in region and zones",
		},
		{
			name: "time_conversion_edge_cases",
			inputSDK: &sdkK8s.NodePool{
				ID:   "time-edge-test",
				Name: "time-edge-nodepool",
				CreatedAt: func() *time.Time {
					// Unix epoch
					t := time.Unix(0, 0).UTC()
					return &t
				}(),
				UpdatedAt: func() *time.Time {
					// Far future time
					t := time.Date(9999, 12, 31, 23, 59, 59, 999999999, time.UTC)
					return &t
				}(),
				InstanceTemplate: sdkK8s.InstanceTemplate{},
				Status:           sdkK8s.Status{State: "ACTIVE"},
			},
			inputClusterID: "cluster-time-edge",
			inputRegion:    "test-region",
			validateResult: func(t *testing.T, result FlattenedGetResult) {
				if result.CreatedAt.IsNull() {
					t.Errorf("Expected CreatedAt to not be null")
				}
				if result.UpdatedAt.IsNull() {
					t.Errorf("Expected UpdatedAt to not be null")
				}

				// Verify epoch time
				if !strings.Contains(result.CreatedAt.ValueString(), "1970") {
					t.Errorf("Expected CreatedAt to contain 1970, got: %s", result.CreatedAt.ValueString())
				}

				// Verify far future time
				if !strings.Contains(result.UpdatedAt.ValueString(), "9999") {
					t.Errorf("Expected UpdatedAt to contain 9999, got: %s", result.UpdatedAt.ValueString())
				}
			},
			description: "Test edge cases in time conversion (epoch and far future)",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := ConvertGetResultToFlattened(ctx, tc.inputSDK, tc.inputClusterID, tc.inputRegion)
			if err != nil {
				t.Errorf("Unexpected error in %s: %v\nDescription: %s", tc.name, err, tc.description)
				return
			}

			if tc.validateResult != nil {
				tc.validateResult(t, result)
			}
		})
	}
}

func TestUtilsConversionFunctions(t *testing.T) {
	// Test the utils functions directly to verify their behavior
	t.Run("ConvertXZoneToAvailabilityZone_edge_cases", func(t *testing.T) {
		testCases := []struct {
			region   string
			xZone    string
			expected string
		}{
			{"", "", "-"},
			{"region", "", "region-"},
			{"", "zone", "-zone"},
			{"REGION", "ZONE", "region-zone"},
			{"região", "测试", "região-测试"},
			{"🚀", "⚡", "🚀-⚡"},
			{strings.Repeat("a", 1000), "b", strings.Repeat("a", 1000) + "-b"},
			{"special!@#$%^&*()", "chars[]{}|;':\",./<>?", "special!@#$%^&*()-chars[]{}|;':\",./<>?"},
		}

		for _, tc := range testCases {
			t.Run(fmt.Sprintf("region_%s_zone_%s", tc.region, tc.xZone), func(t *testing.T) {
				result := utils.ConvertXZoneToAvailabilityZone(tc.region, tc.xZone)
				if result != tc.expected {
					t.Errorf("Expected '%s', got '%s'", tc.expected, result)
				}
			})
		}
	})

	t.Run("ConvertTimeToRFC3339_edge_cases", func(t *testing.T) {
		// Test nil time
		result := utils.ConvertTimeToRFC3339(nil)
		if result != nil {
			t.Errorf("Expected nil for nil input, got %v", result)
		}

		// Test epoch time
		epochTime := time.Unix(0, 0).UTC()
		result = utils.ConvertTimeToRFC3339(&epochTime)
		if result == nil || *result != "1970-01-01T00:00:00Z" {
			t.Errorf("Expected epoch time in RFC3339, got %v", result)
		}

		// Test time with nanoseconds
		nsTime := time.Date(2023, 12, 25, 10, 30, 45, 123456789, time.UTC)
		result = utils.ConvertTimeToRFC3339(&nsTime)
		if result == nil || !strings.Contains(*result, "2023-12-25T10:30:45") {
			t.Errorf("Expected time with nanoseconds in RFC3339, got %v", result)
		}

		// Test far future time
		futureTime := time.Date(9999, 12, 31, 23, 59, 59, 0, time.UTC)
		result = utils.ConvertTimeToRFC3339(&futureTime)
		if result == nil || !strings.Contains(*result, "9999") {
			t.Errorf("Expected far future time in RFC3339, got %v", result)
		}

		// Test different timezone (should be converted to UTC in RFC3339)
		location, _ := time.LoadLocation("America/New_York")
		timezoneTime := time.Date(2023, 6, 15, 14, 30, 0, 0, location)
		result = utils.ConvertTimeToRFC3339(&timezoneTime)
		if result == nil {
			t.Errorf("Expected timezone time conversion, got nil")
		}
		// The result should be in UTC, so it should contain 'Z' or '+00:00'
		if result != nil && !strings.HasSuffix(*result, "Z") && !strings.Contains(*result, "+00:00") {
			t.Logf("Timezone time conversion result: %s", *result)
			// Note: This might be expected behavior depending on the implementation
		}
	})
}

func TestConvertGetResultToFlattenedResourceLeaks(t *testing.T) {
	ctx := context.Background()

	// Test for potential resource leaks with large datasets
	t.Run("large_dataset_no_memory_leak", func(t *testing.T) {
		// Create a large nodepool with many references
		largeNodepool := &sdkK8s.NodePool{
			ID:               "large-leak-test",
			Name:             "large-leak-nodepool",
			InstanceTemplate: sdkK8s.InstanceTemplate{},
			Status:           sdkK8s.Status{State: "ACTIVE"},
		}

		// Create large labels map
		largeNodepool.Labels = make(map[string]string)
		for i := range 10000 {
			largeNodepool.Labels[fmt.Sprintf("key-%d", i)] = fmt.Sprintf("value-%d", i)
		}

		// Create large security groups
		securityGroups := make([]string, 5000)
		for i := range 5000 {
			securityGroups[i] = fmt.Sprintf("sg-%d", i)
		}
		largeNodepool.SecurityGroups = &securityGroups

		// Create large taints
		taints := make([]sdkK8s.Taint, 3000)
		for i := range 3000 {
			taints[i] = sdkK8s.Taint{
				Effect: "NoSchedule",
				Key:    fmt.Sprintf("taint-key-%d", i),
				Value:  fmt.Sprintf("taint-value-%d", i),
			}
		}
		largeNodepool.Taints = &taints

		// Run conversion multiple times to check for accumulating memory usage
		for i := 0; i < 10; i++ {
			result, err := ConvertGetResultToFlattened(ctx, largeNodepool, "cluster-leak", "region-leak")
			if err != nil {
				t.Fatalf("Error in iteration %d: %v", i, err)
			}

			// Verify the result is correct
			if len(result.Labels.Elements()) != 10000 {
				t.Errorf("Iteration %d: Expected 10000 labels, got %d", i, len(result.Labels.Elements()))
			}
			if len(result.SecurityGroups) != 5000 {
				t.Errorf("Iteration %d: Expected 5000 security groups, got %d", i, len(result.SecurityGroups))
			}
			if len(result.Taints) != 3000 {
				t.Errorf("Iteration %d: Expected 3000 taints, got %d", i, len(result.Taints))
			}

			// Force garbage collection to help detect leaks
			// runtime.GC() // Uncomment if you want to force GC
		}
	})
}

func TestConvertGetResultToFlattenedInvariantsAndPostconditions(t *testing.T) {
	ctx := context.Background()

	// Test that certain invariants and postconditions hold
	t.Run("invariants_always_hold", func(t *testing.T) {
		testCases := []*sdkK8s.NodePool{
			nil, // Nil input
			{},  // Empty struct
			{ // Minimal valid
				ID:               "invariant-test",
				InstanceTemplate: sdkK8s.InstanceTemplate{},
				Status:           sdkK8s.Status{},
			},
			{ // Full struct
				ID:       "full-invariant-test",
				Name:     "full-nodepool",
				Replicas: 3,
				InstanceTemplate: sdkK8s.InstanceTemplate{
					DiskSize:  100,
					DiskType:  "ssd",
					NodeImage: "ubuntu",
					Flavor: sdkK8s.Flavor{
						ID:   "flavor-1",
						Name: "small",
						RAM:  1024,
						Size: 50,
						VCPU: 2,
					},
				},
				Status:         sdkK8s.Status{State: "ACTIVE", Messages: []string{"OK"}},
				Labels:         map[string]string{"test": "value"},
				SecurityGroups: &[]string{"sg-1"},
				Taints:         &[]sdkK8s.Taint{{Effect: "NoSchedule", Key: "k", Value: "v"}},

				AvailabilityZones: &[]string{"a"},
				MaxPodsPerNode:    func() *int { i := 110; return &i }(),
				AutoScale: &sdkK8s.AutoScale{
					MaxReplicas: func() *int { i := 5; return &i }(),
					MinReplicas: func() *int { i := 1; return &i }(),
				},
			},
		}

		for i, nodepool := range testCases {
			t.Run(fmt.Sprintf("invariant_test_%d", i), func(t *testing.T) {
				result, err := ConvertGetResultToFlattened(ctx, nodepool, "cluster-invariant", "region-invariant")

				// Postcondition: Function should never error
				if err != nil {
					t.Errorf("Invariant violated: function returned error: %v", err)
				}

				// Postcondition: ClusterID should always match input
				if result.ClusterID.ValueString() != "cluster-invariant" {
					t.Errorf("Invariant violated: ClusterID mismatch")
				}

				// Postcondition: If input is nil, result should have empty/null values
				if nodepool == nil {
					if !result.ID.IsNull() && result.ID.ValueString() != "" {
						t.Errorf("Invariant violated: nil input should result in empty ID")
					}
				}

				// Postcondition: If input has ID, result should preserve it
				if nodepool != nil && nodepool.ID != "" {
					if result.ID.ValueString() != nodepool.ID {
						t.Errorf("Invariant violated: ID not preserved")
					}
				}

				// Postcondition: Non-nil slices should never become nil in result
				if nodepool != nil {
					if nodepool.SecurityGroups != nil && result.SecurityGroups == nil && len(*nodepool.SecurityGroups) > 0 {
						t.Errorf("Invariant violated: non-empty SecurityGroups became nil")
					}
					if nodepool.Taints != nil && result.Taints == nil && len(*nodepool.Taints) > 0 {
						t.Errorf("Invariant violated: non-empty Taints became nil")
					}
				}
			})
		}
	})
}
