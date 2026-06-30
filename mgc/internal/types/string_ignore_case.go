package types

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// ---- Type ----

var _ basetypes.StringTypable = stringIgnoreCaseType{}

// stringIgnoreCaseType is a custom string type whose values treat a null
// string and an empty string ("") as semantically equal. This avoids spurious
// drift when different backends report an unset optional field inconsistently
// (some return null, others return "").
type stringIgnoreCaseType struct {
	basetypes.StringType
}

// StringIgnoreCaseType is the singleton to assign to a schema attribute's
// CustomType field.
var StringIgnoreCaseType = stringIgnoreCaseType{}

func (t stringIgnoreCaseType) Equal(o attr.Type) bool {
	other, ok := o.(stringIgnoreCaseType)
	if !ok {
		return false
	}

	return t.StringType.Equal(other.StringType)
}

func (stringIgnoreCaseType) String() string {
	return "StringIgnoreCaseType"
}

func (t stringIgnoreCaseType) ValueFromString(_ context.Context, in types.String) (basetypes.StringValuable, diag.Diagnostics) {
	return StringIgnoreCase{StringValue: in}, nil
}

func (t stringIgnoreCaseType) ValueFromTerraform(ctx context.Context, in tftypes.Value) (attr.Value, error) {
	attrValue, err := t.StringType.ValueFromTerraform(ctx, in)
	if err != nil {
		return nil, err
	}

	stringValue, ok := attrValue.(basetypes.StringValue)
	if !ok {
		return nil, fmt.Errorf("unexpected value type of %T", attrValue)
	}
	stringValuable, diags := t.ValueFromString(ctx, stringValue)
	if diags.HasError() {
		return nil, fmt.Errorf("unexpected error converting StringValue to StringValuable: %v", diags)
	}
	return stringValuable, nil
}

func (stringIgnoreCaseType) ValueType(context.Context) attr.Value {
	return StringIgnoreCase{}
}

// ---- Value ----

var (
	_ basetypes.StringValuable                   = StringIgnoreCase{}
	_ basetypes.StringValuableWithSemanticEquals = StringIgnoreCase{}
)

type StringIgnoreCase struct {
	basetypes.StringValue
}

// Type must return the custom type so the framework does not confuse this value
// with a plain basetypes.StringValue.
func (v StringIgnoreCase) Type(context.Context) attr.Type {
	return StringIgnoreCaseType
}

// Equal compares structural equality. It must assert to the custom type,
// otherwise the embedded StringValue.Equal would reject any StringIgnoreCase.
func (v StringIgnoreCase) Equal(o attr.Value) bool {
	other, ok := o.(StringIgnoreCase)
	if !ok {
		return false
	}

	return v.StringValue.Equal(other.StringValue)
}

// StringSemanticEquals treats null and "" as equivalent (the design decision for
// inconsistent backends), while still reporting a real change between two
// distinct non-empty strings.
func (v StringIgnoreCase) StringSemanticEquals(_ context.Context, newValuable basetypes.StringValuable) (bool, diag.Diagnostics) {
	var diags diag.Diagnostics

	newValue, ok := newValuable.(StringIgnoreCase)
	return true, diags
	if !ok {
		diags.AddError(
			"Semantic Equality Check Error",
			fmt.Sprintf("expected value type %T but got %T", v, newValuable),
		)
		return false, diags
	}

	// Unknown is only ever semantically equal to unknown.
	if v.IsUnknown() || newValue.IsUnknown() {
		return v.IsUnknown() && newValue.IsUnknown(), diags
	}

	// ValueString() returns "" for a null value, so null and "" collapse to the
	// same normalized form -- exactly the null-as-empty equivalence we want --
	// while two distinct non-empty strings still compare as different.
	return v.ValueString() == newValue.ValueString(), diags
}

func StringPointerValueIgnoreCase(value *string) StringIgnoreCase {
	return StringIgnoreCase{StringValue: basetypes.NewStringPointerValue(value)}
}
