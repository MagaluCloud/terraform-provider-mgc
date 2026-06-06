package mgc

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
)

func TestEndpointURLValidator(t *testing.T) {
	testCases := []struct {
		name          string
		url           string
		expectedValid bool
	}{
		{name: "Valid https URL", url: "https://localhost:8080", expectedValid: true},
		{name: "Valid http URL", url: "http://localhost:8080", expectedValid: true},
		{name: "Valid https URL with host only", url: "https://api.example.com", expectedValid: true},
		{name: "Valid URL with trailing slash", url: "https://localhost:8080/", expectedValid: true},
		{name: "Valid URL with path", url: "https://api.example.com/custom", expectedValid: true},
		{name: "Valid URL surrounded by whitespace", url: "  https://localhost:8080  ", expectedValid: true},
		{name: "Empty string is allowed", url: "", expectedValid: true},
		{name: "Whitespace only is allowed", url: "   ", expectedValid: true},
		{name: "Missing scheme", url: "localhost:8080", expectedValid: false},
		{name: "Missing host", url: "https://", expectedValid: false},
		{name: "Unsupported scheme ftp", url: "ftp://localhost:8080", expectedValid: false},
		{name: "Unsupported scheme tcp", url: "tcp://localhost", expectedValid: false},
		{name: "Plain text", url: "not a url", expectedValid: false},
		{name: "Scheme without host", url: "http:///path", expectedValid: false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := validator.StringRequest{
				Path:        path.Root("test"),
				ConfigValue: types.StringValue(tc.url),
			}
			resp := &validator.StringResponse{Diagnostics: diag.Diagnostics{}}

			endpointURLValidator{}.ValidateString(context.Background(), req, resp)

			if tc.expectedValid {
				assert.Empty(t, resp.Diagnostics, "expected no diagnostics for %q", tc.url)
			} else {
				assert.NotEmpty(t, resp.Diagnostics, "expected diagnostics for %q", tc.url)
			}
		})
	}
}

func TestEndpointURLValidatorNullValue(t *testing.T) {
	req := validator.StringRequest{
		Path:        path.Root("test"),
		ConfigValue: types.StringNull(),
	}
	resp := &validator.StringResponse{Diagnostics: diag.Diagnostics{}}

	endpointURLValidator{}.ValidateString(context.Background(), req, resp)
	assert.Empty(t, resp.Diagnostics, "expected no diagnostics for null value")
}

func TestEndpointURLValidatorUnknownValue(t *testing.T) {
	req := validator.StringRequest{
		Path:        path.Root("test"),
		ConfigValue: types.StringUnknown(),
	}
	resp := &validator.StringResponse{Diagnostics: diag.Diagnostics{}}

	endpointURLValidator{}.ValidateString(context.Background(), req, resp)
	assert.Empty(t, resp.Diagnostics, "expected no diagnostics for unknown value")
}

func TestEndpointURLValidatorDescription(t *testing.T) {
	expected := "value must be an absolute URL with an http or https scheme and a host"
	assert.Equal(t, expected, endpointURLValidator{}.Description(context.Background()))
	assert.Equal(t, expected, endpointURLValidator{}.MarkdownDescription(context.Background()))
}

func TestSetEndpoint(t *testing.T) {
	const service = "block_storage"

	testCases := []struct {
		name      string
		val       types.String
		wantSet   bool
		wantValue string
	}{
		{name: "Plain URL is stored as-is", val: types.StringValue("https://localhost:8080"), wantSet: true, wantValue: "https://localhost:8080"},
		{name: "Trailing slash is trimmed", val: types.StringValue("https://localhost:8080/"), wantSet: true, wantValue: "https://localhost:8080"},
		{name: "Multiple trailing slashes are trimmed", val: types.StringValue("https://localhost:8080///"), wantSet: true, wantValue: "https://localhost:8080"},
		{name: "Surrounding whitespace is trimmed", val: types.StringValue("  https://localhost:8080  "), wantSet: true, wantValue: "https://localhost:8080"},
		{name: "Whitespace and trailing slash are trimmed", val: types.StringValue("  https://localhost:8080/  "), wantSet: true, wantValue: "https://localhost:8080"},
		{name: "Empty string is skipped", val: types.StringValue(""), wantSet: false},
		{name: "Whitespace only is skipped", val: types.StringValue("   "), wantSet: false},
		{name: "Null is skipped", val: types.StringNull(), wantSet: false},
		{name: "Unknown is skipped", val: types.StringUnknown(), wantSet: false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			m := make(map[string]string)
			setEndpoint(m, service, tc.val)

			got, ok := m[service]
			assert.Equal(t, tc.wantSet, ok, "unexpected presence of endpoint in map")
			if tc.wantSet {
				assert.Equal(t, tc.wantValue, got)
			}
		})
	}
}
