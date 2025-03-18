package tfutil

import (
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
)

func TestConvertTimeToRFC3339(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    *time.Time
		expected *string
	}{
		{
			name:     "nil input",
			input:    nil,
			expected: nil,
		},
		{
			name: "valid time",
			input: func() *time.Time {
				tm := time.Date(2023, time.January, 1, 12, 0, 0, 0, time.UTC)
				return &tm
			}(),
			expected: func() *string {
				s := "2023-01-01T12:00:00Z"
				return &s
			}(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ConvertTimeToRFC3339(tt.input)
			if tt.expected == nil {
				assert.Nil(t, result)
			} else {
				assert.Equal(t, *tt.expected, *result)
			}
		})
	}
}

func TestConvertInt64PointerToIntPointer(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    *int64
		expected *int
	}{
		{
			name:     "nil input",
			input:    nil,
			expected: nil,
		},
		{
			name: "valid int64",
			input: func() *int64 {
				i := int64(42)
				return &i
			}(),
			expected: func() *int {
				i := 42
				return &i
			}(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ConvertInt64PointerToIntPointer(tt.input)
			if tt.expected == nil {
				assert.Nil(t, result)
			} else {
				assert.Equal(t, *tt.expected, *result)
			}
		})
	}
}

func TestConvertIntPointerToInt64Pointer(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    *int
		expected *int64
	}{
		{
			name:     "nil input",
			input:    nil,
			expected: nil,
		},
		{
			name: "valid int",
			input: func() *int {
				i := 42
				return &i
			}(),
			expected: func() *int64 {
				i := int64(42)
				return &i
			}(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ConvertIntPointerToInt64Pointer(tt.input)
			if tt.expected == nil {
				assert.Nil(t, result)
			} else {
				assert.Equal(t, *tt.expected, *result)
			}
		})
	}
}

func TestSdkParamValueToString(t *testing.T) {
	t.Parallel()

	stringPtr := func(s string) *string { return &s }
	floatPtr := func(f float64) *float64 { return &f }
	intPtr := func(i int) *int { return &i }
	boolPtr := func(b bool) *bool { return &b }

	tests := []struct {
		name     string
		input    any
		expected string
	}{
		{
			name:     "nil input",
			input:    nil,
			expected: "",
		},
		{
			name:     "string pointer",
			input:    stringPtr("test"),
			expected: "test",
		},
		{
			name:     "nil string pointer",
			input:    (*string)(nil),
			expected: "",
		},
		{
			name:     "float64 pointer",
			input:    floatPtr(3.14),
			expected: "3.14",
		},
		{
			name:     "nil float64 pointer",
			input:    (*float64)(nil),
			expected: "",
		},
		{
			name:     "int pointer",
			input:    intPtr(42),
			expected: "42",
		},
		{
			name:     "nil int pointer",
			input:    (*int)(nil),
			expected: "",
		},
		{
			name:     "bool pointer - true",
			input:    boolPtr(true),
			expected: "true",
		},
		{
			name:     "bool pointer - false",
			input:    boolPtr(false),
			expected: "false",
		},
		{
			name:     "nil bool pointer",
			input:    (*bool)(nil),
			expected: "",
		},
		{
			name:     "string value",
			input:    "test",
			expected: "test",
		},
		{
			name:     "float64 value",
			input:    3.14,
			expected: "3.14",
		},
		{
			name:     "int value",
			input:    42,
			expected: "42",
		},
		{
			name:     "bool value - true",
			input:    true,
			expected: "true",
		},
		{
			name:     "bool value - false",
			input:    false,
			expected: "false",
		},
		{
			name:     "unsupported type",
			input:    []string{"test"},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SdkParamValueToString(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestConvertXZoneToAvailabilityZone(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		region   string
		xZone    string
		expected string
	}{
		{
			name:     "regular case",
			region:   "br-ne1",
			xZone:    "a",
			expected: "br-ne1-a",
		},
		{
			name:     "uppercase input",
			region:   "br-ne1",
			xZone:    "B",
			expected: "br-ne1-b",
		},
		{
			name:     "empty region",
			region:   "",
			xZone:    "c",
			expected: "-c",
		},
		{
			name:     "empty xZone",
			region:   "us-west-2",
			xZone:    "",
			expected: "us-west-2-",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ConvertXZoneToAvailabilityZone(tt.region, tt.xZone)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestConvertAvailabilityZoneToXZone(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name             string
		availabilityZone string
		expectedZone     string
		expectError      bool
	}{
		{
			name:             "regular case",
			availabilityZone: "br-ne1-a",
			expectedZone:     "a",
			expectError:      false,
		},
		{
			name:             "uppercase input - should be converted to lowercase",
			availabilityZone: "BR-NE1-A",
			expectedZone:     "",
			expectError:      true,
		},
		{
			name:             "empty string",
			availabilityZone: "",
			expectedZone:     "",
			expectError:      true,
		},
		{
			name:             "invalid format - no zone",
			availabilityZone: "br-ne1",
			expectedZone:     "",
			expectError:      true,
		},
		{
			name:             "invalid format - wrong pattern",
			availabilityZone: "brne1a",
			expectedZone:     "",
			expectError:      true,
		},
		{
			name:             "invalid format - missing dash",
			availabilityZone: "br-ne1a",
			expectedZone:     "",
			expectError:      true,
		},
		{
			name:             "invalid format - incorrect region length",
			availabilityZone: "abc-ne1-a",
			expectedZone:     "",
			expectError:      true,
		},
		{
			name:             "invalid format - incorrect subregion length",
			availabilityZone: "br-abc1-a",
			expectedZone:     "",
			expectError:      true,
		},
		{
			name:             "invalid format - incorrect zone format",
			availabilityZone: "br-ne1-ab",
			expectedZone:     "",
			expectError:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ConvertAvailabilityZoneToXZone(tt.availabilityZone)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedZone, result)
			}
		})
	}
}

// Structure test - just ensure the structs compile properly
func TestStructs(t *testing.T) {
	t.Parallel()

	// GenericIDNameModel
	idNameModel := GenericIDNameModel{
		Name: types.StringValue("test-name"),
		ID:   types.StringValue("test-id"),
	}
	assert.Equal(t, "test-name", idNameModel.Name.ValueString())
	assert.Equal(t, "test-id", idNameModel.ID.ValueString())

	// GenericIDModel
	idModel := GenericIDModel{
		ID: types.StringValue("test-id"),
	}
	assert.Equal(t, "test-id", idModel.ID.ValueString())
}
