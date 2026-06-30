package types

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// ---- Type ----

var _ basetypes.StringTypable = (*stringOptionalDefaultType)(nil)

// stringOptionalDefaultType is a custom string type whose values treat a null
// string and an empty string ("") as semantically equal. This avoids spurious
// drift when different backends report an unset optional field inconsistently
// (some return null, others return "").
type stringOptionalDefaultType struct {
	basetypes.StringType
}

// StringOptionalDefaultType is the singleton to assign to a schema attribute's
// CustomType field.
var StringOptionalDefaultType = stringOptionalDefaultType{}

func (t stringOptionalDefaultType) Equal(o attr.Type) bool {
	other, ok := o.(stringOptionalDefaultType)
	if !ok {
		return false
	}

	return t.StringType.Equal(other.StringType)
}

func (stringOptionalDefaultType) String() string {
	return "StringOptionalDefaultType"
}

func (t stringOptionalDefaultType) ValueFromString(_ context.Context, in types.String) (basetypes.StringValuable, diag.Diagnostics) {
	return StringOptionalDefault{StringValue: in}, nil
}

func (t stringOptionalDefaultType) ValueFromTerraform(ctx context.Context, in tftypes.Value) (attr.Value, error) {
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

func (stringOptionalDefaultType) ValueType(context.Context) attr.Value {
	return StringOptionalDefault{}
}

// ---- Value ----

var (
	_ basetypes.StringValuable                   = (*StringOptionalDefault)(nil)
	_ basetypes.StringValuableWithSemanticEquals = (*StringOptionalDefault)(nil)
)

type StringOptionalDefault struct {
	basetypes.StringValue
}

func StringNull() StringOptionalDefault {
	return StringOptionalDefault{StringValue: basetypes.NewStringNull()}
}

func StringUnknown() StringOptionalDefault {
	return StringOptionalDefault{StringValue: basetypes.NewStringUnknown()}
}

func StringValue(value string) StringOptionalDefault {
	return StringOptionalDefault{StringValue: basetypes.NewStringValue(value)}
}

// StringPointerValue builds a value from a *string: nil becomes null, so a
// backend that omits an unset optional field maps straight to null (which is
// in turn treated as equal to "").
func StringPointerValue(value *string) StringOptionalDefault {
	return StringOptionalDefault{StringValue: basetypes.NewStringPointerValue(value)}
}

// Type must return the custom type so the framework does not confuse this value
// with a plain basetypes.StringValue.
func (v StringOptionalDefault) Type(context.Context) attr.Type {
	return StringOptionalDefaultType
}

// Equal compares structural equality. It must assert to the custom type,
// otherwise the embedded StringValue.Equal would reject any StringOptionalDefault.
func (v StringOptionalDefault) Equal(o attr.Value) bool {
	other, ok := o.(StringOptionalDefault)
	if !ok {
		return false
	}

	return v.StringValue.Equal(other.StringValue)
}

// StringSemanticEquals treats null and "" as equivalent (the design decision for
// inconsistent backends), while still reporting a real change between two
// distinct non-empty strings.
func (v StringOptionalDefault) StringSemanticEquals(_ context.Context, newValuable basetypes.StringValuable) (bool, diag.Diagnostics) {
	var diags diag.Diagnostics

	newValue, ok := newValuable.(StringOptionalDefault)
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
	return strings.EqualFold(v.ValueString(), newValue.ValueString()), diags
}
