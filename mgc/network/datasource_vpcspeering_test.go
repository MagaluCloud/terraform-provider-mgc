package network

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	dschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func peeringTestDataSourceSchema() dschema.Schema {
	d := &NetworkVpcsPeeringDatasource{}
	resp := &datasource.SchemaResponse{}
	d.Schema(context.Background(), datasource.SchemaRequest{}, resp)
	return resp.Schema
}

func TestNetworkVpcsPeeringDatasource_Schema(t *testing.T) {
	t.Parallel()

	attrs := peeringTestDataSourceSchema().Attributes

	id, ok := attrs["id"].(dschema.StringAttribute)
	require.True(t, ok)
	assert.True(t, id.Required)

	for _, name := range []string{"name", "description", "status", "requester_vpc_id", "accepter_vpc_id", "created_at", "updated_at"} {
		attr, ok := attrs[name].(dschema.StringAttribute)
		require.True(t, ok, name)
		assert.True(t, attr.Computed, "%s must be computed", name)
	}
}
