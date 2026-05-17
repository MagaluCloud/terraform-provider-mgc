package kubernetes

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	k8sSDK "github.com/MagaluCloud/mgc-sdk-go/kubernetes"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/stretchr/testify/assert"
)

type mockNodePoolService struct {
	k8sSDK.NodePoolService
	GetFunc func(ctx context.Context, clusterID, nodepoolID string) (*k8sSDK.NodePool, error)
}

func (m *mockNodePoolService) Get(ctx context.Context, clusterID, nodepoolID string) (*k8sSDK.NodePool, error) {
	if m.GetFunc != nil {
		return m.GetFunc(ctx, clusterID, nodepoolID)
	}
	return nil, errors.New("GetFunc is not implemented")
}

func TestWaitNodePoolState(t *testing.T) {
	ctx := context.Background()
	testTimeout := 100 * time.Millisecond
	testInterval := 10 * time.Millisecond

	t.Run("should return nil when nodepool is in desired state on first check", func(t *testing.T) {
		mockSvc := &mockNodePoolService{
			GetFunc: func(ctx context.Context, clusterID, nodepoolID string) (*k8sSDK.NodePool, error) {
				return &k8sSDK.NodePool{
					Status: k8sSDK.Status{State: NodepoolRunningState},
				}, nil
			},
		}
		r := &NewNodePoolResource{sdkNodepool: mockSvc}

		err := r.waitNodePoolState(ctx, "np-id", "cluster-id", NodepoolRunningState, testTimeout, testInterval)
		assert.NoError(t, err)
	})

	t.Run("should return nil after a few retries", func(t *testing.T) {
		callCount := 0
		mockSvc := &mockNodePoolService{
			GetFunc: func(ctx context.Context, clusterID, nodepoolID string) (*k8sSDK.NodePool, error) {
				callCount++
				if callCount < 3 {
					return &k8sSDK.NodePool{
						Status: k8sSDK.Status{State: "Provisioning"},
					}, nil
				}
				return &k8sSDK.NodePool{
					Status: k8sSDK.Status{State: NodepoolRunningState},
				}, nil
			},
		}
		r := &NewNodePoolResource{sdkNodepool: mockSvc}

		err := r.waitNodePoolState(ctx, "np-id", "cluster-id", NodepoolRunningState, testTimeout, testInterval)
		assert.NoError(t, err)
	})

	t.Run("should return timeout error when state is never reached", func(t *testing.T) {
		mockSvc := &mockNodePoolService{
			GetFunc: func(ctx context.Context, clusterID, nodepoolID string) (*k8sSDK.NodePool, error) {
				return &k8sSDK.NodePool{
					Status: k8sSDK.Status{State: "Provisioning"},
				}, nil
			},
		}
		r := &NewNodePoolResource{sdkNodepool: mockSvc}

		err := r.waitNodePoolState(ctx, "np-id", "cluster-id", NodepoolRunningState, testTimeout, testInterval)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "timeout waiting for node pool creation")
	})

	t.Run("should return error from the SDK get call", func(t *testing.T) {
		expectedErr := errors.New("sdk error")
		mockSvc := &mockNodePoolService{
			GetFunc: func(ctx context.Context, clusterID, nodepoolID string) (*k8sSDK.NodePool, error) {
				return nil, expectedErr
			},
		}
		r := &NewNodePoolResource{sdkNodepool: mockSvc}

		err := r.waitNodePoolState(ctx, "np-id", "cluster-id", NodepoolRunningState, testTimeout, testInterval)
		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)
	})
}

func TestConvertStringArrayTFToSliceString(t *testing.T) {
	t.Run("should return nil for nil input", func(t *testing.T) {
		var input *[]types.String
		result := convertStringArrayTFToSliceString(input)
		assert.Nil(t, result)
	})

	t.Run("should return empty slice for empty slice input", func(t *testing.T) {
		input := &[]types.String{}
		result := convertStringArrayTFToSliceString(input)
		assert.NotNil(t, result)
		assert.Empty(t, *result)
	})

	t.Run("should convert slice with values correctly", func(t *testing.T) {
		input := &[]types.String{
			types.StringValue("az-1"),
			types.StringValue("az-2"),
			types.StringValue("az-3"),
		}
		expected := &[]string{"az-1", "az-2", "az-3"}
		result := convertStringArrayTFToSliceString(input)
		assert.Equal(t, expected, result)
	})
}

