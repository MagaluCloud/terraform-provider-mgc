package resources

import (
	"context"
	"errors"
	"testing"
	"time"

	k8sSDK "github.com/MagaluCloud/mgc-sdk-go/kubernetes"
	"github.com/MagaluCloud/terraform-provider-mgc/mgc/tfutil"
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
		var input *[]tfutil.Taint
		result := convertTaintsNP(input)
		assert.Nil(t, result)
	})

	t.Run("should return empty slice for empty slice input", func(t *testing.T) {
		input := &[]tfutil.Taint{}
		result := convertTaintsNP(input)
		assert.NotNil(t, result)
		assert.Empty(t, *result)
	})

	t.Run("should convert slice with taints correctly", func(t *testing.T) {
		input := &[]tfutil.Taint{
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
