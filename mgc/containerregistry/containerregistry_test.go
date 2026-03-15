package containerregistry

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/stretchr/testify/assert"
)

func TestGetDataSources(t *testing.T) {
	dataSources := GetDataSources()

	assert.NotNil(t, dataSources)
	assert.Len(t, dataSources, 6)

	expectedTypes := []string{
		"*containerregistry.DataSourceCRCredentials",
		"*containerregistry.DataSourceCRImages",
		"*containerregistry.DataSourceCRRegistries",
		"*containerregistry.DataSourceCRRepositories",
		"*containerregistry.ProxyCacheDataSource",
		"*containerregistry.ProxyCacheListDataSource",
	}

	for i, factory := range dataSources {
		assert.NotNil(t, factory)
		ds := factory()
		assert.NotNil(t, ds)
		assert.Implements(t, (*datasource.DataSource)(nil), ds)
		assert.Equal(t, expectedTypes[i], getTypeName(ds), "DataSource type at index %d should match", i)
	}
}

func TestGetResources(t *testing.T) {
	resources := GetResources()

	assert.NotNil(t, resources)
	assert.Len(t, resources, 2)

	expectedTypes := []string{
		"*containerregistry.ContainerRegistryResource",
		"*containerregistry.ProxyCacheResource",
	}

	for i, factory := range resources {
		assert.NotNil(t, factory)
		r := factory()
		assert.NotNil(t, r)
		assert.Implements(t, (*resource.Resource)(nil), r)
		assert.Equal(t, expectedTypes[i], getTypeName(r), "Resource type at index %d should match", i)
	}
}

func getTypeName(v interface{}) string {
	return "*containerregistry." + getConcreteTypeName(v)
}

func getConcreteTypeName(v interface{}) string {
	switch v.(type) {
	case *DataSourceCRCredentials:
		return "DataSourceCRCredentials"
	case *DataSourceCRImages:
		return "DataSourceCRImages"
	case *DataSourceCRRegistries:
		return "DataSourceCRRegistries"
	case *DataSourceCRRepositories:
		return "DataSourceCRRepositories"
	case *ProxyCacheDataSource:
		return "ProxyCacheDataSource"
	case *ProxyCacheListDataSource:
		return "ProxyCacheListDataSource"
	case *ContainerRegistryResource:
		return "ContainerRegistryResource"
	case *ProxyCacheResource:
		return "ProxyCacheResource"
	default:
		return "unknown"
	}
}
