package tags

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
)

// A kind the provider does not know about must not block the apply: the API is
// the authority and new kinds show up without a provider release.
func TestKindValidator(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		value       types.String
		wantWarning bool
	}{
		{
			name:        "known kind",
			value:       types.StringValue("finops"),
			wantWarning: false,
		},
		{
			name:        "unknown kind warns",
			value:       types.StringValue("secops"),
			wantWarning: true,
		},
		{
			// The API is case sensitive, so the wrong casing is a different kind.
			name:        "wrong casing warns",
			value:       types.StringValue("FinOps"),
			wantWarning: true,
		},
		{
			name:        "null is left to the schema",
			value:       types.StringNull(),
			wantWarning: false,
		},
		{
			name:        "unknown value is left to the schema",
			value:       types.StringUnknown(),
			wantWarning: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			req := validator.StringRequest{
				Path:        path.Root("kinds"),
				ConfigValue: tt.value,
			}
			resp := &validator.StringResponse{}

			kindValidator{}.ValidateString(context.Background(), req, resp)

			assert.False(t, resp.Diagnostics.HasError(), "an unknown kind must never fail the plan")
			assert.Equal(t, tt.wantWarning, resp.Diagnostics.WarningsCount() > 0)
		})
	}
}

func TestKindValidatorDescription(t *testing.T) {
	t.Parallel()

	assert.NotEmpty(t, kindValidator{}.Description(context.Background()))
	assert.NotEmpty(t, kindValidator{}.MarkdownDescription(context.Background()))
}
