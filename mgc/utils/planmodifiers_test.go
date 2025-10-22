package utils

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/stretchr/testify/assert"
)

func TestReplaceIfChangeAndNotIsNotSetOnPlan_PlanModifyString(t *testing.T) {
	tests := []struct {
		name            string
		stateValue      types.String
		planValue       types.String
		expectedReplace bool
	}{
		{
			name:            "plan_value_unknown",
			stateValue:      types.StringValue("zone-1"),
			planValue:       types.StringUnknown(),
			expectedReplace: false,
		},
		{
			name:            "values_different",
			stateValue:      types.StringValue("zone-1"),
			planValue:       types.StringValue("zone-2"),
			expectedReplace: true,
		},
		{
			name:            "values_equal",
			stateValue:      types.StringValue("zone-1"),
			planValue:       types.StringValue("zone-1"),
			expectedReplace: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			modifier := ReplaceIfChangeAndNotIsNotSetOnPlan{}
			request := planmodifier.StringRequest{
				StateValue: tt.stateValue,
				PlanValue:  tt.planValue,
			}
			response := &planmodifier.StringResponse{}

			modifier.PlanModifyString(context.Background(), request, response)

			assert.Equal(t, tt.expectedReplace, response.RequiresReplace)
		})
	}
}

func TestReplaceIfChangeAndNotIsNotSetOnPlan_Description(t *testing.T) {
	modifier := ReplaceIfChangeAndNotIsNotSetOnPlan{}
	expected := "Requires replace if the value is different from the state and the plan value is not unknown."

	assert.Equal(t, expected, modifier.Description(context.Background()))
}

func TestReplaceIfChangeAndNotIsNotSetOnPlan_MarkdownDescription(t *testing.T) {
	modifier := ReplaceIfChangeAndNotIsNotSetOnPlan{}
	expected := "Requires replace if the value is different from the state and the plan value is not unknown."

	assert.Equal(t, expected, modifier.MarkdownDescription(context.Background()))
}

func TestSetMembershipChangeRequiresReplace_CountDifferent(t *testing.T) {
	modifier := SetMembershipChangeRequiresReplace()

	req := planmodifier.SetRequest{
		StateValue: makeStringSet(t, "a"),
		PlanValue:  makeStringSet(t, "a", "b"),
	}
	resp := &planmodifier.SetResponse{}

	planCount, diags := countSetElementsGeneric(context.Background(), req.PlanValue)
	assert.False(t, diags.HasError())
	assert.Equal(t, 2, planCount)
	stateCount, diags := countSetElementsGeneric(context.Background(), req.StateValue)
	assert.False(t, diags.HasError())
	assert.Equal(t, 1, stateCount)

	modifier.PlanModifySet(context.Background(), req, resp)

	assert.False(t, resp.Diagnostics.HasError())
	assert.True(t, resp.RequiresReplace)
}

func TestSetMembershipChangeRequiresReplace_SameCountNoReplace(t *testing.T) {
	modifier := SetMembershipChangeRequiresReplace()

	req := planmodifier.SetRequest{
		StateValue: makeStringSet(t, "a", "b"),
		PlanValue:  makeStringSet(t, "c", "d"),
	}
	resp := &planmodifier.SetResponse{}

	planCount, diags := countSetElementsGeneric(context.Background(), req.PlanValue)
	assert.False(t, diags.HasError())
	assert.Equal(t, 2, planCount)
	stateCount, diags := countSetElementsGeneric(context.Background(), req.StateValue)
	assert.False(t, diags.HasError())
	assert.Equal(t, 2, stateCount)

	modifier.PlanModifySet(context.Background(), req, resp)

	assert.False(t, resp.Diagnostics.HasError())
	assert.False(t, resp.RequiresReplace)
}

