package mgc

import (
	"context"
	"testing"

	"github.com/MagaluCloud/terraform-provider-mgc/mgc/internal/project"
	"github.com/MagaluCloud/terraform-provider-mgc/mgc/internal/testutils"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/stretchr/testify/assert"
)

// Only products whose API declares x-mgc-project-scope accept a project; the lists
// are closed so a new resource cannot silently gain or miss project_id.
var projectScopedResources = map[string]bool{
	"mgc_block_storage_schedule":           true,
	"mgc_block_storage_schedule_attach":    true,
	"mgc_block_storage_snapshots":          true,
	"mgc_block_storage_volume_attachment":  true,
	"mgc_block_storage_volumes":            true,
	"mgc_kubernetes_cluster":               true,
	"mgc_kubernetes_nodepool":              true,
	"mgc_lbaas_network":                    true,
	"mgc_virtual_machine_instances":        true,
	"mgc_virtual_machine_interface_attach": true,
	"mgc_virtual_machine_snapshots":        true,
}

var projectScopedDataSources = map[string]bool{
	"mgc_block_storage_schedule":        true,
	"mgc_block_storage_schedules":       true,
	"mgc_block_storage_snapshot":        true,
	"mgc_block_storage_snapshots":       true,
	"mgc_block_storage_volume":          true,
	"mgc_block_storage_volumes":         true,
	"mgc_kubernetes_cluster":            true,
	"mgc_kubernetes_cluster_kubeconfig": true,
	"mgc_kubernetes_clusters":           true,
	"mgc_kubernetes_node":               true,
	"mgc_kubernetes_nodepool":           true,
	"mgc_lbaas_network":                 true,
	"mgc_lbaas_network_backend":         true,
	"mgc_lbaas_network_backends":        true,
	"mgc_lbaas_network_certificate":     true,
	"mgc_lbaas_network_certificates":    true,
	"mgc_lbaas_network_healthcheck":     true,
	"mgc_lbaas_network_healthchecks":    true,
	"mgc_lbaas_network_listener":        true,
	"mgc_lbaas_network_listeners":       true,
	"mgc_lbaas_networks":                true,
	"mgc_virtual_machine_instance":      true,
	"mgc_virtual_machine_instances":     true,
	"mgc_virtual_machine_snapshots":     true,
}

func TestProjectIDOnlyOnProjectScopedResources(t *testing.T) {
	ctx := context.Background()
	registered := map[string]bool{}

	for _, factory := range New("test")().Resources(ctx) {
		r := factory()
		var meta resource.MetadataResponse
		r.Metadata(ctx, resource.MetadataRequest{ProviderTypeName: providerTypeName}, &meta)
		s := testutils.GetResourceTestSchema(t, r)

		attr, declared := s.Schema.Attributes["project_id"]
		assert.Equal(t, projectScopedResources[meta.TypeName], declared,
			"resource %s: project_id declared=%v, listed=%v", meta.TypeName, declared, projectScopedResources[meta.TypeName])
		if declared {
			assert.Equal(t, project.ResourceAttribute(), attr, "resource %s must use project.ResourceAttribute()", meta.TypeName)
		}
		registered[meta.TypeName] = true
	}

	for name := range projectScopedResources {
		assert.True(t, registered[name], "resource %s is listed but not registered", name)
	}
}

func TestProjectIDOnlyOnProjectScopedDataSources(t *testing.T) {
	ctx := context.Background()
	registered := map[string]bool{}

	for _, factory := range New("test")().DataSources(ctx) {
		d := factory()
		var meta datasource.MetadataResponse
		d.Metadata(ctx, datasource.MetadataRequest{ProviderTypeName: providerTypeName}, &meta)
		s := testutils.GetDataSourceTestSchema(t, d)

		attr, declared := s.Schema.Attributes["project_id"]
		assert.Equal(t, projectScopedDataSources[meta.TypeName], declared,
			"data source %s: project_id declared=%v, listed=%v", meta.TypeName, declared, projectScopedDataSources[meta.TypeName])
		if declared {
			assert.Equal(t, project.DataSourceAttribute(), attr, "data source %s must use project.DataSourceAttribute()", meta.TypeName)
		}
		registered[meta.TypeName] = true
	}

	for name := range projectScopedDataSources {
		assert.True(t, registered[name], "data source %s is listed but not registered", name)
	}
}
