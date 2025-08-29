package utils

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type ReplaceIfChangeAndNotIsNotSetOnPlan struct{}

func (p ReplaceIfChangeAndNotIsNotSetOnPlan) PlanModifyString(ctx context.Context, request planmodifier.StringRequest, response *planmodifier.StringResponse) {
	if request.PlanValue.IsUnknown() {
		response.RequiresReplace = false
		return
	}
	if !request.StateValue.Equal(request.PlanValue) {
		response.RequiresReplace = true
		return
	}
	response.RequiresReplace = false
}

func (p ReplaceIfChangeAndNotIsNotSetOnPlan) Description(context.Context) string {
	return "Requires replace if the value is different from the state and the plan value is not unknown."
}

func (p ReplaceIfChangeAndNotIsNotSetOnPlan) MarkdownDescription(context.Context) string {
	return "Requires replace if the value is different from the state and the plan value is not unknown."
}

type ReplaceIfChangeAndNotIsNotSetOnPlanBool struct{}

func (p ReplaceIfChangeAndNotIsNotSetOnPlanBool) PlanModifyBool(ctx context.Context, request planmodifier.BoolRequest, response *planmodifier.BoolResponse) {
	if request.PlanValue.IsUnknown() {
		response.RequiresReplace = false
		return
	}
	if !request.StateValue.Equal(request.PlanValue) {
		response.RequiresReplace = true
		return
	}
	response.RequiresReplace = false
}

func (p ReplaceIfChangeAndNotIsNotSetOnPlanBool) Description(context.Context) string {
	return "Requires replace if the value is different from the state and the plan value is not unknown."
}

func (p ReplaceIfChangeAndNotIsNotSetOnPlanBool) MarkdownDescription(context.Context) string {
	return "Requires replace if the value is different from the state and the plan value is not unknown."
}

type setMembershipChangePlanModifier struct{}

func SetMembershipChangeRequiresReplace() planmodifier.Set {
	return setMembershipChangePlanModifier{}
}

func (m setMembershipChangePlanModifier) Description(_ context.Context) string {
	return "Requires replacement if the set membership changes"
}

func (m setMembershipChangePlanModifier) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (m setMembershipChangePlanModifier) PlanModifySet(ctx context.Context, req planmodifier.SetRequest, resp *planmodifier.SetResponse) {
	if req.PlanValue.IsUnknown() || req.StateValue.IsUnknown() || req.PlanValue.IsNull() || req.StateValue.IsNull() {
		return
	}

	planCount, diags := countSetElementsGeneric(ctx, req.PlanValue)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	stateCount, diags := countSetElementsGeneric(ctx, req.StateValue)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if planCount != stateCount {
		resp.RequiresReplace = true
	}
}

func countSetElementsGeneric(ctx context.Context, set types.Set) (int, diag.Diagnostics) {
	elems := set.Elements()
	return len(elems), diag.Diagnostics{}
}
