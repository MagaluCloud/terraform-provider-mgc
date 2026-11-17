package utils

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
)

func TestCIDRValidator(t *testing.T) {
	testCases := []struct {
		name          string
		cidr          string
		expectedValid bool
	}{
		{
			name:          "Valid IPv4 zero CIDR",
			cidr:          "0.0.0.0/0",
			expectedValid: true,
		},
		{
			name:          "Valid IPv6 zero CIDR",
			cidr:          "::/0",
			expectedValid: true,
		},
		{
			name:          "Invalid CIDR - wrong format",
			cidr:          "192.168.1.0",
			expectedValid: false,
		},
		{
			name:          "Invalid CIDR - out of range",
			cidr:          "192.168.1.0/33",
			expectedValid: false,
		},
		{
			name:          "Invalid CIDR - malformed",
			cidr:          "not a cidr",
			expectedValid: false,
		},
		{
			name:          "Empty string",
			cidr:          "",
			expectedValid: false,
		},
		{
			name:          "Valid IPv4 CIDR",
			cidr:          "192.168.1.0/24",
			expectedValid: true,
		},
		{
			name:          "Valid IPv6 CIDR",
			cidr:          "2001:db8:abcd:0012::0/64",
			expectedValid: true,
		},
		{
			name:          "Invalid IPv6 CIDR - wrong format",
			cidr:          "2001:db8:abcd:0012::0",
			expectedValid: false,
		},
		{
			name:          "Invalid IPv6 CIDR - out of range",
			cidr:          "2001:db8:abcd:0012::0/129",
			expectedValid: false,
		},
		{
			name:          "Valid IPv4 CIDR with single IP",
			cidr:          "10.0.0.1/32",
			expectedValid: true,
		},
		{
			name:          "Valid IPv6 CIDR with single IP",
			cidr:          "2001:db8::1/128",
			expectedValid: true,
		},
		{
			name:          "Invalid IPv4 CIDR - malformed",
			cidr:          "10.0.0.1/abc",
			expectedValid: false,
		},
		{
			name:          "Invalid IPv6 CIDR - malformed",
			cidr:          "2001:db8::1/xyz",
			expectedValid: false,
		},
		{
			name:          "Valid IPv4 CIDR",
			cidr:          "192.168.1.0/24",
			expectedValid: true,
		},
		{
			name:          "Valid IPv6 CIDR",
			cidr:          "2001:db8::/32",
			expectedValid: true,
		},
		{
			name:          "Valid IPv4 zero CIDR",
			cidr:          "0.0.0.0/0",
			expectedValid: true,
		},
		{
			name:          "Valid IPv6 zero CIDR",
			cidr:          "::/0",
			expectedValid: true,
		},
		{
			name:          "Invalid CIDR - wrong format",
			cidr:          "192.168.1.0",
			expectedValid: false,
		},
		{
			name:          "Invalid CIDR - out of range",
			cidr:          "192.168.1.0/33",
			expectedValid: false,
		},
		{
			name:          "Invalid CIDR - malformed",
			cidr:          "not a cidr",
			expectedValid: false,
		},
		{
			name:          "Empty string",
			cidr:          "",
			expectedValid: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			val := CidrValidator{}
			req := validator.StringRequest{
				Path:        path.Root("test"),
				ConfigValue: types.StringValue(tc.cidr),
			}
			resp := &validator.StringResponse{
				Diagnostics: diag.Diagnostics{},
			}

			val.ValidateString(context.Background(), req, resp)

			if tc.expectedValid {
				assert.Empty(t, resp.Diagnostics, "Expected no diagnostics for valid CIDR")
			} else {
				assert.NotEmpty(t, resp.Diagnostics, "Expected diagnostics for invalid CIDR")
			}
		})
	}
}

func TestAvailabilityZoneValidator(t *testing.T) {
	testCases := []struct {
		name          string
		zone          types.String
		expectedValid bool
	}{
		{name: "Valid zone", zone: types.StringValue("br-se1-a"), expectedValid: true},
		{name: "Unsupported zone", zone: types.StringValue("br-ne1-c"), expectedValid: false},
		{name: "Region without zone", zone: types.StringValue("br-se1"), expectedValid: false},
		{name: "Wrong case", zone: types.StringValue("BR-SE1-A"), expectedValid: false},
		{name: "Empty string", zone: types.StringValue(""), expectedValid: false},
		{name: "Null value", zone: types.StringNull(), expectedValid: true},
		{name: "Unknown value", zone: types.StringUnknown(), expectedValid: true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := validator.StringRequest{
				Path:        path.Root("availability_zone"),
				ConfigValue: tc.zone,
			}
			resp := &validator.StringResponse{
				Diagnostics: diag.Diagnostics{},
			}

			AvailabilityZoneValidator().ValidateString(context.Background(), req, resp)

			if tc.expectedValid {
				assert.Empty(t, resp.Diagnostics, "Expected no diagnostics for valid availability zone")
			} else {
				assert.NotEmpty(t, resp.Diagnostics, "Expected diagnostics for invalid availability zone")
			}
		})
	}
}

func TestCIDRValidatorDescription(t *testing.T) {
	validator := CidrValidator{}
	description := validator.Description(context.Background())
	assert.Equal(t, "value must be a valid CIDR notation", description)
}

func TestCIDRValidatorMarkdownDescription(t *testing.T) {
	validator := CidrValidator{}
	description := validator.MarkdownDescription(context.Background())
	assert.Equal(t, "value must be a valid CIDR notation", description)
}

func TestCIDRValidatorNullValue(t *testing.T) {
	val := CidrValidator{}
	req := validator.StringRequest{
		Path:        path.Root("test"),
		ConfigValue: types.StringNull(),
	}
	resp := &validator.StringResponse{
		Diagnostics: diag.Diagnostics{},
	}

	val.ValidateString(context.Background(), req, resp)
	assert.Empty(t, resp.Diagnostics, "Expected no diagnostics for null value")
}

func TestCIDRValidatorUnknownValue(t *testing.T) {
	val := CidrValidator{}
	req := validator.StringRequest{
		Path:        path.Root("test"),
		ConfigValue: types.StringUnknown(),
	}
	resp := &validator.StringResponse{
		Diagnostics: diag.Diagnostics{},
	}

	val.ValidateString(context.Background(), req, resp)
	assert.Empty(t, resp.Diagnostics, "Expected no diagnostics for unknown value")
}
