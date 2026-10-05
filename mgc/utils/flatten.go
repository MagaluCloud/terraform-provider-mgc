package utils

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func FlattenStringValue(tfValue types.String, str *string) types.String {
	if tfValue.IsUnknown() || !(IsNilOrZeroValue(tfValue.ValueStringPointer()) && IsNilOrZeroValue(str)) {
		return types.StringPointerValue(str)
	}
	return tfValue
}

func FlattenTypeSetStringArray(tfValue types.Set, arr *[]string) types.Set {
	tfValueStrArr := ConvertTypeSetToStringArray(tfValue)

	if tfValue.IsUnknown() || !(IsSliceNilOrEmpty(tfValueStrArr) && IsSliceNilOrEmpty(arr)) {
		return StringSliceToTypesSet(arr)
	}

	return tfValue
}
