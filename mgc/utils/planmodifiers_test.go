package utils

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
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
