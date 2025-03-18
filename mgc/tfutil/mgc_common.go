package tfutil

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var azRegex = regexp.MustCompile(`^([a-z]{2})-([a-z]{2})([1-9])[-]([a-z])$`)

type GenericIDNameModel struct {
	Name types.String `tfsdk:"name"`
	ID   types.String `tfsdk:"id"`
}

type GenericIDModel struct {
	ID types.String `tfsdk:"id"`
}

func ConvertTimeToRFC3339(t *time.Time) *string {
	if t == nil {
		return nil
	}
	result := new(string)
	*result = t.Format(time.RFC3339)
	return result
}

func ConvertInt64PointerToIntPointer(int64Ptr *int64) *int {
	if int64Ptr == nil {
		return nil
	}
	intVal := int(*int64Ptr)
	return &intVal
}

func ConvertIntPointerToInt64Pointer(intPtr *int) *int64 {
	if intPtr == nil {
		return nil
	}
	int64Val := int64(*intPtr)
	return &int64Val
}

type ResponseFrom interface {
	resource.ConfigureResponse | datasource.ConfigureResponse
}

func SdkParamValueToString(v any) string {
	if v == nil {
		return ""
	}

	switch val := v.(type) {
	case *string:
		if val != nil {
			return *val
		}
	case *float64:
		if val != nil {
			return fmt.Sprintf("%g", *val)
		}
	case *int:
		if val != nil {
			return fmt.Sprint(*val)
		}
	case *bool:
		if val != nil {
			return fmt.Sprint(*val)
		}
	case string:
		return val
	case float64:
		return fmt.Sprintf("%g", val)
	case int:
		return fmt.Sprint(val)
	case bool:
		return fmt.Sprint(val)
	}

	return ""
}

func ConvertXZoneToAvailabilityZone(region, xZone string) string {
	return strings.ToLower(fmt.Sprintf("%s-%s", region, xZone))
}

func ConvertAvailabilityZoneToXZone(availabilityZone string) (string, error) {
	matches := azRegex.FindStringSubmatch(availabilityZone)
	if len(matches) != 5 {
		return "", fmt.Errorf("invalid availability zone format: %s", availabilityZone)
	}
	return matches[4], nil
}
