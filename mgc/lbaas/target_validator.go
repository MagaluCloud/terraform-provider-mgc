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

func (v TargetValidator) ValidateSet(ctx context.Context, req validator.SetRequest, resp *validator.SetResponse) {
	var targetsType types.String
	if req.Config.Schema == nil {
		return
	}
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, req.Path.ParentPath().AtName("targets_type"), &targetsType)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if targetsType.IsNull() || targetsType.IsUnknown() || targetsType.ValueString() == "" {
		return
	}

	var targets []TargetModel
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	resp.Diagnostics.Append(req.ConfigValue.ElementsAs(ctx, &targets, true)...)
	if resp.Diagnostics.HasError() {
		return
	}

	targetsTypeValue := targetsType.ValueString()

	for i, target := range targets {
		targetPath := req.Path.AtListIndex(i)

		switch targetsTypeValue {
		case "raw":
			// For raw type: ip_address and port are required, nic_id must be empty
			if !target.IPAddress.IsUnknown() && (target.IPAddress.IsNull() || target.IPAddress.ValueString() == "") {
				resp.Diagnostics.AddAttributeError(
					targetPath.AtName("ip_address"),
					"Missing Required Attribute",
					"ip_address is required when targets_type is 'raw'",
				)
			}
			if !target.NICID.IsUnknown() && !target.NICID.IsNull() && target.NICID.ValueString() != "" {
				resp.Diagnostics.AddAttributeError(
					targetPath.AtName("nic_id"),
					"Invalid Attribute",
					"nic_id must be empty when targets_type is 'raw'",
				)
			}
		case "instance":
			// For instance type: nic_id and port are required, ip_address must be empty
			if !target.NICID.IsUnknown() && (target.NICID.IsNull() || target.NICID.ValueString() == "") {
				resp.Diagnostics.AddAttributeError(
					targetPath.AtName("nic_id"),
					"Missing Required Attribute",
					"nic_id is required when targets_type is 'instance'",
				)
			}
			if !target.IPAddress.IsUnknown() && !target.IPAddress.IsNull() && target.IPAddress.ValueString() != "" {
				resp.Diagnostics.AddAttributeError(
					targetPath.AtName("ip_address"),
					"Invalid Attribute",
					"ip_address must be empty when targets_type is 'instance'",
				)
			}
		}
	}
}

// ListenerTLSValidator validates tls_certificate_name according to listener protocol
type ListenerTLSValidator struct{}

func (v ListenerTLSValidator) Description(ctx context.Context) string {
	return "Validates that tls_certificate_name usage matches the listener protocol"
}

func (v ListenerTLSValidator) MarkdownDescription(ctx context.Context) string {
	return "Validates that tls_certificate_name usage matches the listener protocol"
}

func validateTLSCertNameForProtocol(protocol types.String, val types.String) (string, string, bool) {
	if protocol.IsNull() || protocol.IsUnknown() || protocol.ValueString() == "" {
		return "", "", false
	}
	if val.IsUnknown() {
		return "", "", false
	}
	switch protocol.ValueString() {
	case "tcp":
		if !val.IsNull() && val.ValueString() != "" {
			return "Value error", "tls_certificate_name should not be provided when protocol is 'tcp'", true
		}
	case "tls":
		if val.IsNull() || val.ValueString() == "" {
			return "Missing Required Attribute", "tls_certificate_name must be provided when protocol is 'tls'", true
		}
	}
	return "", "", false
}
func (v ListenerTLSValidator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	var protocol types.String
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, req.Path.ParentPath().AtName("protocol"), &protocol)...)
	if resp.Diagnostics.HasError() {
		return
	}

	sum, det, has := validateTLSCertNameForProtocol(protocol, req.ConfigValue)
	if has {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			sum,
			det,
		)
	}
}