func TestConvertTaintsNP(t *testing.T) {
	t.Run("should return nil for nil input", func(t *testing.T) {
		var input *[]Taint
		result := convertTaintsNP(input)
		assert.Nil(t, result)
	})

	t.Run("should return empty slice for empty slice input", func(t *testing.T) {
		input := &[]Taint{}
		result := convertTaintsNP(input)
		assert.NotNil(t, result)
		assert.Empty(t, *result)
	})

	t.Run("should convert slice with taints correctly", func(t *testing.T) {
		input := &[]Taint{
			{
				Effect: types.StringValue("NoSchedule"),
				Key:    types.StringValue("key1"),
				Value:  types.StringValue("value1"),
			},
			{
				Effect: types.StringValue("PreferNoSchedule"),
				Key:    types.StringValue("key2"),
				Value:  types.StringValue("value2"),
			},
		}
		expected := &[]k8sSDK.Taint{
			{
				Effect: "NoSchedule",
				Key:    "key1",
				Value:  "value1",
			},
			{
				Effect: "PreferNoSchedule",
				Key:    "key2",
				Value:  "value2",
			},
		}
		result := convertTaintsNP(input)
		assert.Equal(t, expected, result)
	})
}

func TestGetSubnetIDs(t *testing.T) {
	t.Run("a network's subnets are exposed as the set of their IDs", func(t *testing.T) {
		network := &k8sSDK.Network{
			VPCID: "vpc-xyz",
			Subnets: []k8sSDK.Subnet{
				{ID: "subnet-a", CIDR: "172.18.0.0/20", AvailabilityZone: "a"},
				{ID: "subnet-b", CIDR: "172.18.16.0/20", AvailabilityZone: "b"},
			},
		}

		result := GetSubnetIDs(network)

		assert.False(t, result.IsNull(), "subnet_ids must be populated when the API returns subnets")
		ids := []string{}
		for _, e := range result.Elements() {
			ids = append(ids, e.(types.String).ValueString())
		}
		assert.ElementsMatch(t, []string{"subnet-a", "subnet-b"}, ids)
	})

	t.Run("a nil network returns a null set so the schema reports the attribute as absent", func(t *testing.T) {
		result := GetSubnetIDs(nil)

		assert.True(t, result.IsNull())
	})

	t.Run("a network with no subnets returns a null set", func(t *testing.T) {
		result := GetSubnetIDs(&k8sSDK.Network{VPCID: "vpc-xyz"})

		assert.True(t, result.IsNull())
	})
}

func TestBuildPatchNodePoolRequest(t *testing.T) {
	t.Run("a node pool's autoscale bounds are carried in the patch when set in the plan", func(t *testing.T) {
		plan := NodePoolResourceModel{NodePool: NodePool{
			Replicas:    types.Int64Value(2),
			MaxReplicas: types.Int64Value(5),
			MinReplicas: types.Int64Value(1),
		}}

		patch := buildPatchNodePoolRequest(plan)

		assert.NotNil(t, patch.AutoScale)
		assert.NotNil(t, patch.AutoScale.MaxReplicas)
		assert.Equal(t, 5, *patch.AutoScale.MaxReplicas)
		assert.NotNil(t, patch.AutoScale.MinReplicas)
		assert.Equal(t, 1, *patch.AutoScale.MinReplicas)
	})
}

