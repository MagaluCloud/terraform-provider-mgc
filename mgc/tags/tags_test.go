package tags

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetResources(t *testing.T) {
	t.Parallel()

	assert.NotEmpty(t, GetResources())
}

func TestGetDataSources(t *testing.T) {
	t.Parallel()

	assert.NotEmpty(t, GetDataSources())
}

func newDataSourceConfig(t *testing.T, dataSourceSchema schema.Schema, attributes map[string]tftypes.Value) tfsdk.Config {
	t.Helper()

	objectType, ok := dataSourceSchema.Type().TerraformType(context.Background()).(tftypes.Object)
	require.True(t, ok, "the schema of a data source is always an object")

	values := make(map[string]tftypes.Value, len(objectType.AttributeTypes))
	for name, attributeType := range objectType.AttributeTypes {
		if value, given := attributes[name]; given {
			values[name] = value
			continue
		}
		values[name] = tftypes.NewValue(attributeType, nil)
	}

	return tfsdk.Config{Schema: dataSourceSchema, Raw: tftypes.NewValue(objectType, values)}
}
