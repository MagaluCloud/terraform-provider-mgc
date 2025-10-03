package utils

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
)

func ExampleConvertListOptions() {
	opts := ListOptions{
		Limit:  types.Int64Value(10),
		Offset: types.Int64Value(5),
		Sort:   types.StringValue("created_at:asc"),
		Expand: []types.String{types.StringValue("details")},
	}

	type APIListParams struct {
		Limit  *int
		Offset *int
		Sort   *string
		Expand []string
	}

	params := APIListParams{}

	diags := ConvertListOptions(opts, &params)
	if diags.HasError() {
		// Handle errors
	}
}

// TestTargetStruct mimics a typical SDK ListOptions struct
type TestTargetStruct struct {
	Limit  *int
	Offset *int
	Sort   *string
	Expand []string
}

// TestStructWithExtraFields tests that extra fields don't cause issues
type TestStructWithExtraFields struct {
	Limit     *int
	Offset    *int
	Sort      *string
	Expand    []string
	ExtraData string
}

// TestStructWithMissingFields tests that missing fields don't cause issues
type TestStructWithMissingFields struct {
	Limit *int
}

func TestConvertListOptions(t *testing.T) {
	intPtr := func(i int) *int { return &i }
	strPtr := func(s string) *string { return &s }

	tests := []struct {
		name           string
		opts           ListOptions
		target         any
		wantErr        bool
		wantErrSummary string
		validate       func(t *testing.T, target any)
	}{
		{
			name:           "nil target",
			opts:           ListOptions{},
			target:         nil,
			wantErr:        true,
			wantErrSummary: "Invalid Argument",
		},
		{
			name:           "non-pointer target",
			opts:           ListOptions{},
			target:         TestTargetStruct{},
			wantErr:        true,
			wantErrSummary: "Invalid Argument",
		},
		{
			name:           "pointer to non-struct",
			opts:           ListOptions{},
			target:         new(int),
			wantErr:        true,
			wantErrSummary: "Invalid Argument",
		},
		{
			name:    "empty options with valid target",
			opts:    ListOptions{},
			target:  &TestTargetStruct{},
			wantErr: false,
			validate: func(t *testing.T, target any) {
				ts := target.(*TestTargetStruct)
				assert.Nil(t, ts.Limit)
				assert.Nil(t, ts.Offset)
				assert.Nil(t, ts.Sort)
				assert.Nil(t, ts.Expand)
			},
		},
		{
			name: "null types.Int64 values should not set fields",
			opts: ListOptions{
				Limit:  types.Int64Null(),
				Offset: types.Int64Null(),
				Sort:   types.StringNull(),
			},
			target:  &TestTargetStruct{},
			wantErr: false,
			validate: func(t *testing.T, target any) {
				ts := target.(*TestTargetStruct)
				assert.Nil(t, ts.Limit)
				assert.Nil(t, ts.Offset)
				assert.Nil(t, ts.Sort)
				assert.Nil(t, ts.Expand)
			},
		},
		{
			name: "unknown types values should set fields to zero values",
			opts: ListOptions{
				Limit:  types.Int64Unknown(),
				Offset: types.Int64Unknown(),
				Sort:   types.StringUnknown(),
			},
			target:  &TestTargetStruct{},
			wantErr: false,
			validate: func(t *testing.T, target any) {
				ts := target.(*TestTargetStruct)
				// NOTE: Unknown values are not zero, so they will be set The function will attempt to extract values
				// from Unknown types
				assert.NotNil(t, ts.Limit)
				assert.Equal(t, 0, *ts.Limit) // Unknown Int64 has zero value
				assert.NotNil(t, ts.Offset)
				assert.Equal(t, 0, *ts.Offset)
				assert.NotNil(t, ts.Sort)
				assert.Equal(t, "", *ts.Sort) // Unknown String has empty value
				assert.Nil(t, ts.Expand)
			},
		},
		{
			name: "only Limit set",
			opts: ListOptions{
				Limit: types.Int64Value(100),
			},
			target:  &TestTargetStruct{},
			wantErr: false,
			validate: func(t *testing.T, target any) {
				ts := target.(*TestTargetStruct)
				assert.NotNil(t, ts.Limit)
				assert.Equal(t, 100, *ts.Limit)
				assert.Nil(t, ts.Offset)
				assert.Nil(t, ts.Sort)
				assert.Nil(t, ts.Expand)
			},
		},
		{
			name: "only Offset set",
			opts: ListOptions{
				Offset: types.Int64Value(50),
			},
			target:  &TestTargetStruct{},
			wantErr: false,
			validate: func(t *testing.T, target any) {
				ts := target.(*TestTargetStruct)
				assert.Nil(t, ts.Limit)
				assert.NotNil(t, ts.Offset)
				assert.Equal(t, 50, *ts.Offset)
				assert.Nil(t, ts.Sort)
				assert.Nil(t, ts.Expand)
			},
		},
		{
			name: "only Sort set",
			opts: ListOptions{
				Sort: types.StringValue("created_at:asc"),
			},
			target:  &TestTargetStruct{},
			wantErr: false,
			validate: func(t *testing.T, target any) {
				ts := target.(*TestTargetStruct)
				assert.Nil(t, ts.Limit)
				assert.Nil(t, ts.Offset)
				assert.NotNil(t, ts.Sort)
				assert.Equal(t, "created_at:asc", *ts.Sort)
				assert.Nil(t, ts.Expand)
			},
		},
		{
			name: "only Expand set with single value",
			opts: ListOptions{
				Expand: []types.String{types.StringValue("volumes")},
			},
			target:  &TestTargetStruct{},
			wantErr: false,
			validate: func(t *testing.T, target any) {
				ts := target.(*TestTargetStruct)
				assert.Nil(t, ts.Limit)
				assert.Nil(t, ts.Offset)
				assert.Nil(t, ts.Sort)
				assert.NotNil(t, ts.Expand)
				assert.Len(t, ts.Expand, 1)
				assert.Equal(t, "volumes", ts.Expand[0])
			},
		},
		{
			name: "only Expand set with multiple values",
			opts: ListOptions{
				Expand: []types.String{
					types.StringValue("volumes"),
					types.StringValue("snapshots"),
					types.StringValue("types"),
				},
			},
			target:  &TestTargetStruct{},
			wantErr: false,
			validate: func(t *testing.T, target any) {
				ts := target.(*TestTargetStruct)
				assert.Nil(t, ts.Limit)
				assert.Nil(t, ts.Offset)
				assert.Nil(t, ts.Sort)
				assert.NotNil(t, ts.Expand)
				assert.Len(t, ts.Expand, 3)
				assert.Equal(t, []string{"volumes", "snapshots", "types"}, ts.Expand)
			},
		},
		{
			name: "all fields set with valid values",
			opts: ListOptions{
				Limit:  types.Int64Value(25),
				Offset: types.Int64Value(10),
				Sort:   types.StringValue("name:desc"),
				Expand: []types.String{
					types.StringValue("details"),
					types.StringValue("metadata"),
				},
			},
			target:  &TestTargetStruct{},
			wantErr: false,
			validate: func(t *testing.T, target any) {
				ts := target.(*TestTargetStruct)
				assert.NotNil(t, ts.Limit)
				assert.Equal(t, 25, *ts.Limit)
				assert.NotNil(t, ts.Offset)
				assert.Equal(t, 10, *ts.Offset)
				assert.NotNil(t, ts.Sort)
				assert.Equal(t, "name:desc", *ts.Sort)
				assert.NotNil(t, ts.Expand)
				assert.Equal(t, []string{"details", "metadata"}, ts.Expand)
			},
		},
		{
			name: "zero int64 values (0) should set fields",
			opts: ListOptions{
				Limit:  types.Int64Value(0),
				Offset: types.Int64Value(0),
			},
			target:  &TestTargetStruct{},
			wantErr: false,
			validate: func(t *testing.T, target any) {
				ts := target.(*TestTargetStruct)
				assert.NotNil(t, ts.Limit)
				assert.Equal(t, 0, *ts.Limit)
				assert.NotNil(t, ts.Offset)
				assert.Equal(t, 0, *ts.Offset)
			},
		},
		{
			name: "empty string value should set field",
			opts: ListOptions{
				Sort: types.StringValue(""),
			},
			target:  &TestTargetStruct{},
			wantErr: false,
			validate: func(t *testing.T, target any) {
				ts := target.(*TestTargetStruct)
				assert.NotNil(t, ts.Sort)
				assert.Equal(t, "", *ts.Sort)
			},
		},
		{
			name: "target struct with extra fields",
			opts: ListOptions{
				Limit:  types.Int64Value(10),
				Offset: types.Int64Value(5),
			},
			target: &TestStructWithExtraFields{
				ExtraData: "should-remain",
			},
			wantErr: false,
			validate: func(t *testing.T, target any) {
				ts := target.(*TestStructWithExtraFields)
				assert.NotNil(t, ts.Limit)
				assert.Equal(t, 10, *ts.Limit)
				assert.NotNil(t, ts.Offset)
				assert.Equal(t, 5, *ts.Offset)
				assert.Equal(t, "should-remain", ts.ExtraData)
			},
		},
		{
			name: "target struct with missing fields should not panic",
			opts: ListOptions{
				Limit:  types.Int64Value(10),
				Offset: types.Int64Value(5),
				Sort:   types.StringValue("name:asc"),
			},
			target:  &TestStructWithMissingFields{},
			wantErr: false,
			validate: func(t *testing.T, target any) {
				ts := target.(*TestStructWithMissingFields)
				assert.NotNil(t, ts.Limit)
				assert.Equal(t, 10, *ts.Limit)
			},
		},
		{
			name: "large int64 values",
			opts: ListOptions{
				Limit:  types.Int64Value(9999999),
				Offset: types.Int64Value(8888888),
			},
			target:  &TestTargetStruct{},
			wantErr: false,
			validate: func(t *testing.T, target any) {
				ts := target.(*TestTargetStruct)
				assert.NotNil(t, ts.Limit)
				assert.Equal(t, 9999999, *ts.Limit)
				assert.NotNil(t, ts.Offset)
				assert.Equal(t, 8888888, *ts.Offset)
			},
		},
		{
			name: "special characters in Sort string",
			opts: ListOptions{
				Sort: types.StringValue("field_name:asc"),
			},
			target:  &TestTargetStruct{},
			wantErr: false,
			validate: func(t *testing.T, target any) {
				ts := target.(*TestTargetStruct)
				assert.NotNil(t, ts.Sort)
				assert.Equal(t, "field_name:asc", *ts.Sort)
			},
		},
		{
			name: "special characters in Expand values",
			opts: ListOptions{
				Expand: []types.String{
					types.StringValue("field.nested"),
					types.StringValue("another-field_with_underscore"),
				},
			},
			target:  &TestTargetStruct{},
			wantErr: false,
			validate: func(t *testing.T, target any) {
				ts := target.(*TestTargetStruct)
				assert.NotNil(t, ts.Expand)
				assert.Equal(t, []string{"field.nested", "another-field_with_underscore"}, ts.Expand)
			},
		},
		{
			name: "target with pre-existing values should be overwritten",
			opts: ListOptions{
				Limit: types.Int64Value(100),
			},
			target: &TestTargetStruct{
				Limit:  intPtr(50),
				Offset: intPtr(25),
				Sort:   strPtr("old:value"),
			},
			wantErr: false,
			validate: func(t *testing.T, target any) {
				ts := target.(*TestTargetStruct)
				assert.NotNil(t, ts.Limit)
				assert.Equal(t, 100, *ts.Limit)
				assert.NotNil(t, ts.Offset)
				assert.Equal(t, 25, *ts.Offset)
				assert.NotNil(t, ts.Sort)
				assert.Equal(t, "old:value", *ts.Sort)
			},
		},
		{
			name: "negative int64 values",
			opts: ListOptions{
				Limit:  types.Int64Value(-1),
				Offset: types.Int64Value(-10),
			},
			target:  &TestTargetStruct{},
			wantErr: false,
			validate: func(t *testing.T, target any) {
				ts := target.(*TestTargetStruct)
				assert.NotNil(t, ts.Limit)
				assert.Equal(t, -1, *ts.Limit)
				assert.NotNil(t, ts.Offset)
				assert.Equal(t, -10, *ts.Offset)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			diags := ConvertListOptions(tt.opts, tt.target)

			if tt.wantErr {
				assert.True(t, diags.HasError(), "expected error but got none")
				if tt.wantErrSummary != "" {
					assert.GreaterOrEqual(t, len(diags.Errors()), 1, "expected at least one error")
					assert.Equal(t, tt.wantErrSummary, diags.Errors()[0].Summary())
				}
			} else {
				assert.False(t, diags.HasError(), "unexpected error: %v", diags)
			}

			if tt.validate != nil {
				tt.validate(t, tt.target)
			}
		})
	}
}
