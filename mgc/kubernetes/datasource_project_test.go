package kubernetes

import (
	"context"
	"testing"

	clientSDK "github.com/MagaluCloud/mgc-sdk-go/client"
	k8sSDK "github.com/MagaluCloud/mgc-sdk-go/kubernetes"
	"github.com/MagaluCloud/terraform-provider-mgc/mgc/internal/testutils"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/stretchr/testify/assert"
)

func dataSourceState(t *testing.T, d datasource.DataSource, values map[string]tftypes.Value) (tfsdk.Config, tfsdk.State) {
	s := testutils.GetDataSourceTestSchema(t, d)
	raw := testutils.ObjectWith(context.Background(), s.Schema.Type(), values)
	return tfsdk.Config{Schema: s.Schema, Raw: raw}, tfsdk.State{Schema: s.Schema, Raw: raw}
}

func TestKubernetesDataSourcesKeepProject(t *testing.T) {
	ctx := context.Background()

	for _, projectID := range []string{"550e8400-e29b-41d4-a716-446655440000", ""} {
		expectedCallOpts := clientSDK.CallOptions{ProjectID: projectID}
		expectedProject := types.StringNull()
		if projectID != "" {
			expectedProject = types.StringValue(projectID)
		}

		t.Run("cluster/"+projectID, func(t *testing.T) {
			mock := &mockClusterService{cluster: &k8sSDK.Cluster{ID: "cluster-uuid", Name: "c", Version: "v1.30.0"}}
			d := &DataSourceKubernetesCluster{sdkClient: mock}
			config, state := dataSourceState(t, d, map[string]tftypes.Value{
				"id":         tftypes.NewValue(tftypes.String, "cluster-uuid"),
				"project_id": testutils.OptionalString(projectID),
			})

			resp := &datasource.ReadResponse{State: state}
			d.Read(ctx, datasource.ReadRequest{Config: config}, resp)
			assert.False(t, resp.Diagnostics.HasError(), "%v", resp.Diagnostics)

			assert.Equal(t, expectedCallOpts, mock.gotCallOpts)
			var got types.String
			assert.False(t, resp.State.GetAttribute(ctx, path.Root("project_id"), &got).HasError())
			assert.Equal(t, expectedProject, got)
		})

		t.Run("nodepool/"+projectID, func(t *testing.T) {
			mock := &mockNodePoolService{GetFunc: func(ctx context.Context, clusterID, nodepoolID string) (*k8sSDK.NodePool, error) {
				return &k8sSDK.NodePool{ID: "pool-1", Name: "p"}, nil
			}}
			d := &DataSourceKubernetesNodepool{sdkClient: mock}
			config, state := dataSourceState(t, d, map[string]tftypes.Value{
				"id":         tftypes.NewValue(tftypes.String, "pool-1"),
				"cluster_id": tftypes.NewValue(tftypes.String, "cluster-uuid"),
				"project_id": testutils.OptionalString(projectID),
			})

			resp := &datasource.ReadResponse{State: state}
			d.Read(ctx, datasource.ReadRequest{Config: config}, resp)
			assert.False(t, resp.Diagnostics.HasError(), "%v", resp.Diagnostics)

			assert.Equal(t, expectedCallOpts, mock.gotCallOpts)
			var got types.String
			assert.False(t, resp.State.GetAttribute(ctx, path.Root("project_id"), &got).HasError())
			assert.Equal(t, expectedProject, got)
		})
	}
}
