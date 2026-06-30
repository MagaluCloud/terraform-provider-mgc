package types

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStringOptionalDefault_SemanticEquals(t *testing.T) {
	ctx := context.Background()

	cases := []struct {
		name  string
		prior StringOptionalDefault
		next  StringOptionalDefault
		equal bool
	}{
		{"null equals empty", StringNull(), StringValue(""), true},
		{"empty equals null", StringValue(""), StringNull(), true},
		{"null equals null", StringNull(), StringNull(), true},
		{"same non-empty", StringValue("prod"), StringValue("prod"), true},
		{"different non-empty", StringValue("prod"), StringValue("dev"), false},
		{"null differs from non-empty", StringNull(), StringValue("prod"), false},
		{"empty differs from non-empty", StringValue(""), StringValue("prod"), false},
		{"unknown equals unknown", StringUnknown(), StringUnknown(), true},
		{"unknown differs from null", StringUnknown(), StringNull(), false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, diags := tc.prior.StringSemanticEquals(ctx, tc.next)
			require.False(t, diags.HasError(), diags)
			assert.Equal(t, tc.equal, got)
		})
	}
}

func TestStringOptionalDefault_SemanticEquals_wrongType(t *testing.T) {
	_, diags := StringValue("x").StringSemanticEquals(context.Background(), basetypes.NewStringValue("x"))
	assert.True(t, diags.HasError())
}

func TestStringPointerValue(t *testing.T) {
	assert.True(t, StringPointerValue(nil).IsNull())
	v := "prod"
	assert.Equal(t, "prod", StringPointerValue(&v).ValueString())
}

func listVal(t *testing.T, elems ...string) ListOptionalDefault {
	t.Helper()
	v, diags := ListOptionalDefaultValueFrom(context.Background(), types.StringType, elems)
	require.False(t, diags.HasError(), diags)
	return v
}

func TestListOptionalDefault_SemanticEquals(t *testing.T) {
	ctx := context.Background()
	null := ListOptionalDefaultNull(types.StringType)
	empty := listVal(t)
	unknown := ListOptionalDefault{ListValue: basetypes.NewListUnknown(types.StringType)}

	cases := []struct {
		name  string
		prior ListOptionalDefault
		next  ListOptionalDefault
		equal bool
	}{
		{"null equals empty", null, empty, true},
		{"empty equals null", empty, null, true},
		{"null equals null", null, null, true},
		{"same elements", listVal(t, "a", "b"), listVal(t, "a", "b"), true},
		{"different element", listVal(t, "a"), listVal(t, "b"), false},
		{"different length", listVal(t, "a"), listVal(t, "a", "b"), false},
		{"order matters", listVal(t, "a", "b"), listVal(t, "b", "a"), false},
		{"null differs from non-empty", null, listVal(t, "a"), false},
		{"empty differs from non-empty", empty, listVal(t, "a"), false},
		{"unknown equals unknown", unknown, unknown, true},
		{"unknown differs from null", unknown, null, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, diags := tc.prior.ListSemanticEquals(ctx, tc.next)
			require.False(t, diags.HasError(), diags)
			assert.Equal(t, tc.equal, got)
		})
	}
}
