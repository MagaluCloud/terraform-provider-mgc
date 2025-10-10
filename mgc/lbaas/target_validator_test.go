package lbaas

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
)

func TestValidateTLSCertNameForProtocol_TCP_WithTLSName(t *testing.T) {
	summary, detail, has := validateTLSCertNameForProtocol(types.StringValue("tcp"), types.StringValue("web-ssl-cert"))
	assert.True(t, has)
	assert.Equal(t, "Value error", summary)
	assert.Equal(t, "tls_certificate_name should not be provided when protocol is 'tcp'", detail)
}

func TestValidateTLSCertNameForProtocol_TCP_WithoutTLSName(t *testing.T) {
	_, _, has := validateTLSCertNameForProtocol(types.StringValue("tcp"), types.StringNull())
	assert.False(t, has)
}

func TestValidateTLSCertNameForProtocol_TLS_WithoutTLSName(t *testing.T) {
	summary, detail, has := validateTLSCertNameForProtocol(types.StringValue("tls"), types.StringNull())
	assert.True(t, has)
	assert.Equal(t, "Missing Required Attribute", summary)
	assert.Equal(t, "tls_certificate_name must be provided when protocol is 'tls'", detail)
}

func TestValidateTLSCertNameForProtocol_TLS_WithTLSName(t *testing.T) {
	_, _, has := validateTLSCertNameForProtocol(types.StringValue("tls"), types.StringValue("web-ssl-cert"))
	assert.False(t, has)
}

func TestValidateTLSCertNameForProtocol_UnknownProtocol(t *testing.T) {
	_, _, has := validateTLSCertNameForProtocol(types.StringNull(), types.StringValue("web-ssl-cert"))
	assert.False(t, has)

	_, _, has = validateTLSCertNameForProtocol(types.StringUnknown(), types.StringValue("web-ssl-cert"))
	assert.False(t, has)

	_, _, has = validateTLSCertNameForProtocol(types.StringValue(""), types.StringValue("web-ssl-cert"))
	assert.False(t, has)
}

func makeConfig(t *testing.T, targetsType string) tfsdk.Config {
	return tfsdk.Config{}
}

func runTargetValidator(t *testing.T, targetsType string, set types.Set) diag.Diagnostics {
	cfg := makeConfig(t, targetsType)
	req := validator.SetRequest{
		Path:        path.Root("targets"),
		Config:      cfg,
		ConfigValue: set,
	}
	resp := &validator.SetResponse{
		Diagnostics: diag.Diagnostics{},
	}
	TargetValidator{}.ValidateSet(context.Background(), req, resp)
	return resp.Diagnostics
}

func TestTargetValidator_UnknownSet_NoDiagnostics(t *testing.T) {
	elementType := types.ObjectType{AttrTypes: map[string]attr.Type{
		"nic_id":     types.StringType,
		"ip_address": types.StringType,
		"port":       types.Int64Type,
	}}
	unknownSet := types.SetUnknown(elementType)
	diags := runTargetValidator(t, "raw", unknownSet)
	assert.False(t, diags.HasError())
}

func TestTargetValidator_RawType_MissingIPAddress(t *testing.T) {
	elementType := types.ObjectType{AttrTypes: map[string]attr.Type{
		"nic_id":     types.StringType,
		"ip_address": types.StringType,
		"port":       types.Int64Type,
	}}
	elem, _ := types.ObjectValue(map[string]attr.Type{
		"nic_id":     types.StringType,
		"ip_address": types.StringType,
		"port":       types.Int64Type,
	}, map[string]attr.Value{
		"nic_id":     types.StringNull(),
		"ip_address": types.StringNull(),
		"port":       types.Int64Value(80),
	})
	set := types.SetValueMust(elementType, []attr.Value{elem})
	diags := runTargetValidator(t, "raw", set)
	assert.False(t, diags.HasError())
}

func TestTargetValidator_RawType_WithNICIDInvalid(t *testing.T) {
	elementType := types.ObjectType{AttrTypes: map[string]attr.Type{
		"nic_id":     types.StringType,
		"ip_address": types.StringType,
		"port":       types.Int64Type,
	}}
	elem, _ := types.ObjectValue(map[string]attr.Type{
		"nic_id":     types.StringType,
		"ip_address": types.StringType,
		"port":       types.Int64Type,
	}, map[string]attr.Value{
		"nic_id":     types.StringValue("nic-1"),
		"ip_address": types.StringValue("10.0.0.1"),
		"port":       types.Int64Value(80),
	})
	set := types.SetValueMust(elementType, []attr.Value{elem})
	diags := runTargetValidator(t, "raw", set)
	assert.False(t, diags.HasError())
}

func TestTargetValidator_InstanceType_MissingNICID(t *testing.T) {
	elementType := types.ObjectType{AttrTypes: map[string]attr.Type{
		"nic_id":     types.StringType,
		"ip_address": types.StringType,
		"port":       types.Int64Type,
	}}
	elem, _ := types.ObjectValue(map[string]attr.Type{
		"nic_id":     types.StringType,
		"ip_address": types.StringType,
		"port":       types.Int64Type,
	}, map[string]attr.Value{
		"nic_id":     types.StringNull(),
		"ip_address": types.StringNull(),
		"port":       types.Int64Value(443),
	})
	set := types.SetValueMust(elementType, []attr.Value{elem})
	diags := runTargetValidator(t, "instance", set)
	assert.False(t, diags.HasError())
}

func TestTargetValidator_InstanceType_WithIPAddressInvalid(t *testing.T) {
	elementType := types.ObjectType{AttrTypes: map[string]attr.Type{
		"nic_id":     types.StringType,
		"ip_address": types.StringType,
		"port":       types.Int64Type,
	}}
	elem, _ := types.ObjectValue(map[string]attr.Type{
		"nic_id":     types.StringType,
		"ip_address": types.StringType,
		"port":       types.Int64Type,
	}, map[string]attr.Value{
		"nic_id":     types.StringValue("nic-1"),
		"ip_address": types.StringValue("10.0.0.2"),
		"port":       types.Int64Value(8080),
	})
	set := types.SetValueMust(elementType, []attr.Value{elem})
	diags := runTargetValidator(t, "instance", set)
	assert.False(t, diags.HasError())
}
