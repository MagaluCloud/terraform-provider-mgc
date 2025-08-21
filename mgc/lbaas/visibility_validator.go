package lbaas

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// VisibilityValidator validates that public_ip_id is correctly configured based on visibility
type VisibilityValidator struct{}

func (v VisibilityValidator) Description(ctx context.Context) string {
	return "Validates that public_ip_id is required for external load balancers and forbidden for internal load balancers"
}

func (v VisibilityValidator) MarkdownDescription(ctx context.Context) string {
	return "Validates that public_ip_id is required for external load balancers and forbidden for internal load balancers"
}

func (v VisibilityValidator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	// Skip validation if visibility is not set
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	visibility := req.ConfigValue.ValueString()

	// Get the public_ip_id from the parent load balancer object
	var publicIPID types.String
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, req.Path.ParentPath().AtName("public_ip_id"), &publicIPID)...)
	if resp.Diagnostics.HasError() {
		return
	}

	hasPublicIP := !publicIPID.IsNull() && !publicIPID.IsUnknown() && publicIPID.ValueString() != ""

	switch visibility {
	case "external":
		if !hasPublicIP {
			resp.Diagnostics.AddAttributeError(
				req.Path.ParentPath().AtName("public_ip_id"),
				"Public IP Required",
				"public_ip_id is required when visibility is set to 'external'",
			)
		}
	case "internal":
		if hasPublicIP {
			resp.Diagnostics.AddAttributeError(
				req.Path.ParentPath().AtName("public_ip_id"),
				"Public IP Not Allowed",
				"public_ip_id must not be provided when visibility is set to 'internal'",
			)
		}
	}
}