func TestSetMembershipChangeRequiresReplace_UnknownPlanOrState(t *testing.T) {
	modifier := SetMembershipChangeRequiresReplace()

	{
		req := planmodifier.SetRequest{
			StateValue: makeStringSet(t, "a"),
			PlanValue:  types.SetUnknown(types.StringType),
		}
		resp := &planmodifier.SetResponse{}
		modifier.PlanModifySet(context.Background(), req, resp)
		assert.False(t, resp.RequiresReplace)
	}

	{
		req := planmodifier.SetRequest{
			StateValue: types.SetUnknown(types.StringType),
			PlanValue:  makeStringSet(t, "a"),
		}
		resp := &planmodifier.SetResponse{}
		modifier.PlanModifySet(context.Background(), req, resp)
		assert.False(t, resp.RequiresReplace)
	}
}

func TestSetMembershipChangeRequiresReplace_NullPlanOrState(t *testing.T) {
	modifier := SetMembershipChangeRequiresReplace()

	{
		req := planmodifier.SetRequest{
			StateValue: makeStringSet(t, "a"),
			PlanValue:  types.SetNull(types.StringType),
		}
		resp := &planmodifier.SetResponse{}
		modifier.PlanModifySet(context.Background(), req, resp)
		assert.False(t, resp.RequiresReplace)
	}

	{
		req := planmodifier.SetRequest{
			StateValue: types.SetNull(types.StringType),
			PlanValue:  makeStringSet(t, "a"),
		}
		resp := &planmodifier.SetResponse{}
		modifier.PlanModifySet(context.Background(), req, resp)
		assert.False(t, resp.RequiresReplace)
	}
}

func TestSetMembershipChangeRequiresReplace_Description(t *testing.T) {
	modifier := SetMembershipChangeRequiresReplace()
	expected := "Requires replacement if the set membership changes"
	assert.Equal(t, expected, modifier.Description(context.Background()))
}

func TestSetMembershipChangeRequiresReplace_MarkdownDescription(t *testing.T) {
	modifier := SetMembershipChangeRequiresReplace()
	expected := "Requires replacement if the set membership changes"
	assert.Equal(t, expected, modifier.MarkdownDescription(context.Background()))
}

func makeStringSet(t *testing.T, values ...string) types.Set {
	t.Helper()
	elems := make([]attr.Value, 0, len(values))
	for _, v := range values {
		elems = append(elems, types.StringValue(v))
	}
	return types.SetValueMust(types.StringType, elems)
}

func TestRequireReplacePlanModifier_Description(t *testing.T) {
	tests := []struct {
		name         string
		modifier     RequireReplacePlanModifier
		expectedDesc string
	}{
		{
			name:         "without UseStateForUnknown",
			modifier:     RequireReplacePlanModifier{UseStateForUnknown: false},
			expectedDesc: "Requires resource replacement when value changes.",
		},
		{
			name:         "with UseStateForUnknown",
			modifier:     RequireReplacePlanModifier{UseStateForUnknown: true},
			expectedDesc: "Requires resource replacement when value changes and uses state value when plan value is unknown.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			desc := tt.modifier.Description(ctx)
			assert.Equal(t, tt.expectedDesc, desc)

			markdownDesc := tt.modifier.MarkdownDescription(ctx)
			assert.Equal(t, tt.expectedDesc, markdownDesc)
		})
	}
}

func TestRequireReplace(t *testing.T) {
	modifier := RequireReplace()
	reqModifier, ok := modifier.(RequireReplacePlanModifier)
	assert.True(t, ok, "RequireReplace() should return RequireReplacePlanModifier")
	assert.False(t, reqModifier.UseStateForUnknown, "RequireReplace() should not use state for unknown")
}

func TestRequireReplaceWithStateForUnknown(t *testing.T) {
	modifier := RequireReplaceWithStateForUnknown()
	reqModifier, ok := modifier.(RequireReplacePlanModifier)
	assert.True(t, ok, "RequireReplaceWithStateForUnknown() should return RequireReplacePlanModifier")
	assert.True(t, reqModifier.UseStateForUnknown, "RequireReplaceWithStateForUnknown() should use state for unknown")
}

