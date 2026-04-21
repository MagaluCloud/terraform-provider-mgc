package kubernetes

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	k8sSDK "github.com/MagaluCloud/mgc-sdk-go/kubernetes"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
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

func TestNodePoolNetworkFrom(t *testing.T) {
	t.Run("a node pool without an explicit network inherits the subnets chosen for the cluster", func(t *testing.T) {
		request := nodePoolNetworkFrom(nil)

		assert.Nil(t, request, "no network payload should be sent so the pool inherits the cluster network")
	})

	t.Run("a node pool's network is described by the subnet IDs its nodes should run on", func(t *testing.T) {
		network := &NodePoolNetwork{
			SubnetIDs: types.SetValueMust(types.StringType, []attr.Value{
				types.StringValue("11111111-1111-1111-1111-111111111111"),
				types.StringValue("22222222-2222-2222-2222-222222222222"),
			}),
		}

		request := nodePoolNetworkFrom(network)

		assert.NotNil(t, request)
		assert.ElementsMatch(t, []string{
			"11111111-1111-1111-1111-111111111111",
			"22222222-2222-2222-2222-222222222222",
		}, request.SubnetIDs)
	})

	t.Run("the subnet IDs form an unordered set where duplicates are not allowed", func(t *testing.T) {
		network := &NodePoolNetwork{
			SubnetIDs: types.SetValueMust(types.StringType, []attr.Value{
				types.StringValue("subnet-c"),
				types.StringValue("subnet-a"),
				types.StringValue("subnet-b"),
			}),
		}

		request := nodePoolNetworkFrom(network)

		assert.ElementsMatch(t, []string{"subnet-a", "subnet-b", "subnet-c"}, request.SubnetIDs)
	})
}

func TestConvertNodePoolReadsNetworkSubnetIDs(t *testing.T) {
	t.Run("a node pool's network subnets are exposed as the set of their IDs", func(t *testing.T) {
		sdkPool := &k8sSDK.NodePool{
			Network: &k8sSDK.Network{
				VPCID: "vpc-xyz",
				Subnets: []k8sSDK.Subnet{
					{ID: "subnet-a", CIDR: "172.18.0.0/20", AvailabilityZone: "a"},
					{ID: "subnet-b", CIDR: "172.18.16.0/20", AvailabilityZone: "b"},
				},
			},
		}

		network := nodePoolNetworkFromSDK(sdkPool.Network)

		assert.NotNil(t, network, "the network block must be populated when the API returns subnets")
		ids := []string{}
		for _, e := range network.SubnetIDs.Elements() {
			ids = append(ids, e.(types.String).ValueString())
		}
		assert.ElementsMatch(t, []string{"subnet-a", "subnet-b"}, ids)
	})

	t.Run("a node pool without network information leaves the network block unset", func(t *testing.T) {
		network := nodePoolNetworkFromSDK(nil)

		assert.Nil(t, network)
	})
}

