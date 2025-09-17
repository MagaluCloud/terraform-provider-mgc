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

type RequireReplacePlanModifier struct {
	UseStateForUnknown bool
}

func RequireReplace() planmodifier.String {
	return RequireReplacePlanModifier{
		UseStateForUnknown: false,
	}
}

func RequireReplaceWithStateForUnknown() planmodifier.String {
	return RequireReplacePlanModifier{
		UseStateForUnknown: true,
	}
}

func (m RequireReplacePlanModifier) Description(_ context.Context) string {
	if m.UseStateForUnknown {
		return "Requires resource replacement when value changes and uses state value when plan value is unknown."
	}
	return "Requires resource replacement when value changes."
}

func (m RequireReplacePlanModifier) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (m RequireReplacePlanModifier) PlanModifyString(ctx context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	if req.PlanValue.IsNull() && !req.StateValue.IsNull() {
		resp.RequiresReplace = true
		return
	}

	if req.Plan.Raw.IsNull() {
		return
	}

	if req.PlanValue.IsUnknown() && m.UseStateForUnknown {
		if !req.StateValue.IsNull() && !req.StateValue.IsUnknown() {
			resp.PlanValue = req.StateValue
		}
		return
	}

	if req.PlanValue.IsUnknown() {
		return
	}

	if req.StateValue.IsNull() || req.StateValue.IsUnknown() {
		return
	}

	if !req.StateValue.Equal(req.PlanValue) {
		resp.RequiresReplace = true
	}
}

type setRequiresReplaceOnChange struct{}

// SetRequiresReplaceOnChange returns a planmodifier that requires resource replacement
// when the set contents change (elements added, removed, or modified).
//
// This planmodifier is particularly useful for fields like availability_zones where
// changes to the set membership should trigger resource replacement.
//
// Example usage:
//
//	"availability_zones": schema.SetAttribute{
//	  Description: "List of availability zones where the resource is deployed.",
//	  Optional:    true,
//	  Computed:    true,
//	  PlanModifiers: []planmodifier.Set{
//	    utils.SetRequiresReplaceOnChange(),
//	  },
//	  ElementType: types.StringType,
//	},
func SetRequiresReplaceOnChange() planmodifier.Set {
	return setRequiresReplaceOnChange{}
}

func (m setRequiresReplaceOnChange) Description(_ context.Context) string {
	return "Requires resource replacement when the set contents change (elements added, removed, or modified)."
}

func (m setRequiresReplaceOnChange) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (m setRequiresReplaceOnChange) PlanModifySet(ctx context.Context, req planmodifier.SetRequest, resp *planmodifier.SetResponse) {
	// If plan value is unknown, we can't determine if replacement is needed
	if req.PlanValue.IsUnknown() {
		return
	}

	// If state is null/unknown and plan has a value, no replacement needed (resource creation)
	if req.StateValue.IsNull() || req.StateValue.IsUnknown() {
		return
	}

	// If plan is null but state has a value, require replacement
	if req.PlanValue.IsNull() && !req.StateValue.IsNull() {
		resp.RequiresReplace = true
		return
	}

	// If plan is being destroyed, no replacement needed
	if req.Plan.Raw.IsNull() {
		return
	}

	// Convert both sets to string slices for comparison
	var stateElements, planElements []types.String

	stateDiags := req.StateValue.ElementsAs(ctx, &stateElements, false)
	resp.Diagnostics.Append(stateDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	planDiags := req.PlanValue.ElementsAs(ctx, &planElements, false)
	resp.Diagnostics.Append(planDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// If lengths are different, replacement is required
	if len(stateElements) != len(planElements) {
		resp.RequiresReplace = true
		return
	}

	// Create maps for membership comparison (since sets are unordered)
	stateMap := make(map[string]bool)
	for _, elem := range stateElements {
		stateMap[elem.ValueString()] = true
	}

	planMap := make(map[string]bool)
	for _, elem := range planElements {
		planMap[elem.ValueString()] = true
	}

	// Check if all state elements exist in plan and vice versa
	for stateValue := range stateMap {
		if !planMap[stateValue] {
			resp.RequiresReplace = true
			return
		}
	}

	for planValue := range planMap {
		if !stateMap[planValue] {
			resp.RequiresReplace = true
			return
		}
	}

	// If we reach here, the sets have the same membership
	resp.RequiresReplace = false
}