func TestRequireReplacePlanModifier_PlanModifyString(t *testing.T) {
	tests := []struct {
		name                    string
		useStateForUnknown      bool
		stateValue              types.String
		planValue               types.String
		configValue             types.String
		expectedPlanValue       types.String
		expectedRequiresReplace bool
		planRaw                 *tftypes.Value
	}{
		{
			name:                    "no change - same value",
			useStateForUnknown:      false,
			stateValue:              types.StringValue("value-a"),
			planValue:               types.StringValue("value-a"),
			configValue:             types.StringValue("value-a"),
			expectedPlanValue:       types.StringValue("value-a"),
			expectedRequiresReplace: false,
		},
		{
			name:                    "change value - requires replace",
			useStateForUnknown:      false,
			stateValue:              types.StringValue("value-a"),
			planValue:               types.StringValue("value-b"),
			configValue:             types.StringValue("value-b"),
			expectedPlanValue:       types.StringValue("value-b"),
			expectedRequiresReplace: true,
		},
		{
			name:                    "new resource - no replace",
			useStateForUnknown:      false,
			stateValue:              types.StringNull(),
			planValue:               types.StringValue("value-a"),
			configValue:             types.StringValue("value-a"),
			expectedPlanValue:       types.StringValue("value-a"),
			expectedRequiresReplace: false,
		},
		{
			name:                    "remove value - requires replace",
			useStateForUnknown:      false,
			stateValue:              types.StringValue("value-a"),
			planValue:               types.StringNull(),
			configValue:             types.StringNull(),
			expectedPlanValue:       types.StringNull(),
			expectedRequiresReplace: true,
		},
		{
			name:                    "unknown plan value without UseStateForUnknown",
			useStateForUnknown:      false,
			stateValue:              types.StringValue("value-a"),
			planValue:               types.StringUnknown(),
			configValue:             types.StringUnknown(),
			expectedPlanValue:       types.StringUnknown(),
			expectedRequiresReplace: false,
		},
		{
			name:                    "unknown plan value with UseStateForUnknown",
			useStateForUnknown:      true,
			stateValue:              types.StringValue("value-a"),
			planValue:               types.StringUnknown(),
			configValue:             types.StringUnknown(),
			expectedPlanValue:       types.StringValue("value-a"),
			expectedRequiresReplace: false,
		},
		{
			name:                    "unknown plan value with UseStateForUnknown but null state",
			useStateForUnknown:      true,
			stateValue:              types.StringNull(),
			planValue:               types.StringUnknown(),
			configValue:             types.StringUnknown(),
			expectedPlanValue:       types.StringUnknown(),
			expectedRequiresReplace: false,
		},
		{
			name:                    "unknown plan value with UseStateForUnknown but unknown state",
			useStateForUnknown:      true,
			stateValue:              types.StringUnknown(),
			planValue:               types.StringUnknown(),
			configValue:             types.StringUnknown(),
			expectedPlanValue:       types.StringUnknown(),
			expectedRequiresReplace: false,
		},
		{
			name:                    "unknown state value - no replace",
			useStateForUnknown:      false,
			stateValue:              types.StringUnknown(),
			planValue:               types.StringValue("value-a"),
			configValue:             types.StringValue("value-a"),
			expectedPlanValue:       types.StringValue("value-a"),
			expectedRequiresReplace: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			modifier := RequireReplacePlanModifier{
				UseStateForUnknown: tt.useStateForUnknown,
			}

			planRaw := tt.planRaw
			if planRaw == nil {
				if tt.planValue.IsNull() {
					nullValue := tftypes.NewValue(tftypes.String, nil)
					planRaw = &nullValue
				} else if tt.planValue.IsUnknown() {
					unknownValue := tftypes.NewValue(tftypes.String, tftypes.UnknownValue)
					planRaw = &unknownValue
				} else {
					value := tftypes.NewValue(tftypes.String, tt.planValue.ValueString())
					planRaw = &value
				}
			}

			req := planmodifier.StringRequest{
				StateValue:  tt.stateValue,
				PlanValue:   tt.planValue,
				ConfigValue: tt.configValue,
				Plan: tfsdk.Plan{
					Raw: *planRaw,
				},
			}

			resp := &planmodifier.StringResponse{
				PlanValue: tt.planValue,
			}

			modifier.PlanModifyString(ctx, req, resp)

			assert.True(t, resp.PlanValue.Equal(tt.expectedPlanValue), "PlanValue = %v, want %v", resp.PlanValue, tt.expectedPlanValue)
			assert.Equal(t, tt.expectedRequiresReplace, resp.RequiresReplace, "RequiresReplace = %v, want %v", resp.RequiresReplace, tt.expectedRequiresReplace)
		})
	}
}

