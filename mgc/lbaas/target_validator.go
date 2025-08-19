package lbaas

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// TargetValidator validates target configuration based on targets_type
type TargetValidator struct{}

func (v TargetValidator) Description(ctx context.Context) string {
	return "Validates target configuration based on targets_type"
}

func (v TargetValidator) MarkdownDescription(ctx context.Context) string {
	return "Validates target configuration based on targets_type"
}

func (v TargetValidator) ValidateList(ctx context.Context, req validator.ListRequest, resp *validator.ListResponse) {
	// Get targets_type from the parent backend object
	var targetsType types.String
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, req.Path.ParentPath().AtName("targets_type"), &targetsType)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Skip validation if targets_type is not set
	if targetsType.IsNull() || targetsType.IsUnknown() || targetsType.ValueString() == "" {
		return
	}

	// Get the targets list
	var targets []TargetModel
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, req.Path, &targets)...)
	if resp.Diagnostics.HasError() {
		return
	}

	targetsTypeValue := targetsType.ValueString()

	// Validate each target
	for i, target := range targets {
		targetPath := req.Path.AtListIndex(i)

		switch targetsTypeValue {
		case "raw":
			// For raw type: ip_address and port are required, nic_id must be empty
			if target.IPAddress.IsNull() || target.IPAddress.IsUnknown() || target.IPAddress.ValueString() == "" {
				resp.Diagnostics.AddAttributeError(
					targetPath.AtName("ip_address"),
					"Missing Required Attribute",
					"ip_address is required when targets_type is 'raw'",
				)
			}
			if !target.NICID.IsNull() && !target.NICID.IsUnknown() && target.NICID.ValueString() != "" {
				resp.Diagnostics.AddAttributeError(
					targetPath.AtName("nic_id"),
					"Invalid Attribute",
					"nic_id must be empty when targets_type is 'raw'",
				)
			}
		case "instance":
			// For instance type: nic_id and port are required, ip_address must be empty
			if target.NICID.IsNull() || target.NICID.IsUnknown() || target.NICID.ValueString() == "" {
				resp.Diagnostics.AddAttributeError(
					targetPath.AtName("nic_id"),
					"Missing Required Attribute",
					"nic_id is required when targets_type is 'instance'",
				)
			}
			if !target.IPAddress.IsNull() && !target.IPAddress.IsUnknown() && target.IPAddress.ValueString() != "" {
				resp.Diagnostics.AddAttributeError(
					targetPath.AtName("ip_address"),
					"Invalid Attribute",
					"ip_address must be empty when targets_type is 'instance'",
				)
			}
		}
	}
}
