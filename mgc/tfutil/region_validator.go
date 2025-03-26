package tfutil

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

type RegionValidator struct{}

func (v *RegionValidator) Description(ctx context.Context) string {
	return "region is only validated when env is 'prod'"
}

func (v *RegionValidator) MarkdownDescription(ctx context.Context) string {
	return "region is only validated when env is `prod`"
}

func (v *RegionValidator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	var env string
	diags := req.Config.GetAttribute(ctx, path.Root("env"), &env)
	if diags.HasError() {
		return
	}

	if env != "dev-qa" {
		validRegions := []string{"br-ne1", "br-se1", "br-mgl1"}
		region := req.ConfigValue.ValueString()

		for _, validRegion := range validRegions {
			if region == validRegion {
				return
			}
		}

		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid Region",
			fmt.Sprintf("Expected region to be one of %v, got: %s", validRegions, region),
		)
	}
}
