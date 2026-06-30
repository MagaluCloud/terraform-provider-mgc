package types

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// ---- Type ----

var _ basetypes.ListTypable = (*listOptionalDefaultType)(nil)

// listOptionalDefaultType is a custom list type whose values treat a null list
// and an empty list ([]) as semantically equal, the list counterpart of
// StringOptionalDefaultType. This avoids spurious drift when a backend reports
// an unset optional collection inconsistently (some return null, some return []).
type listOptionalDefaultType struct {
	basetypes.ListType
}

// NewListOptionalDefaultType builds the type for a given element type, to assign
// to a schema attribute's CustomType field (do not also set ElementType).
func NewListOptionalDefaultType(elemType attr.Type) listOptionalDefaultType {
	return listOptionalDefaultType{basetypes.ListType{ElemType: elemType}}
}

func (t listOptionalDefaultType) Equal(o attr.Type) bool {
	other, ok := o.(listOptionalDefaultType)
	if !ok {
		return false
	}
	return t.ListType.Equal(other.ListType)
}

func (listOptionalDefaultType) String() string {
	return "ListOptionalDefaultType"
}

func (t listOptionalDefaultType) ValueFromList(_ context.Context, in basetypes.ListValue) (basetypes.ListValuable, diag.Diagnostics) {
	return ListOptionalDefault{ListValue: in}, nil
}

func (t listOptionalDefaultType) ValueFromTerraform(ctx context.Context, in tftypes.Value) (attr.Value, error) {
	attrValue, err := t.ListType.ValueFromTerraform(ctx, in)
	if err != nil {
		return nil, err
	}

	listValue, ok := attrValue.(basetypes.ListValue)
	if !ok {
		return nil, fmt.Errorf("unexpected value type of %T", attrValue)
	}
	listValuable, diags := t.ValueFromList(ctx, listValue)
	if diags.HasError() {
		return nil, fmt.Errorf("unexpected error converting ListValue to ListValuable: %v", diags)
	}
	return listValuable, nil
}

func (t listOptionalDefaultType) ValueType(context.Context) attr.Value {
	return ListOptionalDefault{ListValue: basetypes.NewListNull(t.ElemType)}
}

// ---- Value ----

var (
	_ basetypes.ListValuable                   = (*ListOptionalDefault)(nil)
	_ basetypes.ListValuableWithSemanticEquals = (*ListOptionalDefault)(nil)
)

type ListOptionalDefault struct {
	basetypes.ListValue
}

// ListOptionalDefaultNull returns a null value for the given element type.
func ListOptionalDefaultNull(elemType attr.Type) ListOptionalDefault {
	return ListOptionalDefault{ListValue: basetypes.NewListNull(elemType)}
}

// ListOptionalDefaultValueFrom builds a value from a Go slice (or any value the
// framework can reflect into a list), e.g. a []string of CIDRs.
func ListOptionalDefaultValueFrom(ctx context.Context, elemType attr.Type, elements any) (ListOptionalDefault, diag.Diagnostics) {
	listValue, diags := basetypes.NewListValueFrom(ctx, elemType, elements)
	return ListOptionalDefault{ListValue: listValue}, diags
}

// Type must return the custom type so the framework does not confuse this value
// with a plain basetypes.ListValue.
func (v ListOptionalDefault) Type(ctx context.Context) attr.Type {
	return listOptionalDefaultType{basetypes.ListType{ElemType: v.ElementType(ctx)}}
}

// Equal compares structural equality. It must assert to the custom type,
// otherwise the embedded ListValue.Equal would reject any ListOptionalDefault.
func (v ListOptionalDefault) Equal(o attr.Value) bool {
	other, ok := o.(ListOptionalDefault)
	if !ok {
		return false
	}
	return v.ListValue.Equal(other.ListValue)
}

// ListSemanticEquals treats null and the empty list as equivalent, while still
// reporting a real change between two different non-empty lists (order matters,
// as it does for a Terraform list).
func (v ListOptionalDefault) ListSemanticEquals(_ context.Context, newValuable basetypes.ListValuable) (bool, diag.Diagnostics) {
	var diags diag.Diagnostics

	newValue, ok := newValuable.(ListOptionalDefault)
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

	// null and [] both count as "empty" and are therefore equal to each other.
	vEmpty := v.IsNull() || len(v.Elements()) == 0
	newEmpty := newValue.IsNull() || len(newValue.Elements()) == 0
	if vEmpty || newEmpty {
		return vEmpty && newEmpty, diags
	}

	return v.ListValue.Equal(newValue.ListValue), diags
}
