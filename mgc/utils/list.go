package utils

import (
	"reflect"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// ListOptions represents common listing options for data sources.
type ListOptions struct {
	// Limit specifies the maximum number of items to return.
	// If not set, the default value from the API will be used.
	Limit types.Int64 `tfsdk:"limit"`
	// Offset specifies the number of items to skip before starting to collect the result set.
	// If not set, the default value from the API will be used.
	Offset types.Int64 `tfsdk:"offset"`
	// Sort specifies the field by which to sort the items.
	// The format is "field:direction", e.g., "created_at:asc".
	// If not set, the default value from the API will be used.
	Sort types.String `tfsdk:"sort"`
	// Expand specifies related resources to expand in the response.
	Expand []types.String `tfsdk:"expand"`
}

// ConvertListOptions converts ListOptions to the corresponding fields in the provided struct.
// The struct must be passed as a pointer and can have fields named Limit, Offset, Sort, and Expand.
// Only non-null fields in ListOptions will be set in the struct.
// NOTE: This function uses reflection and may have performance implications.
func ConvertListOptions(opts ListOptions, o any) diag.Diagnostics {
	diags := diag.Diagnostics{}

	if o == nil {
		diags.AddError(
			"Invalid Argument",
			"ConvertListOptions received a nil argument",
		)
		return diags
	}

	tfType := reflect.TypeOf(o)
	if tfType.Kind() != reflect.Ptr || tfType.Elem().Kind() != reflect.Struct {
		diags.AddError(
			"Invalid Argument",
			"ConvertListOptions expects a pointer to a struct",
		)
		return diags
	}

	tfValue := reflect.ValueOf(o).Elem()

	if !opts.Limit.IsNull() {
		n := int(opts.Limit.ValueInt64())
		if tfValue.FieldByName("Limit").IsValid() {
			tfValue.FieldByName("Limit").Set(reflect.ValueOf(&n))
		}
	}

	if !opts.Offset.IsNull() {
		n := int(opts.Offset.ValueInt64())
		if tfValue.FieldByName("Offset").IsValid() {
			tfValue.FieldByName("Offset").Set(reflect.ValueOf(&n))
		}
	}

	if !opts.Sort.IsNull() {
		s := opts.Sort.ValueString()
		if tfValue.FieldByName("Sort").IsValid() {
			tfValue.FieldByName("Sort").Set(reflect.ValueOf(&s))
		}
	}

	if len(opts.Expand) > 0 {
		if _, ok := tfType.Elem().FieldByName("Expand"); ok {
			var expands []string
			for _, e := range opts.Expand {
				if !e.IsNull() {
					expands = append(expands, e.ValueString())
				}
			}
			if len(expands) > 0 {
				tfValue.FieldByName("Expand").Set(reflect.ValueOf(expands))
			}
		}
	}

	return diags
}
