package testutils

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func ObjectWith(ctx context.Context, typ attr.Type, values map[string]tftypes.Value) tftypes.Value {
	obj := typ.TerraformType(ctx).(tftypes.Object)
	attrs := make(map[string]tftypes.Value, len(obj.AttributeTypes))
	for name, t := range obj.AttributeTypes {
		if v, ok := values[name]; ok {
			attrs[name] = v
			continue
		}
		attrs[name] = tftypes.NewValue(t, nil)
	}
	return tftypes.NewValue(obj, attrs)
}

func OptionalString(s string) tftypes.Value {
	if s == "" {
		return tftypes.NewValue(tftypes.String, nil)
	}
	return tftypes.NewValue(tftypes.String, s)
}
