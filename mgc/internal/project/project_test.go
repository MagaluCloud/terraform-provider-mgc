package project

import (
	"context"
	"net/http"
	"net/url"
	"strings"
	"testing"

	clientSDK "github.com/MagaluCloud/mgc-sdk-go/client"
	"github.com/MagaluCloud/terraform-provider-mgc/mgc/internal/importid"
	"github.com/MagaluCloud/terraform-provider-mgc/mgc/utils"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
)

const testProjectID = "550e8400-e29b-41d4-a716-446655440000"

func TestIDValidator(t *testing.T) {
	testCases := []struct {
		name          string
		value         types.String
		expectedValid bool
	}{
		{name: "Valid lowercase UUID", value: types.StringValue(testProjectID), expectedValid: true},
		{name: "Valid uppercase UUID", value: types.StringValue("550E8400-E29B-41D4-A716-446655440000"), expectedValid: true},
		{name: "Valid UUID of a version other than 4", value: types.StringValue("018f3a2b-7c4d-7e5f-8a9b-0c1d2e3f4a5b"), expectedValid: true},
		{name: "Missing a group", value: types.StringValue("550e8400-e29b-41d4-a716"), expectedValid: false},
		{name: "Surrounding whitespace", value: types.StringValue(" " + testProjectID + " "), expectedValid: false},
		{name: "Project name instead of id", value: types.StringValue("my-project"), expectedValid: false},
		{name: "Empty string", value: types.StringValue(""), expectedValid: false},
		{name: "Null is left to Optional", value: types.StringNull(), expectedValid: true},
		{name: "Unknown is validated later", value: types.StringUnknown(), expectedValid: true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := validator.StringRequest{Path: path.Root("project_id"), ConfigValue: tc.value}
			resp := &validator.StringResponse{Diagnostics: diag.Diagnostics{}}

			idValidator().ValidateString(context.Background(), req, resp)

			if tc.expectedValid {
				assert.Empty(t, resp.Diagnostics)
			} else {
				assert.NotEmpty(t, resp.Diagnostics)
			}
		})
	}
}

func TestNamespaceValidatesProjectID(t *testing.T) {
	parsed, err := importid.Parse("project/"+testProjectID+"/resource/abc-123", Namespace)
	assert.NoError(t, err)
	assert.Equal(t, types.StringValue(testProjectID), parsed.Value(Namespace))

	_, err = importid.Parse("project/my-project/resource/abc-123", Namespace)
	assert.ErrorContains(t, err, "valid project ID")
}

func TestCallOptions(t *testing.T) {
	testCases := []struct {
		name     string
		value    types.String
		expected clientSDK.CallOptions
	}{
		{name: "Project id is carried", value: types.StringValue(testProjectID), expected: clientSDK.CallOptions{ProjectID: testProjectID}},
		{name: "Null targets the default project", value: types.StringNull(), expected: clientSDK.CallOptions{}},
		{name: "Unknown targets the default project", value: types.StringUnknown(), expected: clientSDK.CallOptions{}},
		{name: "Blank targets the default project", value: types.StringValue("   "), expected: clientSDK.CallOptions{}},
		{name: "Surrounding whitespace is trimmed", value: types.StringValue(" " + testProjectID + " "), expected: clientSDK.CallOptions{ProjectID: testProjectID}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, CallOptions(tc.value))
		})
	}
}

func TestResourceAttribute(t *testing.T) {
	attr := ResourceAttribute()

	assert.True(t, attr.Optional)
	assert.True(t, attr.Computed)
	assert.False(t, attr.Required)
	assert.Contains(t, attr.Description, "removing it keeps the current project")
	assert.Len(t, attr.Validators, 1)
	assert.Equal(t, []any{keepWhenUnset{}}, []any{attr.PlanModifiers[0]})
}

func TestDataSourceAttribute(t *testing.T) {
	attr := DataSourceAttribute()

	assert.True(t, attr.Optional)
	assert.False(t, attr.Computed)
	assert.False(t, attr.Required)
	assert.Len(t, attr.Validators, 1)
}

func TestIdentityAttribute(t *testing.T) {
	attr := IdentityAttribute()

	assert.True(t, attr.OptionalForImport)
	assert.False(t, attr.RequiredForImport)
}

func TestParseSDKError(t *testing.T) {
	reqURL, _ := url.Parse("http://example.com/v0/clusters/abc")
	httpErr := func(status int) error {
		return &clientSDK.HTTPError{
			StatusCode: status,
			Status:     http.StatusText(status),
			Body:       []byte(`{"error":"x"}`),
			Response:   &http.Response{Request: &http.Request{URL: reqURL}, Header: http.Header{}},
		}
	}

	testCases := []struct {
		name         string
		err          error
		projectID    types.String
		expectedHint string
	}{
		{name: "Not found in the default project suggests a project", err: httpErr(http.StatusNotFound), projectID: types.StringNull(), expectedHint: "project/{project_id}/resource/{id}"},
		{name: "Not found in a project names the project", err: httpErr(http.StatusNotFound), projectID: types.StringValue(testProjectID), expectedHint: `project "` + testProjectID + `"`},
		{name: "Other status codes are left untouched", err: httpErr(http.StatusInternalServerError), projectID: types.StringNull()},
		{name: "Non HTTP errors are left untouched", err: &clientSDK.ValidationError{Field: "id", Message: "empty"}, projectID: types.StringNull()},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			baseMsg, baseDetail := utils.ParseSDKError(tc.err)

			msg, detail := ParseSDKError(tc.err, tc.projectID)

			assert.Equal(t, baseMsg, msg)
			if tc.expectedHint == "" {
				assert.Equal(t, baseDetail, detail)
				return
			}
			assert.True(t, strings.HasPrefix(detail, baseDetail))
			assert.Contains(t, detail, tc.expectedHint)
		})
	}
}