func TestConvertToNodePoolReadsVersion(t *testing.T) {
	t.Run("a node pool's current Kubernetes version is exposed when the API returns it", func(t *testing.T) {
		version := "v1.30.1"
		sdkPool := &k8sSDK.NodePool{
			ID:      "np-id",
			Name:    "worker",
			Version: &version,
		}

		converted := ConvertToNodePoolToTFModel(sdkPool, "br-se1")

		assert.Equal(t, types.StringValue("v1.30.1"), converted.Version)
	})

	t.Run("a node pool without version information leaves the version field null", func(t *testing.T) {
		sdkPool := &k8sSDK.NodePool{
			ID:      "np-id",
			Name:    "worker",
			Version: nil,
		}

		converted := ConvertToNodePoolToTFModel(sdkPool, "br-se1")

		assert.True(t, converted.Version.IsNull())
	})
}

func TestNodePoolImportState(t *testing.T) {
	ctx := context.Background()
	r := &NewNodePoolResource{}

	schemaResp := &resource.SchemaResponse{}
	r.Schema(ctx, resource.SchemaRequest{}, schemaResp)
	npSchema := schemaResp.Schema

	newImportResp := func() *resource.ImportStateResponse {
		return &resource.ImportStateResponse{
			State: tfsdk.State{
				Schema: npSchema,
				Raw:    tftypes.NewValue(npSchema.Type().TerraformType(ctx), nil),
			},
		}
	}

	t.Run("a node pool import id missing the comma separator is rejected with an explanatory error", func(t *testing.T) {
		resp := newImportResp()
		r.ImportState(ctx, resource.ImportStateRequest{ID: "single-id-no-comma"}, resp)

		assert.True(t, resp.Diagnostics.HasError(), "import id without comma must be rejected")
		assert.Contains(t, resp.Diagnostics.Errors()[0].Detail(), "cluster_id,node_pool_id",
			"the error must teach the user the expected import id format")
	})

	t.Run("a valid 'cluster_id,node_pool_id' import id puts both values into state", func(t *testing.T) {
		resp := newImportResp()
		r.ImportState(ctx, resource.ImportStateRequest{ID: "cluster-uuid,np-uuid"}, resp)

		assert.False(t, resp.Diagnostics.HasError(), "valid import must not error: %v", resp.Diagnostics)

		var clusterID types.String
		resp.State.GetAttribute(ctx, path.Root("cluster_id"), &clusterID)
		assert.Equal(t, types.StringValue("cluster-uuid"), clusterID)

		var npID types.String
		resp.State.GetAttribute(ctx, path.Root("id"), &npID)
		assert.Equal(t, types.StringValue("np-uuid"), npID)
	})
}

func TestNodePoolSchemaMaxPodsPerNodePlanModifiers(t *testing.T) {
	r := &NewNodePoolResource{}
	resp := &resource.SchemaResponse{}

	r.Schema(context.Background(), resource.SchemaRequest{}, resp)

	t.Run("should include max_pods_per_node in schema", func(t *testing.T) {
		_, ok := resp.Schema.Attributes["max_pods_per_node"]
		assert.True(t, ok)
	})

	t.Run("should define max_pods_per_node as int64 attribute", func(t *testing.T) {
		attrRaw, ok := resp.Schema.Attributes["max_pods_per_node"]
		assert.True(t, ok)

		_, ok = attrRaw.(schema.Int64Attribute)
		assert.True(t, ok)
	})

	t.Run("should keep requires replace and add use state for unknown plan modifiers", func(t *testing.T) {
		attrRaw, ok := resp.Schema.Attributes["max_pods_per_node"]
		assert.True(t, ok)

		attr, ok := attrRaw.(schema.Int64Attribute)
		assert.True(t, ok)

		hasRequiresReplace := false
		hasUseStateForUnknown := false

		for _, modifier := range attr.PlanModifiers {
			if reflect.TypeOf(modifier) == reflect.TypeOf(int64planmodifier.RequiresReplace()) {
				hasRequiresReplace = true
			}
			if reflect.TypeOf(modifier) == reflect.TypeOf(int64planmodifier.UseStateForUnknown()) {
				hasUseStateForUnknown = true
			}
		}

		assert.True(t, hasRequiresReplace)
		assert.True(t, hasUseStateForUnknown)
	})
}
