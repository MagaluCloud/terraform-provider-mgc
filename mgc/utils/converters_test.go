package utils

import (
	"math/big"
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

type (
	testStatus string
	testAccess string
	testRole   string
)

// Satisfy the Stringish constraint
const (
	testStatusPending  testStatus = "PENDING"
	testStatusFailed   testStatus = "FAILED"
	testAccessInternal testAccess = "INTERNAL"
	testAccessExternal testAccess = "EXTERNAL"
	testRoleUser       testRole   = "USER"
	testRoleAdmin      testRole   = "ADMIN"
	testRoleDuplicate  testRole   = "INTERNAL" // Same value as testAccessInternal
	testStatusEmpty    testStatus = ""
)

func TestSdkEnumToTFString(t *testing.T) {
	tests := []struct {
		name     string
		input    any
		expected *string
	}{
		{
			name:     "nil status input",
			input:    (*testStatus)(nil),
			expected: nil,
		},
		{
			name:  "valid testStatus input",
			input: ptr(testStatusFailed),
			expected: func() *string {
				s := "FAILED"
				return &s
			}(),
		},
		{
			name:  "valid testAccess input",
			input: ptr(testAccessExternal),
			expected: func() *string {
				s := "EXTERNAL"
				return &s
			}(),
		},
		{
			name:  "different type same string value",
			input: ptr(testRoleDuplicate),
			expected: func() *string {
				s := "INTERNAL"
				return &s
			}(),
		},
		{
			name:  "empty string value",
			input: ptr(testStatusEmpty),
			expected: func() *string {
				s := ""
				return &s
			}(),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var got *string
			switch v := tc.input.(type) {
			case *testStatus:
				got = SdkEnumToTFString(v)
			case *testAccess:
				got = SdkEnumToTFString(v)
			case *testRole:
				got = SdkEnumToTFString(v)
			default:
				t.Fatalf("unsupported type %T", v)
			}

			if (got == nil) != (tc.expected == nil) {
				t.Fatalf("Expected nil: %v, got: %v", tc.expected == nil, got == nil)
			}
			if got != nil && *got != *tc.expected {
				t.Errorf("Expected %q, got %q", *tc.expected, *got)
			}
		})
	}
}

func ptr[T any](v T) *T {
	return &v
}

func TestGoToDynamic(t *testing.T) {
	cases := []struct {
		name   string
		input  any
		assert func(types.Dynamic, error)
	}{
		{
			name:  "string",
			input: "hello",
			assert: func(d types.Dynamic, err error) {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				v, ok := d.UnderlyingValue().(types.String)
				if !ok || v.ValueString() != "hello" {
					t.Fatalf("expected StringValue \"hello\", got %v", d.UnderlyingValue())
				}
			},
		},
		{
			name:  "int",
			input: 42,
			assert: func(d types.Dynamic, err error) {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				v, ok := d.UnderlyingValue().(types.Int64)
				if !ok || v.ValueInt64() != 42 {
					t.Fatalf("expected Int64Value(42), got %v", d.UnderlyingValue())
				}
			},
		},
		{
			name:  "int32",
			input: int32(7),
			assert: func(d types.Dynamic, err error) {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				v, ok := d.UnderlyingValue().(types.Int32)
				if !ok || v.ValueInt32() != 7 {
					t.Fatalf("expected Int32Value(7), got %v", d.UnderlyingValue())
				}
			},
		},
		{
			name:  "float64",
			input: 3.14,
			assert: func(d types.Dynamic, err error) {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				v, ok := d.UnderlyingValue().(types.Number)
				if !ok {
					t.Fatalf("expected NumberValue, got %v", d.UnderlyingValue())
				}
				f, _ := v.ValueBigFloat().Float64()
				if f != 3.14 {
					t.Fatalf("expected NumberValue(3.14), got %v", f)
				}
			},
		},
		{
			name:  "bool",
			input: true,
			assert: func(d types.Dynamic, err error) {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				v, ok := d.UnderlyingValue().(types.Bool)
				if !ok || !v.ValueBool() {
					t.Fatalf("expected BoolValue(true), got %v", d.UnderlyingValue())
				}
			},
		},
		{
			name:  "unsupported",
			input: struct{}{},
			assert: func(d types.Dynamic, err error) {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				if !d.IsNull() {
					t.Fatalf("expected null dynamic on error, got %v", d)
				}
			},
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			d, err := GoToDynamic(tc.input)
			tc.assert(d, err)
		})
	}
}

func TestDynamicToGoString(t *testing.T) {
	out, err := DynamicToGo[string](types.DynamicValue(types.StringValue("world")))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out != "world" {
		t.Fatalf("expected \"world\", got %v", out)
	}
}

func TestDynamicToGoFloat64(t *testing.T) {
	val := big.NewFloat(2.718)
	out, err := DynamicToGo[float64](types.DynamicValue(types.NumberValue(val)))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out != 2.718 {
		t.Fatalf("expected 2.718, got %v", out)
	}
}

func TestDynamicToGoBool(t *testing.T) {
	out, err := DynamicToGo[bool](types.DynamicValue(types.BoolValue(true)))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !out {
		t.Fatalf("expected true, got %v", out)
	}
}

func TestDynamicToGoNull(t *testing.T) {
	out, err := DynamicToGo[string](types.DynamicNull())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out != "" {
		t.Fatalf("expected zero value, got %v", out)
	}
}

func TestDynamicToGoUnknown(t *testing.T) {
	_, err := DynamicToGo[string](types.DynamicUnknown())
	if err == nil {
		t.Fatalf("expected error for unknown dynamic, got nil")
	}
}

func TestStringSliceToTypesList(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    []string
		expected []string
	}{
		{
			name:     "empty slice",
			input:    []string{},
			expected: []string{},
		},
		{
			name:     "nil slice",
			input:    nil,
			expected: []string{},
		},
		{
			name:     "single element",
			input:    []string{"test"},
			expected: []string{"test"},
		},
		{
			name:     "multiple elements",
			input:    []string{"first", "second", "third"},
			expected: []string{"first", "second", "third"},
		},
		{
			name:     "empty strings in slice",
			input:    []string{"", "test", ""},
			expected: []string{"", "test", ""},
		},
		{
			name:     "special characters",
			input:    []string{"hello world", "test@example.com", "path/to/file"},
			expected: []string{"hello world", "test@example.com", "path/to/file"},
		},
		{
			name:     "unicode characters",
			input:    []string{"тест", "测试", "🚀"},
			expected: []string{"тест", "测试", "🚀"},
		},
		{
			name:     "large slice",
			input:    []string{"a", "b", "c", "d", "e", "f", "g", "h", "i", "j"},
			expected: []string{"a", "b", "c", "d", "e", "f", "g", "h", "i", "j"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := StringSliceToTypesList(tt.input)

			// Verify the result is a List type
			assert.False(t, result.IsNull())
			assert.False(t, result.IsUnknown())

			// Convert back to slice to verify contents
			elements := result.Elements()
			assert.Equal(t, len(tt.expected), len(elements))

			for i, element := range elements {
				stringElement, ok := element.(types.String)
				assert.True(t, ok, "element at index %d should be types.String", i)
				assert.Equal(t, tt.expected[i], stringElement.ValueString())
			}
		})
	}
}
