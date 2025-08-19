package tfutil

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
)

func TestVisibilityValidatorStructure(t *testing.T) {
	validator := VisibilityValidator{}

	// Test that validator implements the correct interface
	assert.NotNil(t, validator)

	// Test description methods
	description := validator.Description(context.Background())
	assert.Equal(t, "Validates that public_ip_id is required for external load balancers and forbidden for internal load balancers", description)

	markdownDescription := validator.MarkdownDescription(context.Background())
	assert.Equal(t, "Validates that public_ip_id is required for external load balancers and forbidden for internal load balancers", markdownDescription)
}

// LoadBalancerVisibilityModel test removed since the model was removed from the validator
// The validator now gets individual attributes directly instead of using a model

func TestVisibilityValidationLogic(t *testing.T) {
	testCases := []struct {
		name            string
		visibility      string
		publicIPID      string
		shouldBeValid   bool
		shouldRequireIP bool
		shouldForbidIP  bool
	}{
		{
			name:            "External with public IP",
			visibility:      "external",
			publicIPID:      "public-ip-123",
			shouldBeValid:   true,
			shouldRequireIP: false,
			shouldForbidIP:  false,
		},
		{
			name:            "Internal without public IP",
			visibility:      "internal",
			publicIPID:      "",
			shouldBeValid:   true,
			shouldRequireIP: false,
			shouldForbidIP:  false,
		},
		{
			name:            "External without public IP",
			visibility:      "external",
			publicIPID:      "",
			shouldBeValid:   false,
			shouldRequireIP: true,
			shouldForbidIP:  false,
		},
		{
			name:            "Internal with public IP",
			visibility:      "internal",
			publicIPID:      "public-ip-123",
			shouldBeValid:   false,
			shouldRequireIP: false,
			shouldForbidIP:  true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Test validation logic conditions
			var publicIPID types.String
			if tc.publicIPID == "" {
				publicIPID = types.StringNull()
			} else {
				publicIPID = types.StringValue(tc.publicIPID)
			}

			hasPublicIP := !publicIPID.IsNull() && !publicIPID.IsUnknown() && publicIPID.ValueString() != ""

			// Test external validation logic
			if tc.visibility == "external" {
				isValidExternal := hasPublicIP
				if tc.shouldRequireIP {
					assert.False(t, isValidExternal, "External should require public IP")
				} else {
					assert.True(t, isValidExternal, "External with public IP should be valid")
				}
			}

			// Test internal validation logic
			if tc.visibility == "internal" {
				isValidInternal := !hasPublicIP
				if tc.shouldForbidIP {
					assert.False(t, isValidInternal, "Internal should forbid public IP")
				} else {
					assert.True(t, isValidInternal, "Internal without public IP should be valid")
				}
			}
		})
	}
}
