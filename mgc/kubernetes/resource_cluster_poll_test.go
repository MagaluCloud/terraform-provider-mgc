package kubernetes

import (
	"context"
	"testing"
	"time"

	clientSDK "github.com/MagaluCloud/mgc-sdk-go/client"
	k8sSDK "github.com/MagaluCloud/mgc-sdk-go/kubernetes"
	"github.com/stretchr/testify/require"
)

type pollingClusterService struct {
	k8sSDK.ClusterService
	get func(context.Context) (*k8sSDK.Cluster, error)
}

func (s pollingClusterService) Get(ctx context.Context, _ string) (*k8sSDK.Cluster, error) {
	return s.get(ctx)
}

func TestClusterWaitRequiresTargetVersion(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	calls := 0
	r := k8sClusterResource{k8sCluster: pollingClusterService{get: func(context.Context) (*k8sSDK.Cluster, error) {
		calls++
		return &k8sSDK.Cluster{Version: "v1.31.0", Status: &k8sSDK.MessageState{State: "running"}}, nil
	}}}
	_, err := r.GetClusterPooling(ctx, "cluster-1", "v1.32.0", "running")
	require.ErrorIs(t, err, context.DeadlineExceeded)
	require.Equal(t, 1, calls)
}

func TestClusterWaitRejectsMissingStatus(t *testing.T) {
	r := k8sClusterResource{k8sCluster: pollingClusterService{get: func(context.Context) (*k8sSDK.Cluster, error) { return &k8sSDK.Cluster{}, nil }}}
	_, err := r.GetClusterPooling(context.Background(), "cluster-1", "", "running")
	require.ErrorContains(t, err, "missing status")
}

func TestClusterWaitRetriesEventualNotFound(t *testing.T) {
	calls := 0
	r := k8sClusterResource{k8sCluster: pollingClusterService{get: func(context.Context) (*k8sSDK.Cluster, error) {
		calls++
		if calls == 1 {
			return nil, &clientSDK.HTTPError{StatusCode: 404}
		}
		return &k8sSDK.Cluster{Version: "v1.32.0", Status: &k8sSDK.MessageState{State: "running"}}, nil
	}}}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	cluster, err := r.pollCluster(ctx, "cluster-1", "v1.32.0", time.Millisecond, "running")
	require.NoError(t, err)
	require.Equal(t, "v1.32.0", cluster.Version)
	require.Equal(t, 2, calls)
}

func TestClusterWaitRequiresPatchedFields(t *testing.T) {
	oldDescription, description := "original", "edited"
	oldCIDRs, cidrs := []string{"192.0.2.0/24"}, []string{"198.51.100.0/24"}
	patch := k8sSDK.PatchClusterRequest{Description: &description, AllowedCIDRs: &cidrs}
	calls := 0
	r := k8sClusterResource{k8sCluster: pollingClusterService{get: func(context.Context) (*k8sSDK.Cluster, error) {
		calls++
		cluster := &k8sSDK.Cluster{Version: "v1.32.0", Status: &k8sSDK.MessageState{State: "running"}, Description: &description, AllowedCIDRs: &cidrs}
		if calls == 1 {
			cluster.Description = &oldDescription
		}
		if calls <= 2 {
			cluster.AllowedCIDRs = &oldCIDRs
		}
		return cluster, nil
	}}}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	got, err := r.pollClusterUntil(ctx, "cluster-1", "v1.32.0", time.Millisecond, func(cluster k8sSDK.Cluster) bool { return clusterMatchesPatch(cluster, patch) }, "running")
	require.NoError(t, err)
	require.Equal(t, 3, calls)
	require.Equal(t, description, *got.Description)
	require.Equal(t, cidrs, *got.AllowedCIDRs)
}
func TestClusterPatchMatching(t *testing.T) {
	description := "edited"
	version := "v1.32.0"
	empty := []string{}
	ordered := []string{"192.0.2.0/24", "198.51.100.0/24"}
	reversed := []string{"198.51.100.0/24", "192.0.2.0/24"}
	require.True(t, clusterMatchesPatch(k8sSDK.Cluster{}, k8sSDK.PatchClusterRequest{}))
	require.False(t, clusterMatchesPatch(k8sSDK.Cluster{}, k8sSDK.PatchClusterRequest{Description: &description}))
	require.False(t, clusterMatchesPatch(k8sSDK.Cluster{}, k8sSDK.PatchClusterRequest{Version: &version}))
	require.True(t, clusterMatchesPatch(k8sSDK.Cluster{}, k8sSDK.PatchClusterRequest{AllowedCIDRs: &empty}))
	require.False(t, clusterMatchesPatch(k8sSDK.Cluster{AllowedCIDRs: &reversed}, k8sSDK.PatchClusterRequest{AllowedCIDRs: &ordered}))
}
