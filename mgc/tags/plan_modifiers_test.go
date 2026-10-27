package tags

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTagValueIDFromNames(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	_, valueSchema := newTestTagValueResource(t, nil)

	tests := []struct {
		name     string
		tagName  types.String
		expected types.String
	}{
		{
			name:     "renamed tag",
			tagName:  types.StringValue("env"),
			expected: types.StringValue("env,producao"),
		},
		{
			name:     "tag name known only after apply",
			tagName:  types.StringUnknown(),
			expected: types.StringUnknown(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			plan := tfsdk.Plan{Schema: valueSchema}
			require.False(t, plan.Set(ctx, &tagValueResourceModel{
				ID:          types.StringUnknown(),
				TagName:     tt.tagName,
				Name:        types.StringValue("producao"),
				Description: types.StringNull(),
				CreatedAt:   types.StringUnknown(),
				UpdatedAt:   types.StringUnknown(),
			}).HasError())

			resp := &planmodifier.StringResponse{PlanValue: types.StringUnknown()}
			tagValueIDFromNames().PlanModifyString(ctx, planmodifier.StringRequest{
				Plan:       plan,
				StateValue: types.StringValue("ambiente,producao"),
				PlanValue:  types.StringUnknown(),
			}, resp)

			require.False(t, resp.Diagnostics.HasError(), "%v", resp.Diagnostics)
			assert.Equal(t, tt.expected, resp.PlanValue)
		})
	}
}

func TestTagIDFromNameOnDestroy(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	_, tagSchema := newTestTagResource(t, nil)

	resp := &planmodifier.StringResponse{PlanValue: types.StringNull()}
	tagIDFromName().PlanModifyString(ctx, planmodifier.StringRequest{
		Plan:      tfsdk.Plan{Schema: tagSchema, Raw: tftypes.NewValue(tagSchema.Type().TerraformType(ctx), nil)},
		PlanValue: types.StringNull(),
	}, resp)

	require.False(t, resp.Diagnostics.HasError(), "%v", resp.Diagnostics)

	assert.True(t, resp.PlanValue.IsNull())
}
