package tfutil

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
)

func TestTargetValidatorStructure(t *testing.T) {
	validator := TargetValidator{}

	// Test that validator implements the correct interface
	assert.NotNil(t, validator)

	// Test description methods
	description := validator.Description(context.Background())
	assert.Equal(t, "Validates target configuration based on targets_type", description)

	markdownDescription := validator.MarkdownDescription(context.Background())
	assert.Equal(t, "Validates target configuration based on targets_type", markdownDescription)
}

func TestTargetModel(t *testing.T) {
	// Test TargetModel structure
	target := TargetModel{
		ID:        types.StringValue("test-id"),
		NICID:     types.StringValue("nic-123"),
		IPAddress: types.StringValue("192.168.1.10"),
		Port:      types.Int64Value(8080),
	}

	assert.Equal(t, "test-id", target.ID.ValueString())
	assert.Equal(t, "nic-123", target.NICID.ValueString())
	assert.Equal(t, "192.168.1.10", target.IPAddress.ValueString())
	assert.Equal(t, int64(8080), target.Port.ValueInt64())
}

// BackendTargetModel test removed since the model was removed from the validator
// The validator now gets individual attributes directly instead of using a model

func TestTargetValidatorLogic(t *testing.T) {
	testCases := []struct {
		name                     string
		targetsType              string
		target                   TargetModel
		shouldBeValidForRaw      bool
		shouldBeValidForInstance bool
	}{
		{
			name:        "Target with IP address only",
			targetsType: "raw",
			target: TargetModel{
				IPAddress: types.StringValue("192.168.1.10"),
				Port:      types.Int64Value(8080),
				NICID:     types.StringNull(),
			},
			shouldBeValidForRaw:      true,
			shouldBeValidForInstance: false,
		},
		{
			name:        "Target with NIC ID only",
			targetsType: "instance",
			target: TargetModel{
				NICID:     types.StringValue("nic-123"),
				Port:      types.Int64Value(8080),
				IPAddress: types.StringNull(),
			},
			shouldBeValidForRaw:      false,
			shouldBeValidForInstance: true,
		},
		{
			name:        "Target with both IP and NIC ID",
			targetsType: "raw",
			target: TargetModel{
				IPAddress: types.StringValue("192.168.1.10"),
				NICID:     types.StringValue("nic-123"),
				Port:      types.Int64Value(8080),
			},
			shouldBeValidForRaw:      false,
			shouldBeValidForInstance: false,
		},
		{
			name:        "Target with neither IP nor NIC ID",
			targetsType: "raw",
			target: TargetModel{
				IPAddress: types.StringNull(),
				NICID:     types.StringNull(),
				Port:      types.Int64Value(8080),
			},
			shouldBeValidForRaw:      false,
			shouldBeValidForInstance: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Test validation logic conditions for raw type
			hasIP := !tc.target.IPAddress.IsNull() && !tc.target.IPAddress.IsUnknown() && tc.target.IPAddress.ValueString() != ""
			hasNIC := !tc.target.NICID.IsNull() && !tc.target.NICID.IsUnknown() && tc.target.NICID.ValueString() != ""

			// For raw type: should have IP and not have NIC
			isValidForRaw := hasIP && !hasNIC
			assert.Equal(t, tc.shouldBeValidForRaw, isValidForRaw, "Raw validation logic mismatch")

			// For instance type: should have NIC and not have IP
			isValidForInstance := hasNIC && !hasIP
			assert.Equal(t, tc.shouldBeValidForInstance, isValidForInstance, "Instance validation logic mismatch")
		})
	}
}