func TestRequireReplacePlanModifier_PlanModifyString_DestroyPlan(t *testing.T) {
	ctx := context.Background()
	modifier := RequireReplacePlanModifier{
		UseStateForUnknown: true,
	}

	objType := tftypes.Object{
		AttributeTypes: map[string]tftypes.Type{
			"test": tftypes.String,
		},
	}
	nullPlan := tftypes.NewValue(objType, nil)

	req := planmodifier.StringRequest{
		StateValue:  types.StringValue("value-a"),
		PlanValue:   types.StringValue("value-b"),
		ConfigValue: types.StringValue("value-b"),
		Plan: tfsdk.Plan{
			Raw: nullPlan,
		},
	}

	resp := &planmodifier.StringResponse{
		PlanValue: types.StringValue("value-b"),
	}

	modifier.PlanModifyString(ctx, req, resp)

	assert.False(t, resp.RequiresReplace, "RequiresReplace should be false when destroying resource")
	assert.True(t, resp.PlanValue.Equal(types.StringValue("value-b")), "PlanValue = %v, want %v", resp.PlanValue, types.StringValue("value-b"))
}

func TestRequireReplacePlanModifier_IntegrationWithFramework(t *testing.T) {
	var _ planmodifier.String = RequireReplace()
	var _ planmodifier.String = RequireReplaceWithStateForUnknown()
	var _ planmodifier.String = RequireReplacePlanModifier{}
}

func BenchmarkRequireReplacePlanModifier_PlanModifyString(b *testing.B) {
	ctx := context.Background()
	modifier := RequireReplacePlanModifier{
		UseStateForUnknown: true,
	}

	req := planmodifier.StringRequest{
		StateValue:  types.StringValue("value-a"),
		PlanValue:   types.StringValue("value-b"),
		ConfigValue: types.StringValue("value-b"),
		Plan: tfsdk.Plan{
			Raw: tftypes.NewValue(tftypes.String, "value-b"),
		},
	}

	resp := &planmodifier.StringResponse{
		PlanValue: types.StringValue("value-b"),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		modifier.PlanModifyString(ctx, req, resp)
	}
}

