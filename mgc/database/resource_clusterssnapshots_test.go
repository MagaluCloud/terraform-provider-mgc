package database

import (
	"context"
	"errors"
	"testing"
	"time"

	dbSDK "github.com/MagaluCloud/mgc-sdk-go/dbaas"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type mockClusterServiceForSnapshots struct {
	mock.Mock
	dbSDK.ClusterService
}

func (m *mockClusterServiceForSnapshots) GetSnapshot(ctx context.Context, clusterID, snapshotID string) (*dbSDK.ClusterSnapshotDetailResponse, error) {
	args := m.Called(ctx, clusterID, snapshotID)
	result := args.Get(0)
	if result == nil {
		return nil, args.Error(1)
	}
	return result.(*dbSDK.ClusterSnapshotDetailResponse), args.Error(1)
}

func TestWaitUntilSnapshotStatusMatches(t *testing.T) {
	origInterval := snapshotStatusPollInterval
	snapshotStatusPollInterval = 5 * time.Millisecond
	t.Cleanup(func() { snapshotStatusPollInterval = origInterval })

	tests := []struct {
		name        string
		mockSetup   func(*mockClusterServiceForSnapshots)
		ctx         func() context.Context
		expectError bool
		errSubstr   string
	}{
		{
			name: "status matches",
			mockSetup: func(m *mockClusterServiceForSnapshots) {
				m.On("GetSnapshot", mock.Anything, "cluster-1", "snap-1").Return(&dbSDK.ClusterSnapshotDetailResponse{
					Status: dbSDK.SnapshotStatusAvailable,
				}, nil)
			},
			ctx:         func() context.Context { return context.Background() },
			expectError: false,
		},
		{
			name: "status is error",
			mockSetup: func(m *mockClusterServiceForSnapshots) {
				m.On("GetSnapshot", mock.Anything, "cluster-1", "snap-1").Return(&dbSDK.ClusterSnapshotDetailResponse{
					Status: dbSDK.SnapshotStatusError,
				}, nil)
			},
			ctx:         func() context.Context { return context.Background() },
			expectError: true,
			errSubstr:   "is in error state",
		},
		{
			name: "sdk error",
			mockSetup: func(m *mockClusterServiceForSnapshots) {
				m.On("GetSnapshot", mock.Anything, "cluster-1", "snap-1").Return(nil, errors.New("network error"))
			},
			ctx:         func() context.Context { return context.Background() },
			expectError: true,
			errSubstr:   "network error",
		},
		{
			name:      "timeout",
			mockSetup: func(m *mockClusterServiceForSnapshots) {},
			ctx: func() context.Context {
				ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
				defer cancel()
				time.Sleep(2 * time.Millisecond)
				return ctx
			},
			expectError: true,
			errSubstr:   "timeout waiting for snapshot",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSvc := &mockClusterServiceForSnapshots{}
			r := &DBaaSClusterSnapshotResource{dbaasClusters: mockSvc}
			tt.mockSetup(mockSvc)

			err := r.waitUntilSnapshotStatusMatches(tt.ctx(), "cluster-1", "snap-1", DBaaSClusterSnapshotStatusAvailable)

			if tt.expectError {
				require.Error(t, err)
				if tt.errSubstr != "" {
					assert.Contains(t, err.Error(), tt.errSubstr)
				}
			} else {
				assert.NoError(t, err)
			}

			mockSvc.AssertExpectations(t)
		})
	}
}
