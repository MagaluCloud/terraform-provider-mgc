package utils

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
)

func stringSet(values ...string) types.Set {
	elements := make([]attr.Value, 0, len(values))
	for _, value := range values {
		elements = append(elements, types.StringValue(value))
	}
	return types.SetValueMust(types.StringType, elements)
}

// FlattenStringValue keeps the representation the configuration uses whenever
// both sides are semantically empty, so "" and null never fight each other.
func TestFlattenStringValue(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		tfValue  types.String
		apiValue *string
		expected types.String
	}{
		{
			name:     "null config, absent in the api",
			tfValue:  types.StringNull(),
			apiValue: nil,
			expected: types.StringNull(),
		},
		{
			name:     "null config, empty in the api",
			tfValue:  types.StringNull(),
			apiValue: ptr(""),
			expected: types.StringNull(),
		},
		{
			name:     "empty config, absent in the api",
			tfValue:  types.StringValue(""),
			apiValue: nil,
			expected: types.StringValue(""),
		},
		{
			name:     "empty config, empty in the api",
			tfValue:  types.StringValue(""),
			apiValue: ptr(""),
			expected: types.StringValue(""),
		},
		{
			name:     "value only in the api",
			tfValue:  types.StringNull(),
			apiValue: ptr("from-api"),
			expected: types.StringValue("from-api"),
		},
		{
			// Cleared out of band: the state has to follow the API, otherwise
			// the drift would never show up in the plan.
			name:     "value in the state, absent in the api",
			tfValue:  types.StringValue("from-state"),
			apiValue: nil,
			expected: types.StringNull(),
		},
		{
			name:     "changed in the api",
			tfValue:  types.StringValue("from-state"),
			apiValue: ptr("from-api"),
			expected: types.StringValue("from-api"),
		},
		{
			name:     "same on both sides",
			tfValue:  types.StringValue("same"),
			apiValue: ptr("same"),
			expected: types.StringValue("same"),
		},
		{
			name:     "unknown is always replaced by the api value",
			tfValue:  types.StringUnknown(),
			apiValue: ptr("from-api"),
			expected: types.StringValue("from-api"),
		},
		{
			// An unknown must never survive into the state: it would break the
			// apply with a "value is not wholly known" error.
			name:     "unknown with nothing in the api becomes null",
			tfValue:  types.StringUnknown(),
			apiValue: nil,
			expected: types.StringNull(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.expected, FlattenStringValue(tt.tfValue, tt.apiValue))
		})
	}
}

func TestFlattenTypeSetStringArray(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		tfValue  types.Set
		apiValue *[]string
		expected types.Set
	}{
		{
			name:     "null config, absent in the api",
			tfValue:  types.SetNull(types.StringType),
			apiValue: nil,
			expected: types.SetNull(types.StringType),
		},
		{
			name:     "null config, empty in the api",
			tfValue:  types.SetNull(types.StringType),
			apiValue: ptr([]string{}),
			expected: types.SetNull(types.StringType),
		},
		{
			name:     "empty config, empty in the api",
			tfValue:  stringSet(),
			apiValue: ptr([]string{}),
			expected: stringSet(),
		},
		{
			name:     "empty config, absent in the api",
			tfValue:  stringSet(),
			apiValue: nil,
			expected: stringSet(),
		},
		{
			name:     "values only in the api",
			tfValue:  types.SetNull(types.StringType),
			apiValue: ptr([]string{"a", "b"}),
			expected: stringSet("a", "b"),
		},
		{
			// Emptied out of band. An empty list and an absent one are both
			// "no values", but they do not land in the state as the same value:
			// only a nil becomes null.
			name:     "values in the state, emptied in the api",
			tfValue:  stringSet("a"),
			apiValue: ptr([]string{}),
			expected: stringSet(),
		},
		{
			name:     "values in the state, absent in the api",
			tfValue:  stringSet("a"),
			apiValue: nil,
			expected: types.SetNull(types.StringType),
		},
		{
			name:     "changed in the api",
			tfValue:  stringSet("a"),
			apiValue: ptr([]string{"b"}),
			expected: stringSet("b"),
		},
		{
			// A set has no order, so the API may echo the elements shuffled.
			name:     "same values in another order",
			tfValue:  stringSet("a", "b"),
			apiValue: ptr([]string{"b", "a"}),
			expected: stringSet("b", "a"),
		},
		{
			name:     "unknown is always replaced by the api value",
			tfValue:  types.SetUnknown(types.StringType),
			apiValue: ptr([]string{"a"}),
			expected: stringSet("a"),
		},
		{
			name:     "unknown with nothing in the api becomes null",
			tfValue:  types.SetUnknown(types.StringType),
			apiValue: nil,
			expected: types.SetNull(types.StringType),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := FlattenTypeSetStringArray(tt.tfValue, tt.apiValue)

			assert.Equal(t, tt.expected, got)
			assert.True(t, tt.expected.Equal(got), "sets must compare equal regardless of order")
		})
	}
}
