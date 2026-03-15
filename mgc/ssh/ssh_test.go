package ssh

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/stretchr/testify/assert"
)

func TestGetDataSources(t *testing.T) {
	dataSources := GetDataSources()

	assert.NotNil(t, dataSources)
	assert.Len(t, dataSources, 1)

	factory := dataSources[0]
	assert.NotNil(t, factory)

	ds := factory()
	assert.NotNil(t, ds)
	assert.Implements(t, (*datasource.DataSource)(nil), ds)

	_, ok := ds.(*DataSourceSSH)
	assert.True(t, ok)
}

func TestGetResources(t *testing.T) {
	resources := GetResources()

	assert.NotNil(t, resources)
	assert.Len(t, resources, 1)

	factory := resources[0]
	assert.NotNil(t, factory)

	r := factory()
	assert.NotNil(t, r)
	assert.Implements(t, (*resource.Resource)(nil), r)

	_, ok := r.(*sshKeys)
	assert.True(t, ok)
}
