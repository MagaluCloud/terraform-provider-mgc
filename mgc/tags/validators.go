package tags

import (
	"context"
	"fmt"
	"slices"
	"strings"

	tagSDK "github.com/MagaluCloud/mgc-sdk-go/tag"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

// knownKinds are the kinds the API accepts today. The list grows on the API
// side, so it drives a warning, never an error.
var knownKinds = []string{string(tagSDK.TagKindFinops)}

var _ validator.String = kindValidator{}

type kindValidator struct{}

func (v kindValidator) Description(context.Context) string {
	return fmt.Sprintf("warns when the kind is not one of: %s", strings.Join(knownKinds, ", "))
}

func (v kindValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v kindValidator) ValidateString(_ context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	kind := req.ConfigValue.ValueString()
	if slices.Contains(knownKinds, kind) {
		return
	}

	resp.Diagnostics.AddAttributeWarning(
		req.Path,
		"Unknown tag kind",
		fmt.Sprintf(
			"%q is not a kind this provider knows about (known kinds: %s). "+
				"It is sent to the API as is: newer kinds work without a provider upgrade, but a typo is rejected only at apply time.",
			kind, strings.Join(knownKinds, ", "),
		),
	)
}
