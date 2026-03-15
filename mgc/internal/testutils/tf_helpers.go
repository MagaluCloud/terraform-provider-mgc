package testutils

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/stretchr/testify/assert"
)

// GetResourceTestSchema is a generic helper to retrieve the schema of a Resource for unit testing.
func GetResourceTestSchema(t *testing.T, r resource.Resource) resource.SchemaResponse {
	ctx := context.Background()
	schemaReq := resource.SchemaRequest{}
	schemaResp := &resource.SchemaResponse{}

	r.Schema(ctx, schemaReq, schemaResp)

	assert.False(t, schemaResp.Diagnostics.HasError(), "Resource schema generation returned errors: %v", schemaResp.Diagnostics)

	return *schemaResp
}

// GetDataSourceTestSchema is a generic helper to retrieve the schema of a DataSource for unit testing.
func GetDataSourceTestSchema(t *testing.T, d datasource.DataSource) datasource.SchemaResponse {
	ctx := context.Background()
	schemaReq := datasource.SchemaRequest{}
	schemaResp := &datasource.SchemaResponse{}

	d.Schema(ctx, schemaReq, schemaResp)

	assert.False(t, schemaResp.Diagnostics.HasError(), "DataSource schema generation returned errors: %v", schemaResp.Diagnostics)

	return *schemaResp
}
