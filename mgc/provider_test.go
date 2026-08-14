package mgc

import (
	"context"
	"testing"
	"time"

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

func TestDurationValidator(t *testing.T) {
	testCases := []struct {
		name          string
		value         string
		expectedValid bool
	}{
		{name: "Valid minutes", value: "1m", expectedValid: true},
		{name: "Valid milliseconds", value: "500ms", expectedValid: true},
		{name: "Valid hours", value: "2h", expectedValid: true},
		{name: "Valid compound", value: "160m", expectedValid: true},
		{name: "Valid surrounded by whitespace", value: "  1m  ", expectedValid: true},
		{name: "Empty string is allowed", value: "", expectedValid: true},
		{name: "Whitespace only is allowed", value: "   ", expectedValid: true},
		{name: "Missing unit", value: "160", expectedValid: false},
		{name: "Plain text", value: "soon", expectedValid: false},
		{name: "Zero is rejected", value: "0s", expectedValid: false},
		{name: "Negative is rejected", value: "-1m", expectedValid: false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := validator.StringRequest{
				Path:        path.Root("test"),
				ConfigValue: types.StringValue(tc.value),
			}
			resp := &validator.StringResponse{Diagnostics: diag.Diagnostics{}}

			durationValidator{}.ValidateString(context.Background(), req, resp)

			if tc.expectedValid {
				assert.Empty(t, resp.Diagnostics, "expected no diagnostics for %q", tc.value)
			} else {
				assert.NotEmpty(t, resp.Diagnostics, "expected diagnostics for %q", tc.value)
			}
		})
	}
}

func TestDurationValidatorNullAndUnknown(t *testing.T) {
	for _, cv := range []types.String{types.StringNull(), types.StringUnknown()} {
		req := validator.StringRequest{Path: path.Root("test"), ConfigValue: cv}
		resp := &validator.StringResponse{Diagnostics: diag.Diagnostics{}}
		durationValidator{}.ValidateString(context.Background(), req, resp)
		assert.Empty(t, resp.Diagnostics)
	}
}

func TestParseDurationOr0(t *testing.T) {
	testCases := []struct {
		name  string
		value types.String
		want  time.Duration
	}{
		{name: "Minutes", value: types.StringValue("1m"), want: time.Minute},
		{name: "Milliseconds", value: types.StringValue("500ms"), want: 500 * time.Millisecond},
		{name: "Hours", value: types.StringValue("2h"), want: 2 * time.Hour},
		{name: "Trimmed", value: types.StringValue("  160m  "), want: 160 * time.Minute},
		{name: "Empty is zero", value: types.StringValue(""), want: 0},
		{name: "Invalid is zero", value: types.StringValue("soon"), want: 0},
		{name: "Zero string is zero", value: types.StringValue("0s"), want: 0},
		{name: "Negative is zero", value: types.StringValue("-1m"), want: 0},
		{name: "Null is zero", value: types.StringNull(), want: 0},
		{name: "Unknown is zero", value: types.StringUnknown(), want: 0},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, parseDurationOr0(tc.value))
		})
	}
}

func TestNewConfigDataPolling(t *testing.T) {
	base := func() ProviderModel {
		return ProviderModel{
			ApiKey: types.StringValue("00000000-0000-4000-8000-000000000000"),
			Region: types.StringValue("br-se1"),
			Env:    types.StringValue("prod"),
		}
	}

	t.Run("set values are parsed into the config", func(t *testing.T) {
		plan := base()
		plan.PollingInterval = types.StringValue("30s")
		plan.PollingTimeout = types.StringValue("10m")

		cfg := NewConfigData(plan, "test")

		assert.Equal(t, 30*time.Second, cfg.PollingInterval)
		assert.Equal(t, 10*time.Minute, cfg.PollingTimeout)
	})

	t.Run("unset values leave zero so resources use their defaults", func(t *testing.T) {
		cfg := NewConfigData(base(), "test")

		assert.Equal(t, time.Duration(0), cfg.PollingInterval)
		assert.Equal(t, time.Duration(0), cfg.PollingTimeout)
	})
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