func TestBuildPatchNodePoolRequest(t *testing.T) {
	t.Run("a node pool's Kubernetes version is upgraded in place when the plan version differs from the state version", func(t *testing.T) {
		state := NodePoolResourceModel{NodePool: NodePool{
			Version:  types.StringValue("v1.28.0"),
			Replicas: types.Int64Value(2),
		}}
		plan := NodePoolResourceModel{NodePool: NodePool{
			Version:  types.StringValue("v1.29.0"),
			Replicas: types.Int64Value(2),
		}}

		patch, versionChanged := buildPatchNodePoolRequest(state, plan)

		assert.True(t, versionChanged, "version change must be reported so Update can poll until running")
		assert.NotNil(t, patch.Version)
		assert.Equal(t, "v1.29.0", *patch.Version)
	})

	t.Run("a node pool keeps its current version when the plan version equals the state version", func(t *testing.T) {
		state := NodePoolResourceModel{NodePool: NodePool{
			Version:  types.StringValue("v1.28.0"),
			Replicas: types.Int64Value(2),
		}}
		plan := NodePoolResourceModel{NodePool: NodePool{
			Version:  types.StringValue("v1.28.0"),
			Replicas: types.Int64Value(2),
		}}

		patch, versionChanged := buildPatchNodePoolRequest(state, plan)

		assert.False(t, versionChanged)
		assert.Nil(t, patch.Version, "no upgrade should be triggered when version did not change")
	})

	t.Run("a node pool keeps its current version when the plan version is unknown", func(t *testing.T) {
		state := NodePoolResourceModel{NodePool: NodePool{
			Version:  types.StringValue("v1.28.0"),
			Replicas: types.Int64Value(2),
		}}
		plan := NodePoolResourceModel{NodePool: NodePool{
			Version:  types.StringUnknown(),
			Replicas: types.Int64Value(2),
		}}

		patch, versionChanged := buildPatchNodePoolRequest(state, plan)

		assert.False(t, versionChanged)
		assert.Nil(t, patch.Version)
	})

	t.Run("a node pool keeps its current version when the plan version is null", func(t *testing.T) {
		state := NodePoolResourceModel{NodePool: NodePool{
			Version:  types.StringValue("v1.28.0"),
			Replicas: types.Int64Value(2),
		}}
		plan := NodePoolResourceModel{NodePool: NodePool{
			Version:  types.StringNull(),
			Replicas: types.Int64Value(2),
		}}

		patch, versionChanged := buildPatchNodePoolRequest(state, plan)

		assert.False(t, versionChanged)
		assert.Nil(t, patch.Version)
	})

	t.Run("a node pool's replicas are always carried from the plan into the patch", func(t *testing.T) {
		state := NodePoolResourceModel{NodePool: NodePool{
			Version:  types.StringValue("v1.28.0"),
			Replicas: types.Int64Value(2),
		}}
		plan := NodePoolResourceModel{NodePool: NodePool{
			Version:  types.StringValue("v1.28.0"),
			Replicas: types.Int64Value(5),
		}}

		patch, _ := buildPatchNodePoolRequest(state, plan)

		assert.NotNil(t, patch.Replicas)
		assert.Equal(t, 5, *patch.Replicas)
	})

	t.Run("a node pool can have its version and replicas updated together in the same patch", func(t *testing.T) {
		state := NodePoolResourceModel{NodePool: NodePool{
			Version:  types.StringValue("v1.28.0"),
			Replicas: types.Int64Value(2),
		}}
		plan := NodePoolResourceModel{NodePool: NodePool{
			Version:  types.StringValue("v1.30.1"),
			Replicas: types.Int64Value(4),
		}}

		patch, versionChanged := buildPatchNodePoolRequest(state, plan)

		assert.True(t, versionChanged)
		assert.NotNil(t, patch.Version)
		assert.Equal(t, "v1.30.1", *patch.Version)
		assert.NotNil(t, patch.Replicas)
		assert.Equal(t, 4, *patch.Replicas)
	})

	t.Run("a node pool's autoscale bounds are carried in the patch when set in the plan", func(t *testing.T) {
		state := NodePoolResourceModel{NodePool: NodePool{
			Version:  types.StringValue("v1.28.0"),
			Replicas: types.Int64Value(2),
		}}
		plan := NodePoolResourceModel{NodePool: NodePool{
			Version:     types.StringValue("v1.28.0"),
			Replicas:    types.Int64Value(2),
			MaxReplicas: types.Int64Value(5),
			MinReplicas: types.Int64Value(1),
		}}

		patch, _ := buildPatchNodePoolRequest(state, plan)

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

func TestNodePoolSchemaVersionPlanModifiers(t *testing.T) {
	r := &NewNodePoolResource{}
	resp := &resource.SchemaResponse{}

	r.Schema(context.Background(), resource.SchemaRequest{}, resp)

	t.Run("the version attribute is exposed so users can declare the node pool's Kubernetes version", func(t *testing.T) {
		_, ok := resp.Schema.Attributes["version"]
		assert.True(t, ok, "version attribute must be present on the schema")
	})

	t.Run("the version attribute is optional and computed so the provider can echo back the server value", func(t *testing.T) {
		attrRaw, ok := resp.Schema.Attributes["version"].(schema.StringAttribute)
		assert.True(t, ok)
		assert.True(t, attrRaw.Optional)
		assert.True(t, attrRaw.Computed)
	})

	t.Run("changing the version must trigger an in-place upgrade, never a replacement of the node pool", func(t *testing.T) {
		attrRaw, ok := resp.Schema.Attributes["version"].(schema.StringAttribute)
		assert.True(t, ok)

		hasUseStateForUnknown := false
		hasRequiresReplace := false
		for _, modifier := range attrRaw.PlanModifiers {
			if reflect.TypeOf(modifier) == reflect.TypeOf(stringplanmodifier.UseStateForUnknown()) {
				hasUseStateForUnknown = true
			}
			if reflect.TypeOf(modifier) == reflect.TypeOf(stringplanmodifier.RequiresReplace()) {
				hasRequiresReplace = true
			}
		}

		assert.True(t, hasUseStateForUnknown, "version must use state for unknown to avoid spurious diffs on read")
		assert.False(t, hasRequiresReplace, "version change is upgraded in place, not by recreating the node pool")
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
