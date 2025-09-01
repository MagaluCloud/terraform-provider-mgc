package lbaas

import (
	"testing"

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