func TestSetRequiresReplaceOnChange(t *testing.T) {
	testCases := []struct {
		name          string
		stateValue    types.Set
		planValue     types.Set
		planRaw       tftypes.Value
		expectReplace bool
		expectError   bool
		description   string
	}{
		{
			name:          "null state, non-null plan - no replacement (creation)",
			stateValue:    types.SetNull(types.StringType),
			planValue:     types.SetValueMust(types.StringType, []attr.Value{types.StringValue("zone-a")}),
			planRaw:       tftypes.NewValue(tftypes.Set{ElementType: tftypes.String}, []tftypes.Value{tftypes.NewValue(tftypes.String, "zone-a")}),
			expectReplace: false,
			description:   "Resource creation should not require replacement",
		},
		{
			name:          "unknown state, non-null plan - no replacement",
			stateValue:    types.SetUnknown(types.StringType),
			planValue:     types.SetValueMust(types.StringType, []attr.Value{types.StringValue("zone-a")}),
			planRaw:       tftypes.NewValue(tftypes.Set{ElementType: tftypes.String}, []tftypes.Value{tftypes.NewValue(tftypes.String, "zone-a")}),
			expectReplace: false,
			description:   "Unknown state should not require replacement",
		},
		{
			name:          "non-null state, null plan - requires replacement (deletion)",
			stateValue:    types.SetValueMust(types.StringType, []attr.Value{types.StringValue("zone-a")}),
			planValue:     types.SetNull(types.StringType),
			planRaw:       tftypes.NewValue(tftypes.Set{ElementType: tftypes.String}, nil),
			expectReplace: true,
			description:   "Removing all availability zones should require replacement",
		},
		{
			name:          "unknown plan value - no replacement",
			stateValue:    types.SetValueMust(types.StringType, []attr.Value{types.StringValue("zone-a")}),
			planValue:     types.SetUnknown(types.StringType),
			planRaw:       tftypes.NewValue(tftypes.Set{ElementType: tftypes.String}, tftypes.UnknownValue),
			expectReplace: false,
			description:   "Unknown plan value should not require replacement",
		},
		{
			name:          "plan is being destroyed - no replacement",
			stateValue:    types.SetValueMust(types.StringType, []attr.Value{types.StringValue("zone-a")}),
			planValue:     types.SetValueMust(types.StringType, []attr.Value{types.StringValue("zone-b")}),
			planRaw:       tftypes.NewValue(tftypes.Set{ElementType: tftypes.String}, nil),
			expectReplace: false,
			description:   "Resource destruction should not require replacement",
		},
		{
			name:          "identical sets - no replacement",
			stateValue:    types.SetValueMust(types.StringType, []attr.Value{types.StringValue("zone-a"), types.StringValue("zone-b")}),
			planValue:     types.SetValueMust(types.StringType, []attr.Value{types.StringValue("zone-a"), types.StringValue("zone-b")}),
			planRaw:       tftypes.NewValue(tftypes.Set{ElementType: tftypes.String}, []tftypes.Value{tftypes.NewValue(tftypes.String, "zone-a"), tftypes.NewValue(tftypes.String, "zone-b")}),
			expectReplace: false,
			description:   "Identical sets should not require replacement",
		},
		{
			name:          "identical sets different order - no replacement",
			stateValue:    types.SetValueMust(types.StringType, []attr.Value{types.StringValue("zone-a"), types.StringValue("zone-b")}),
			planValue:     types.SetValueMust(types.StringType, []attr.Value{types.StringValue("zone-b"), types.StringValue("zone-a")}),
			planRaw:       tftypes.NewValue(tftypes.Set{ElementType: tftypes.String}, []tftypes.Value{tftypes.NewValue(tftypes.String, "zone-b"), tftypes.NewValue(tftypes.String, "zone-a")}),
			expectReplace: false,
			description:   "Sets with same elements in different order should not require replacement",
		},
		{
			name:          "element added - requires replacement",
			stateValue:    types.SetValueMust(types.StringType, []attr.Value{types.StringValue("zone-a")}),
			planValue:     types.SetValueMust(types.StringType, []attr.Value{types.StringValue("zone-a"), types.StringValue("zone-b")}),
			planRaw:       tftypes.NewValue(tftypes.Set{ElementType: tftypes.String}, []tftypes.Value{tftypes.NewValue(tftypes.String, "zone-a"), tftypes.NewValue(tftypes.String, "zone-b")}),
			expectReplace: true,
			description:   "Adding availability zones should require replacement",
		},
		{
			name:          "element removed - requires replacement",
			stateValue:    types.SetValueMust(types.StringType, []attr.Value{types.StringValue("zone-a"), types.StringValue("zone-b")}),
			planValue:     types.SetValueMust(types.StringType, []attr.Value{types.StringValue("zone-a")}),
			planRaw:       tftypes.NewValue(tftypes.Set{ElementType: tftypes.String}, []tftypes.Value{tftypes.NewValue(tftypes.String, "zone-a")}),
			expectReplace: true,
			description:   "Removing availability zones should require replacement",
		},
		{
			name:          "element changed - requires replacement",
			stateValue:    types.SetValueMust(types.StringType, []attr.Value{types.StringValue("zone-a")}),
			planValue:     types.SetValueMust(types.StringType, []attr.Value{types.StringValue("zone-b")}),
			planRaw:       tftypes.NewValue(tftypes.Set{ElementType: tftypes.String}, []tftypes.Value{tftypes.NewValue(tftypes.String, "zone-b")}),
			expectReplace: true,
			description:   "Changing availability zones should require replacement",
		},
		{
			name:          "multiple elements changed - requires replacement",
			stateValue:    types.SetValueMust(types.StringType, []attr.Value{types.StringValue("zone-a"), types.StringValue("zone-b")}),
			planValue:     types.SetValueMust(types.StringType, []attr.Value{types.StringValue("zone-c"), types.StringValue("zone-d")}),
			planRaw:       tftypes.NewValue(tftypes.Set{ElementType: tftypes.String}, []tftypes.Value{tftypes.NewValue(tftypes.String, "zone-c"), tftypes.NewValue(tftypes.String, "zone-d")}),
			expectReplace: true,
			description:   "Changing all availability zones should require replacement",
		},
		{
			name:          "empty to non-empty - no replacement (creation case)",
			stateValue:    types.SetValueMust(types.StringType, []attr.Value{}),
			planValue:     types.SetValueMust(types.StringType, []attr.Value{types.StringValue("zone-a")}),
			planRaw:       tftypes.NewValue(tftypes.Set{ElementType: tftypes.String}, []tftypes.Value{tftypes.NewValue(tftypes.String, "zone-a")}),
			expectReplace: true,
			description:   "Adding zones to empty set should require replacement",
		},
		{
			name:          "non-empty to empty - requires replacement",
			stateValue:    types.SetValueMust(types.StringType, []attr.Value{types.StringValue("zone-a")}),
			planValue:     types.SetValueMust(types.StringType, []attr.Value{}),
			planRaw:       tftypes.NewValue(tftypes.Set{ElementType: tftypes.String}, []tftypes.Value{}),
			expectReplace: true,
			description:   "Removing all zones should require replacement",
		},
		{
			name:          "both empty sets - no replacement",
			stateValue:    types.SetValueMust(types.StringType, []attr.Value{}),
			planValue:     types.SetValueMust(types.StringType, []attr.Value{}),
			planRaw:       tftypes.NewValue(tftypes.Set{ElementType: tftypes.String}, []tftypes.Value{}),
			expectReplace: false,
			description:   "Both empty sets should not require replacement",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			modifier := SetRequiresReplaceOnChange()

			// Test Description method
			desc := modifier.Description(ctx)
			if desc == "" {
				t.Error("Description should not be empty")
			}

			// Test MarkdownDescription method
			mdDesc := modifier.MarkdownDescription(ctx)
			if mdDesc == "" {
				t.Error("MarkdownDescription should not be empty")
			}

			// Create request
			req := planmodifier.SetRequest{
				StateValue: tc.stateValue,
				PlanValue:  tc.planValue,
				Plan: tfsdk.Plan{
					Raw: tc.planRaw,
				},
			}

			// Create response
			resp := &planmodifier.SetResponse{}

			// Execute the plan modifier
			modifier.PlanModifySet(ctx, req, resp)

			// Check for unexpected errors
			if tc.expectError && !resp.Diagnostics.HasError() {
				t.Errorf("Expected error but got none")
			}
			if !tc.expectError && resp.Diagnostics.HasError() {
				t.Errorf("Unexpected error: %v", resp.Diagnostics)
			}

			// Check replacement requirement
			if resp.RequiresReplace != tc.expectReplace {
				t.Errorf("Expected RequiresReplace=%v, got %v. %s", tc.expectReplace, resp.RequiresReplace, tc.description)
			}
		})
	}
}

