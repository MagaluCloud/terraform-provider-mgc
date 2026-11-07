package project

import (
	"context"
	"testing"

	rschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/stretchr/testify/assert"
)

func TestKeepWhenUnset(t *testing.T) {
	ctx := context.Background()
	const otherProjectID = "11111111-2222-4333-8444-555555555555"

	schema := rschema.Schema{Attributes: map[string]rschema.Attribute{"project_id": ResourceAttribute()}}
	objectType := schema.Type().TerraformType(ctx)
	object := func(v any) tftypes.Value {
		return tftypes.NewValue(objectType, map[string]tftypes.Value{"project_id": tftypes.NewValue(tftypes.String, v)})
	}
	absent := tftypes.NewValue(objectType, nil)

	testCases := []struct {
		name            string
		state           tftypes.Value
		plan            tftypes.Value
		stateValue      types.String
		configValue     types.String
		planValue       types.String
		expectedPlan    types.String
		expectedReplace bool
	}{
		{
			name:         "Create without project plans null instead of unknown",
			state:        absent,
			plan:         object(tftypes.UnknownValue),
			stateValue:   types.StringNull(),
			configValue:  types.StringNull(),
			planValue:    types.StringUnknown(),
			expectedPlan: types.StringNull(),
		},
		{
			name:         "Create with project",
			state:        absent,
			plan:         object(testProjectID),
			stateValue:   types.StringNull(),
			configValue:  types.StringValue(testProjectID),
			planValue:    types.StringValue(testProjectID),
			expectedPlan: types.StringValue(testProjectID),
		},
		{
			name:         "Removing the project keeps the one in state",
			state:        object(testProjectID),
			plan:         object(tftypes.UnknownValue),
			stateValue:   types.StringValue(testProjectID),
			configValue:  types.StringNull(),
			planValue:    types.StringUnknown(),
			expectedPlan: types.StringValue(testProjectID),
		},
		{
			name:         "Default project stays default",
			state:        object(nil),
			plan:         object(tftypes.UnknownValue),
			stateValue:   types.StringNull(),
			configValue:  types.StringNull(),
			planValue:    types.StringUnknown(),
			expectedPlan: types.StringNull(),
		},
		{
			name:            "Changing the project forces replacement",
			state:           object(testProjectID),
			plan:            object(otherProjectID),
			stateValue:      types.StringValue(testProjectID),
			configValue:     types.StringValue(otherProjectID),
			planValue:       types.StringValue(otherProjectID),
			expectedPlan:    types.StringValue(otherProjectID),
			expectedReplace: true,
		},
		{
			name:            "Moving out of the default project forces replacement",
			state:           object(nil),
			plan:            object(testProjectID),
			stateValue:      types.StringNull(),
			configValue:     types.StringValue(testProjectID),
			planValue:       types.StringValue(testProjectID),
			expectedPlan:    types.StringValue(testProjectID),
			expectedReplace: true,
		},
		{
			name:         "Same project is not a change",
			state:        object(testProjectID),
			plan:         object(testProjectID),
			stateValue:   types.StringValue(testProjectID),
			configValue:  types.StringValue(testProjectID),
			planValue:    types.StringValue(testProjectID),
			expectedPlan: types.StringValue(testProjectID),
		},
		{
			name:            "Unknown project on an existing resource may change, so it replaces",
			state:           object(testProjectID),
			plan:            object(tftypes.UnknownValue),
			stateValue:      types.StringValue(testProjectID),
			configValue:     types.StringUnknown(),
			planValue:       types.StringUnknown(),
			expectedPlan:    types.StringUnknown(),
			expectedReplace: true,
		},
		{
			name:         "Destroy is left alone",
			state:        object(testProjectID),
			plan:         absent,
			stateValue:   types.StringValue(testProjectID),
			configValue:  types.StringNull(),
			planValue:    types.StringNull(),
			expectedPlan: types.StringNull(),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := planmodifier.StringRequest{
				State:       tfsdk.State{Schema: schema, Raw: tc.state},
				Plan:        tfsdk.Plan{Schema: schema, Raw: tc.plan},
				StateValue:  tc.stateValue,
				ConfigValue: tc.configValue,
				PlanValue:   tc.planValue,
			}
			resp := &planmodifier.StringResponse{PlanValue: tc.planValue}

			keepWhenUnset{}.PlanModifyString(ctx, req, resp)

			assert.Equal(t, tc.expectedPlan, resp.PlanValue)
			assert.Equal(t, tc.expectedReplace, resp.RequiresReplace)
		})
	}
}
