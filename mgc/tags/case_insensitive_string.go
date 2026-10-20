package tags

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// caseInsensitiveStringType is a string whose value is compared ignoring case.
// The tags API stores color lowercased, so a config written in uppercase would
// otherwise produce a permanent diff.
type caseInsensitiveStringType struct {
	basetypes.StringType
}

var (
	_ basetypes.StringTypable                    = caseInsensitiveStringType{}
	_ basetypes.StringValuableWithSemanticEquals = caseInsensitiveStringValue{}
)

func (t caseInsensitiveStringType) Equal(o attr.Type) bool {
	other, ok := o.(caseInsensitiveStringType)
	if !ok {
		return false
	}
	return t.StringType.Equal(other.StringType)
}

func (t caseInsensitiveStringType) String() string {
	return "caseInsensitiveStringType"
}

func (t caseInsensitiveStringType) ValueFromString(_ context.Context, in basetypes.StringValue) (basetypes.StringValuable, diag.Diagnostics) {
	return caseInsensitiveStringValue{StringValue: in}, nil
}

func (t caseInsensitiveStringType) ValueFromTerraform(ctx context.Context, in tftypes.Value) (attr.Value, error) {
	attrValue, err := t.StringType.ValueFromTerraform(ctx, in)
	if err != nil {
		return nil, err
	}

	stringValue, ok := attrValue.(basetypes.StringValue)
	if !ok {
		return nil, fmt.Errorf("unexpected value type %T", attrValue)
	}

	stringValuable, diags := t.ValueFromString(ctx, stringValue)
	if diags.HasError() {
		return nil, fmt.Errorf("unexpected error converting StringValue to StringValuable: %v", diags)
	}
	return stringValuable, nil
}

func (t caseInsensitiveStringType) ValueType(context.Context) attr.Value {
	return caseInsensitiveStringValue{}
}

type caseInsensitiveStringValue struct {
	basetypes.StringValue
}

func (v caseInsensitiveStringValue) Type(context.Context) attr.Type {
	return caseInsensitiveStringType{}
}

func (v caseInsensitiveStringValue) Equal(o attr.Value) bool {
	other, ok := o.(caseInsensitiveStringValue)
	if !ok {
		return false
	}
	return v.StringValue.Equal(other.StringValue)
}

func (v caseInsensitiveStringValue) StringSemanticEquals(_ context.Context, newValuable basetypes.StringValuable) (bool, diag.Diagnostics) {
	var diags diag.Diagnostics

	newValue, ok := newValuable.(caseInsensitiveStringValue)
	if !ok {
		return false, diags
	}

	// Null and unknown carry no text to fold, so fall back to exact comparison.
	if v.IsNull() || v.IsUnknown() || newValue.IsNull() || newValue.IsUnknown() {
		return v.StringValue.Equal(newValue.StringValue), diags
	}

	return strings.EqualFold(v.ValueString(), newValue.ValueString()), diags
}

func newCaseInsensitiveString(s string) caseInsensitiveStringValue {
	return caseInsensitiveStringValue{StringValue: basetypes.NewStringValue(s)}
}
