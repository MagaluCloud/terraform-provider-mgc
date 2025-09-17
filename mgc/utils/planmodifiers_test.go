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
