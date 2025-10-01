package lbaas

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

type TLSCertificatesValidator struct{}

func (v TLSCertificatesValidator) Description(ctx context.Context) string {
	return "Validates that tls_certificates list is not empty when provided"
}

func (v TLSCertificatesValidator) MarkdownDescription(ctx context.Context) string {
	return "Validates that tls_certificates list is not empty when provided"
}

func (v TLSCertificatesValidator) ValidateList(ctx context.Context, req validator.ListRequest, resp *validator.ListResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	if len(req.ConfigValue.Elements()) == 0 {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Empty TLS Certificates List",
			"tls_certificates cannot be an empty list. Either provide at least one certificate or omit the attribute entirely.",
		)
	}
}
