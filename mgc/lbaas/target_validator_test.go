package lbaas

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	fwschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
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

func TestValidateTLSCertNameForProtocol_TCP_WithEmptyTLSName(t *testing.T) {
	_, _, has := validateTLSCertNameForProtocol(types.StringValue("tcp"), types.StringValue(""))
	assert.False(t, has)
}

func TestValidateTLSCertNameForProtocol_TLS_WithoutTLSName(t *testing.T) {
	summary, detail, has := validateTLSCertNameForProtocol(types.StringValue("tls"), types.StringNull())
	assert.True(t, has)
	assert.Equal(t, "Missing Required Attribute", summary)
	assert.Equal(t, "tls_certificate_name must be provided when protocol is 'tls'", detail)
}

func TestValidateTLSCertNameForProtocol_TLS_WithEmptyTLSName(t *testing.T) {
	summary, detail, has := validateTLSCertNameForProtocol(types.StringValue("tls"), types.StringValue(""))
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

func TestValidateTLSCertNameForProtocol_UnknownValue(t *testing.T) {
	_, _, has := validateTLSCertNameForProtocol(types.StringValue("tcp"), types.StringUnknown())
	assert.False(t, has)

	_, _, has = validateTLSCertNameForProtocol(types.StringValue("tls"), types.StringUnknown())
	assert.False(t, has)
}

func runTargetValidator(t *testing.T, targetsType string, set types.Set) diag.Diagnostics {
	testSchema := fwschema.Schema{
		Attributes: map[string]fwschema.Attribute{
			"targets_type": fwschema.StringAttribute{
				Optional: true,
			},
			"targets": fwschema.SetNestedAttribute{
				NestedObject: fwschema.NestedAttributeObject{
					Attributes: map[string]fwschema.Attribute{
						"nic_id": fwschema.StringAttribute{
							Optional: true,
						},
						"ip_address": fwschema.StringAttribute{
							Optional: true,
						},
						"port": fwschema.Int64Attribute{
							Optional: true,
						},
					},
				},
			},
		},
	}

	targetObjectType := tftypes.Object{
		AttributeTypes: map[string]tftypes.Type{
			"nic_id":     tftypes.String,
			"ip_address": tftypes.String,
			"port":       tftypes.Number,
		},
	}

	rawValue := tftypes.NewValue(
		tftypes.Object{
			AttributeTypes: map[string]tftypes.Type{
				"targets_type": tftypes.String,
				"targets": tftypes.Set{
					ElementType: targetObjectType,
				},
			},
		},
		map[string]tftypes.Value{
			"targets_type": tftypes.NewValue(tftypes.String, targetsType),
			"targets":      tftypes.NewValue(tftypes.Set{ElementType: targetObjectType}, nil),
		},
	)

	cfg := tfsdk.Config{
		Schema: testSchema,
		Raw:    rawValue,
	}

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

func TestTargetValidator_RawType_UnknownIPAddress_NoDiagnostics(t *testing.T) {
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
		"ip_address": types.StringUnknown(),
		"port":       types.Int64Value(80),
	})
	set := types.SetValueMust(elementType, []attr.Value{elem})
	diags := runTargetValidator(t, "raw", set)
	assert.False(t, diags.HasError())
}

func TestTargetValidator_RawType_UnknownNICID_NoDiagnostics(t *testing.T) {
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
		"nic_id":     types.StringUnknown(),
		"ip_address": types.StringValue("10.0.0.1"),
		"port":       types.Int64Value(80),
	})
	set := types.SetValueMust(elementType, []attr.Value{elem})
	diags := runTargetValidator(t, "raw", set)
	assert.False(t, diags.HasError())
}

func TestTargetValidator_InstanceType_UnknownNICID_NoDiagnostics(t *testing.T) {
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
		"nic_id":     types.StringUnknown(),
		"ip_address": types.StringNull(),
		"port":       types.Int64Value(443),
	})
	set := types.SetValueMust(elementType, []attr.Value{elem})
	diags := runTargetValidator(t, "instance", set)
	assert.False(t, diags.HasError())
}

func TestTargetValidator_InstanceType_UnknownIPAddress_NoDiagnostics(t *testing.T) {
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
		"ip_address": types.StringUnknown(),
		"port":       types.Int64Value(8080),
	})
	set := types.SetValueMust(elementType, []attr.Value{elem})
	diags := runTargetValidator(t, "instance", set)
	assert.False(t, diags.HasError())
}

func TestTargetValidator_RawType_MissingIPAddress_HasError(t *testing.T) {
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
	assert.True(t, diags.HasError())
	assert.Contains(t, diags[0].Summary(), "Missing Required Attribute")
	assert.Contains(t, diags[0].Detail(), "ip_address is required when targets_type is 'raw'")
}

func TestTargetValidator_RawType_EmptyIPAddress_HasError(t *testing.T) {
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
		"ip_address": types.StringValue(""),
		"port":       types.Int64Value(80),
	})
	set := types.SetValueMust(elementType, []attr.Value{elem})
	diags := runTargetValidator(t, "raw", set)
	assert.True(t, diags.HasError())
	assert.Contains(t, diags[0].Summary(), "Missing Required Attribute")
}

func TestTargetValidator_RawType_WithNICID_HasError(t *testing.T) {
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
	assert.True(t, diags.HasError())
	assert.Contains(t, diags[0].Summary(), "Invalid Attribute")
	assert.Contains(t, diags[0].Detail(), "nic_id must be empty when targets_type is 'raw'")
}

func TestTargetValidator_RawType_Valid_NoError(t *testing.T) {
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
		"ip_address": types.StringValue("10.0.0.1"),
		"port":       types.Int64Value(80),
	})
	set := types.SetValueMust(elementType, []attr.Value{elem})
	diags := runTargetValidator(t, "raw", set)
	assert.False(t, diags.HasError())
}

func TestTargetValidator_InstanceType_MissingNICID_HasError(t *testing.T) {
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
	assert.True(t, diags.HasError())
	assert.Contains(t, diags[0].Summary(), "Missing Required Attribute")
	assert.Contains(t, diags[0].Detail(), "nic_id is required when targets_type is 'instance'")
}

func TestTargetValidator_InstanceType_EmptyNICID_HasError(t *testing.T) {
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
		"nic_id":     types.StringValue(""),
		"ip_address": types.StringNull(),
		"port":       types.Int64Value(443),
	})
	set := types.SetValueMust(elementType, []attr.Value{elem})
	diags := runTargetValidator(t, "instance", set)
	assert.True(t, diags.HasError())
	assert.Contains(t, diags[0].Summary(), "Missing Required Attribute")
}

func TestTargetValidator_InstanceType_WithIPAddress_HasError(t *testing.T) {
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
	assert.True(t, diags.HasError())
	assert.Contains(t, diags[0].Summary(), "Invalid Attribute")
	assert.Contains(t, diags[0].Detail(), "ip_address must be empty when targets_type is 'instance'")
}

func TestTargetValidator_InstanceType_Valid_NoError(t *testing.T) {
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
		"ip_address": types.StringNull(),
		"port":       types.Int64Value(8080),
	})
	set := types.SetValueMust(elementType, []attr.Value{elem})
	diags := runTargetValidator(t, "instance", set)
	assert.False(t, diags.HasError())
}
