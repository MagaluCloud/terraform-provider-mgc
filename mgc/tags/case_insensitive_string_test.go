package tags

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The tags API always stores color lowercased, so a config written in uppercase
// must compare equal to what the API returns.
func TestCaseInsensitiveStringSemanticEquals(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		prior    basetypes.StringValue
		new      basetypes.StringValue
		expected bool
	}{
		{
			name:     "same case",
			prior:    basetypes.NewStringValue("ff0000"),
			new:      basetypes.NewStringValue("ff0000"),
			expected: true,
		},
		{
			name:     "config uppercase, api lowercase",
			prior:    basetypes.NewStringValue("FF0000"),
			new:      basetypes.NewStringValue("ff0000"),
			expected: true,
		},
		{
			name:     "mixed case",
			prior:    basetypes.NewStringValue("Ff00Aa"),
			new:      basetypes.NewStringValue("ff00aa"),
			expected: true,
		},
		{
			name:     "different colors",
			prior:    basetypes.NewStringValue("ff0000"),
			new:      basetypes.NewStringValue("00ff00"),
			expected: false,
		},
		{
			name:     "both null",
			prior:    basetypes.NewStringNull(),
			new:      basetypes.NewStringNull(),
			expected: true,
		},
		{
			name:     "null and value",
			prior:    basetypes.NewStringNull(),
			new:      basetypes.NewStringValue("ff0000"),
			expected: false,
		},
		{
			name:     "value and null",
			prior:    basetypes.NewStringValue("ff0000"),
			new:      basetypes.NewStringNull(),
			expected: false,
		},
		{
			name:     "both unknown",
			prior:    basetypes.NewStringUnknown(),
			new:      basetypes.NewStringUnknown(),
			expected: true,
		},
		{
			name:     "unknown and value",
			prior:    basetypes.NewStringUnknown(),
			new:      basetypes.NewStringValue("ff0000"),
			expected: false,
		},
		{
			name:     "null and unknown",
			prior:    basetypes.NewStringNull(),
			new:      basetypes.NewStringUnknown(),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			prior := caseInsensitiveStringValue{StringValue: tt.prior}
			next := caseInsensitiveStringValue{StringValue: tt.new}

			equal, diags := prior.StringSemanticEquals(context.Background(), next)

			require.False(t, diags.HasError(), "unexpected diagnostics: %v", diags)
			assert.Equal(t, tt.expected, equal)
		})
	}
}

func TestCaseInsensitiveStringSemanticEqualsForeignType(t *testing.T) {
	t.Parallel()

	prior := newCaseInsensitiveString("ff0000")

	equal, diags := prior.StringSemanticEquals(context.Background(), basetypes.NewStringValue("ff0000"))

	require.False(t, diags.HasError(), "unexpected diagnostics: %v", diags)
	assert.False(t, equal, "a plain StringValue is not semantically comparable")
}

func TestCaseInsensitiveStringTypeValueFromTerraform(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   tftypes.Value
		want caseInsensitiveStringValue
	}{
		{
			name: "known value",
			in:   tftypes.NewValue(tftypes.String, "FF0000"),
			want: newCaseInsensitiveString("FF0000"),
		},
		{
			name: "null",
			in:   tftypes.NewValue(tftypes.String, nil),
			want: caseInsensitiveStringValue{StringValue: basetypes.NewStringNull()},
		},
		{
			name: "unknown",
			in:   tftypes.NewValue(tftypes.String, tftypes.UnknownValue),
			want: caseInsensitiveStringValue{StringValue: basetypes.NewStringUnknown()},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := caseInsensitiveStringType{}.ValueFromTerraform(context.Background(), tt.in)

			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestCaseInsensitiveStringTypeContract(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	typ := caseInsensitiveStringType{}

	t.Run("equal only to itself", func(t *testing.T) {
		t.Parallel()

		assert.True(t, typ.Equal(caseInsensitiveStringType{}))
		assert.False(t, typ.Equal(basetypes.StringType{}))
	})

	t.Run("value type round trip", func(t *testing.T) {
		t.Parallel()

		var _ attr.Type = typ
		assert.IsType(t, caseInsensitiveStringValue{}, typ.ValueType(ctx))

		value, diags := typ.ValueFromString(ctx, basetypes.NewStringValue("ff0000"))
		require.False(t, diags.HasError(), "unexpected diagnostics: %v", diags)
		assert.Equal(t, newCaseInsensitiveString("ff0000"), value)
		assert.True(t, typ.Equal(value.Type(ctx)))
	})

	t.Run("values equal only within the custom type", func(t *testing.T) {
		t.Parallel()

		value := newCaseInsensitiveString("ff0000")
		assert.True(t, value.Equal(newCaseInsensitiveString("ff0000")))
		assert.False(t, value.Equal(newCaseInsensitiveString("FF0000")), "Equal is exact; only SemanticEquals folds case")
		assert.False(t, value.Equal(basetypes.NewStringValue("ff0000")))
	})
}