func TestSetRequiresReplaceOnChangeEdgeCases(t *testing.T) {
	testCases := []struct {
		name          string
		stateValue    types.Set
		planValue     types.Set
		planRaw       tftypes.Value
		expectReplace bool
		description   string
	}{
		{
			name:          "single element sets with same value",
			stateValue:    types.SetValueMust(types.StringType, []attr.Value{types.StringValue("us-east-1a")}),
			planValue:     types.SetValueMust(types.StringType, []attr.Value{types.StringValue("us-east-1a")}),
			planRaw:       tftypes.NewValue(tftypes.Set{ElementType: tftypes.String}, []tftypes.Value{tftypes.NewValue(tftypes.String, "us-east-1a")}),
			expectReplace: false,
			description:   "Single element sets with same value should not require replacement",
		},
		{
			name:          "large sets with same membership",
			stateValue:    types.SetValueMust(types.StringType, []attr.Value{types.StringValue("zone-1"), types.StringValue("zone-2"), types.StringValue("zone-3"), types.StringValue("zone-4"), types.StringValue("zone-5")}),
			planValue:     types.SetValueMust(types.StringType, []attr.Value{types.StringValue("zone-5"), types.StringValue("zone-1"), types.StringValue("zone-3"), types.StringValue("zone-2"), types.StringValue("zone-4")}),
			planRaw:       tftypes.NewValue(tftypes.Set{ElementType: tftypes.String}, []tftypes.Value{tftypes.NewValue(tftypes.String, "zone-5"), tftypes.NewValue(tftypes.String, "zone-1"), tftypes.NewValue(tftypes.String, "zone-3"), tftypes.NewValue(tftypes.String, "zone-2"), tftypes.NewValue(tftypes.String, "zone-4")}),
			expectReplace: false,
			description:   "Large sets with same membership should not require replacement",
		},
		{
			name:          "sets with special characters",
			stateValue:    types.SetValueMust(types.StringType, []attr.Value{types.StringValue("zone-with-dashes"), types.StringValue("zone_with_underscores")}),
			planValue:     types.SetValueMust(types.StringType, []attr.Value{types.StringValue("zone_with_underscores"), types.StringValue("zone-with-dashes")}),
			planRaw:       tftypes.NewValue(tftypes.Set{ElementType: tftypes.String}, []tftypes.Value{tftypes.NewValue(tftypes.String, "zone_with_underscores"), tftypes.NewValue(tftypes.String, "zone-with-dashes")}),
			expectReplace: false,
			description:   "Sets with special characters should work correctly",
		},
		{
			name:          "partial membership change",
			stateValue:    types.SetValueMust(types.StringType, []attr.Value{types.StringValue("zone-a"), types.StringValue("zone-b"), types.StringValue("zone-c")}),
			planValue:     types.SetValueMust(types.StringType, []attr.Value{types.StringValue("zone-a"), types.StringValue("zone-b"), types.StringValue("zone-d")}),
			planRaw:       tftypes.NewValue(tftypes.Set{ElementType: tftypes.String}, []tftypes.Value{tftypes.NewValue(tftypes.String, "zone-a"), tftypes.NewValue(tftypes.String, "zone-b"), tftypes.NewValue(tftypes.String, "zone-d")}),
			expectReplace: true,
			description:   "Partial membership change should require replacement",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			modifier := SetRequiresReplaceOnChange()

			req := planmodifier.SetRequest{
				StateValue: tc.stateValue,
				PlanValue:  tc.planValue,
				Plan: tfsdk.Plan{
					Raw: tc.planRaw,
				},
			}

			resp := &planmodifier.SetResponse{}
			modifier.PlanModifySet(ctx, req, resp)

			if resp.Diagnostics.HasError() {
				t.Errorf("Unexpected error: %v", resp.Diagnostics)
			}

			if resp.RequiresReplace != tc.expectReplace {
				t.Errorf("Expected RequiresReplace=%v, got %v. %s", tc.expectReplace, resp.RequiresReplace, tc.description)
			}
		})
	}
}
