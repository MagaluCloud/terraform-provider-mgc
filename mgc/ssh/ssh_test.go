package ssh

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/stretchr/testify/assert"
)

func TestGetDataSources(t *testing.T) {
	t.Parallel()
	dataSources := GetDataSources()

	assert.NotEmpty(t, dataSources)

	for _, factory := range dataSources {
		ds := factory()
		assert.NotNil(t, ds)
		assert.Implements(t, (*datasource.DataSource)(nil), ds)
	}
}

func TestGetResources(t *testing.T) {
	t.Parallel()
	resources := GetResources()

	assert.NotEmpty(t, resources)

	for _, factory := range resources {
		r := factory()
		assert.NotNil(t, r)
		assert.Implements(t, (*resource.Resource)(nil), r)
	}
}
