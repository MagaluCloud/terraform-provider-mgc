package project

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
)

type keepWhenUnset struct{}

func (keepWhenUnset) Description(context.Context) string {
	return "Keeps the current project when project_id is removed from the configuration and requires replacement when it changes."
}

func (m keepWhenUnset) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (keepWhenUnset) PlanModifyString(_ context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	if req.Plan.Raw.IsNull() {
		return
	}
	if req.ConfigValue.IsNull() {
		resp.PlanValue = req.StateValue
		return
	}
	if req.State.Raw.IsNull() {
		return
	}
	if !req.PlanValue.Equal(req.StateValue) {
		resp.RequiresReplace = true
	}
}
