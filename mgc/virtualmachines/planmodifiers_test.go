package virtualmachines

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func instanceSchema(t *testing.T) schema.Schema {
	t.Helper()
	resp := &resource.SchemaResponse{}
	(&vmInstances{}).Schema(context.Background(), resource.SchemaRequest{}, resp)
	require.False(t, resp.Diagnostics.HasError())
	return resp.Schema
}

// instanceObject builds a raw instance object with every attribute null except the given ones.
func instanceObject(t *testing.T, s schema.Schema, values map[string]tftypes.Value) tftypes.Value {
	t.Helper()
	objType, ok := s.Type().TerraformType(context.Background()).(tftypes.Object)
	require.True(t, ok)

	attrs := map[string]tftypes.Value{}
	for name, attrType := range objType.AttributeTypes {
		attrs[name] = tftypes.NewValue(attrType, nil)
	}
	for name, v := range values {
		attrs[name] = v
	}
	return tftypes.NewValue(objType, attrs)
}

func TestNullPublicIPv4WhenNotAllocated(t *testing.T) {
	s := instanceSchema(t)
	objType := s.Type().TerraformType(context.Background())
	nullState := tftypes.NewValue(objType, nil)
	existingState := instanceObject(t, s, map[string]tftypes.Value{
		"id": tftypes.NewValue(tftypes.String, "vm-1"),
	})

	tests := []struct {
		name     string
		config   map[string]tftypes.Value
		state    tftypes.Value
		plan     types.String
		expected types.String
	}{
		{
			name:     "create without allocate_public_ipv4",
			config:   map[string]tftypes.Value{},
			state:    nullState,
			plan:     types.StringUnknown(),
			expected: types.StringNull(),
		},
		{
			name:     "create with allocate_public_ipv4 false",
			config:   map[string]tftypes.Value{"allocate_public_ipv4": tftypes.NewValue(tftypes.Bool, false)},
			state:    nullState,
			plan:     types.StringUnknown(),
			expected: types.StringNull(),
		},
		{
			name:     "create with allocate_public_ipv4 true",
			config:   map[string]tftypes.Value{"allocate_public_ipv4": tftypes.NewValue(tftypes.Bool, true)},
			state:    nullState,
			plan:     types.StringUnknown(),
			expected: types.StringUnknown(),
		},
		{
			name:     "create with unknown allocate_public_ipv4",
			config:   map[string]tftypes.Value{"allocate_public_ipv4": tftypes.NewValue(tftypes.Bool, tftypes.UnknownValue)},
			state:    nullState,
			plan:     types.StringUnknown(),
			expected: types.StringUnknown(),
		},
		{
			name:     "create with existing network interface",
			config:   map[string]tftypes.Value{"network_interface_id": tftypes.NewValue(tftypes.String, "port-1")},
			state:    nullState,
			plan:     types.StringUnknown(),
			expected: types.StringUnknown(),
		},
		{
			name:     "update keeps unknown for later modifiers",
			config:   map[string]tftypes.Value{},
			state:    existingState,
			plan:     types.StringUnknown(),
			expected: types.StringUnknown(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := planmodifier.StringRequest{
				Config:     tfsdk.Config{Schema: s, Raw: instanceObject(t, s, tt.config)},
				State:      tfsdk.State{Schema: s, Raw: tt.state},
				PlanValue:  tt.plan,
				StateValue: types.StringNull(),
			}
			resp := &planmodifier.StringResponse{PlanValue: tt.plan}

			NullPublicIPv4WhenNotAllocated().PlanModifyString(context.Background(), req, resp)

			require.False(t, resp.Diagnostics.HasError(), resp.Diagnostics)
			assert.Equal(t, tt.expected, resp.PlanValue)
		})
	}
}

func TestPublicIPv4Value(t *testing.T) {
	assert.True(t, publicIPv4Value(nil).IsNull())
	assert.True(t, publicIPv4Value(ptrString("")).IsNull())
	assert.Equal(t, "1.2.3.4", publicIPv4Value(ptrString("1.2.3.4")).ValueString())
}
