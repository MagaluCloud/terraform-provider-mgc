package project

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/stretchr/testify/assert"
)

func identityWith(ctx context.Context, values map[string]tftypes.Value) *tfsdk.ResourceIdentity {
	schema := IdentitySchema()
	var raw tftypes.Value
	if values == nil {
		raw = tftypes.NewValue(schema.Type().TerraformType(ctx), nil)
	} else {
		raw = tftypes.NewValue(schema.Type().TerraformType(ctx), values)
	}
	return &tfsdk.ResourceIdentity{Schema: schema, Raw: raw}
}

func TestResolveImport(t *testing.T) {
	ctx := context.Background()

	testCases := []struct {
		name          string
		req           resource.ImportStateRequest
		expected      IdentityModel
		expectedError bool
	}{
		{
			name:     "Import ID without project",
			req:      resource.ImportStateRequest{ID: "abc-123"},
			expected: IdentityModel{ID: types.StringValue("abc-123"), ProjectID: types.StringNull()},
		},
		{
			name:     "Import ID with project",
			req:      resource.ImportStateRequest{ID: "project/" + testProjectID + "/abc-123"},
			expected: IdentityModel{ID: types.StringValue("abc-123"), ProjectID: types.StringValue(testProjectID)},
		},
		{
			name:          "Malformed import ID",
			req:           resource.ImportStateRequest{ID: "region/br-ne1/abc-123"},
			expectedError: true,
		},
		{
			name:          "Neither import ID nor identity",
			req:           resource.ImportStateRequest{},
			expectedError: true,
		},
		{
			name: "Identity with project",
			req: resource.ImportStateRequest{Identity: identityWith(ctx, map[string]tftypes.Value{
				"id":         tftypes.NewValue(tftypes.String, "abc-123"),
				"project_id": tftypes.NewValue(tftypes.String, testProjectID),
			})},
			expected: IdentityModel{ID: types.StringValue("abc-123"), ProjectID: types.StringValue(testProjectID)},
		},
		{
			name: "Identity without project",
			req: resource.ImportStateRequest{Identity: identityWith(ctx, map[string]tftypes.Value{
				"id":         tftypes.NewValue(tftypes.String, "abc-123"),
				"project_id": tftypes.NewValue(tftypes.String, nil),
			})},
			expected: IdentityModel{ID: types.StringValue("abc-123"), ProjectID: types.StringNull()},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, diags := ResolveImport(ctx, tc.req)

			if tc.expectedError {
				assert.True(t, diags.HasError())
				return
			}

			assert.False(t, diags.HasError(), "%v", diags)
			assert.Equal(t, tc.expected, got)
		})
	}
}

func TestSetIdentity(t *testing.T) {
	ctx := context.Background()

	for _, model := range []IdentityModel{
		{ID: types.StringValue("abc-123"), ProjectID: types.StringValue(testProjectID)},
		{ID: types.StringValue("abc-123"), ProjectID: types.StringNull()},
	} {
		identity := identityWith(ctx, nil)

		assert.False(t, SetIdentity(ctx, identity, model.ID, model.ProjectID).HasError())

		var got IdentityModel
		assert.False(t, identity.Get(ctx, &got).HasError())
		assert.Equal(t, model, got)
	}
}

func TestSetIdentityWithoutIdentitySupport(t *testing.T) {
	assert.False(t, SetIdentity(context.Background(), nil, types.StringValue("abc-123"), types.StringNull()).HasError())
}

func TestIdentitySchema(t *testing.T) {
	schema := IdentitySchema()

	id, ok := schema.Attributes["id"]
	assert.True(t, ok)
	assert.True(t, id.IsRequiredForImport())

	project, ok := schema.Attributes["project_id"]
	assert.True(t, ok)
	assert.True(t, project.IsOptionalForImport())
}
