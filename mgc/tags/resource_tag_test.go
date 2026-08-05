package tags

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	tagSDK "github.com/MagaluCloud/mgc-sdk-go/tag"
	"github.com/MagaluCloud/terraform-provider-mgc/mgc/internal/mocks"
	"github.com/MagaluCloud/terraform-provider-mgc/mgc/internal/testutils"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// tagFromJSON builds a tag out of a raw API response, so the fixtures are the
// payloads the API really returns, timestamps included.
func tagFromJSON(t *testing.T, raw string) tagSDK.Tag {
	t.Helper()

	var tag tagSDK.Tag
	require.NoError(t, json.Unmarshal([]byte(raw), &tag))
	return tag
}

func ptr[T any](v T) *T {
	return &v
}

func kindsSet(kinds ...string) types.Set {
	values := make([]attr.Value, 0, len(kinds))
	for _, kind := range kinds {
		values = append(values, types.StringValue(kind))
	}
	return types.SetValueMust(types.StringType, values)
}

func TestFlattenTag(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		prior    tagResourceModel
		response string
		expected tagResourceModel
	}{
		{
			name: "api echoes the config",
			prior: tagResourceModel{
				Name:        types.StringValue("tagfin"),
				Description: types.StringValue("monitoramento"),
				Color:       newCaseInsensitiveString("0086ff"),
				Kinds:       kindsSet("finops"),
			},
			response: `{
				"name": "tagfin",
				"description": "monitoramento",
				"color": "0086ff",
				"kinds": ["finops"],
				"created_at": "2026-08-03T00:57:47.908658",
				"updated_at": null,
				"values": []
			}`,
			expected: tagResourceModel{
				ID:          types.StringValue("tagfin"),
				Name:        types.StringValue("tagfin"),
				Description: types.StringValue("monitoramento"),
				Color:       newCaseInsensitiveString("0086ff"),
				Kinds:       kindsSet("finops"),
				CreatedAt:   types.StringValue("2026-08-03T00:57:47Z"),
				UpdatedAt:   types.StringNull(),
			},
		},
		{
			// The custom type holds the semantic equality, so flatten is a
			// pass-through and the framework keeps the planned casing.
			name: "color lowercased by the api",
			prior: tagResourceModel{
				Name:  types.StringValue("tagfin"),
				Color: newCaseInsensitiveString("FF0000"),
			},
			response: `{
				"name": "tagfin",
				"description": null,
				"color": "ff0000",
				"kinds": [],
				"created_at": "2026-08-03T00:57:47.908658",
				"updated_at": null,
				"values": []
			}`,
			expected: tagResourceModel{
				ID:        types.StringValue("tagfin"),
				Name:      types.StringValue("tagfin"),
				Color:     newCaseInsensitiveString("ff0000"),
				CreatedAt: types.StringValue("2026-08-03T00:57:47Z"),
				UpdatedAt: types.StringNull(),
			},
		},
		{
			name: "optional fields absent on both sides stay null",
			prior: tagResourceModel{
				Name: types.StringValue("tagfin"),
			},
			response: `{
				"name": "tagfin",
				"description": null,
				"color": "0086ff",
				"kinds": [],
				"created_at": "2026-08-03T00:57:47.908658",
				"updated_at": null,
				"values": []
			}`,
			expected: tagResourceModel{
				ID:        types.StringValue("tagfin"),
				Name:      types.StringValue("tagfin"),
				Color:     newCaseInsensitiveString("0086ff"),
				CreatedAt: types.StringValue("2026-08-03T00:57:47Z"),
				UpdatedAt: types.StringNull(),
			},
		},
		{
			// The API stores "" and null as distinct values, so a description
			// emptied out of band must not flip a null config into "".
			name: "empty description from the api keeps the config null",
			prior: tagResourceModel{
				Name: types.StringValue("tagfin"),
			},
			response: `{
				"name": "tagfin",
				"description": "",
				"color": "0086ff",
				"kinds": [],
				"created_at": "2026-08-03T00:57:47.908658",
				"updated_at": "2026-08-03T01:14:56.834488",
				"values": []
			}`,
			expected: tagResourceModel{
				ID:        types.StringValue("tagfin"),
				Name:      types.StringValue("tagfin"),
				Color:     newCaseInsensitiveString("0086ff"),
				CreatedAt: types.StringValue("2026-08-03T00:57:47Z"),
				UpdatedAt: types.StringValue("2026-08-03T01:14:56Z"),
			},
		},
		{
			name: "kinds set out of band show up in the state",
			prior: tagResourceModel{
				Name: types.StringValue("tagfin"),
			},
			response: `{
				"name": "tagfin",
				"description": null,
				"color": "0086ff",
				"kinds": ["finops"],
				"created_at": "2026-08-03T00:57:47.908658",
				"updated_at": null,
				"values": []
			}`,
			expected: tagResourceModel{
				ID:        types.StringValue("tagfin"),
				Name:      types.StringValue("tagfin"),
				Color:     newCaseInsensitiveString("0086ff"),
				Kinds:     kindsSet("finops"),
				CreatedAt: types.StringValue("2026-08-03T00:57:47Z"),
				UpdatedAt: types.StringNull(),
			},
		},
		{
			name: "color never set is null",
			prior: tagResourceModel{
				Name: types.StringValue("tagfin"),
			},
			response: `{
				"name": "tagfin",
				"description": null,
				"color": null,
				"kinds": [],
				"created_at": "2026-08-03T00:57:47.908658",
				"updated_at": null,
				"values": []
			}`,
			expected: tagResourceModel{
				ID:        types.StringValue("tagfin"),
				Name:      types.StringValue("tagfin"),
				Color:     caseInsensitiveStringValue{StringValue: types.StringNull()},
				CreatedAt: types.StringValue("2026-08-03T00:57:47Z"),
				UpdatedAt: types.StringNull(),
			},
		},
		{
			// The API is case sensitive: a rename by casing is a different tag.
			name: "name is echoed as stored",
			prior: tagResourceModel{
				Name: types.StringValue("tagFin"),
			},
			response: `{
				"name": "tagFin",
				"description": null,
				"color": "0086ff",
				"kinds": [],
				"created_at": "2026-08-03T00:57:47.908658",
				"updated_at": null,
				"values": []
			}`,
			expected: tagResourceModel{
				ID:        types.StringValue("tagFin"),
				Name:      types.StringValue("tagFin"),
				Color:     newCaseInsensitiveString("0086ff"),
				CreatedAt: types.StringValue("2026-08-03T00:57:47Z"),
				UpdatedAt: types.StringNull(),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := flattenTag(tt.prior, tagFromJSON(t, tt.response))

			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestBuildCreateTagRequest(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		plan     tagResourceModel
		expected tagSDK.CreateTagRequest
	}{
		{
			name: "every field set",
			plan: tagResourceModel{
				Name:        types.StringValue("tagfin"),
				Description: types.StringValue("monitoramento"),
				Color:       newCaseInsensitiveString("FF0000"),
				Kinds:       kindsSet("finops"),
			},
			expected: tagSDK.CreateTagRequest{
				Name:        "tagfin",
				Description: ptr("monitoramento"),
				Color:       ptr("FF0000"),
				Kinds:       []tagSDK.TagKind{"finops"},
			},
		},
		{
			// color is Optional+Computed, so it is unknown when absent from the
			// config; sending nil lets the API apply its own default.
			name: "only the required name",
			plan: tagResourceModel{
				Name:        types.StringValue("tagfin"),
				Description: types.StringNull(),
				Color:       caseInsensitiveStringValue{StringValue: types.StringUnknown()},
				Kinds:       types.SetNull(types.StringType),
			},
			expected: tagSDK.CreateTagRequest{
				Name: "tagfin",
			},
		},
		{
			name: "empty description is sent as empty, not dropped",
			plan: tagResourceModel{
				Name:        types.StringValue("tagfin"),
				Description: types.StringValue(""),
				Color:       caseInsensitiveStringValue{StringValue: types.StringUnknown()},
				Kinds:       types.SetNull(types.StringType),
			},
			expected: tagSDK.CreateTagRequest{
				Name:        "tagfin",
				Description: ptr(""),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.expected, buildCreateTagRequest(tt.plan))
		})
	}
}

func TestBuildUpdateTagRequest(t *testing.T) {
	t.Parallel()

	withDescription := func(d string) tagResourceModel {
		return tagResourceModel{
			Name:        types.StringValue("tagfin"),
			Description: types.StringValue(d),
			Color:       newCaseInsensitiveString("0086ff"),
			Kinds:       kindsSet("finops"),
		}
	}

	tests := []struct {
		name     string
		state    tagResourceModel
		plan     tagResourceModel
		expected tagSDK.UpdateTagRequest
	}{
		{
			// Every update states the desired description, so the request never
			// leaves the reader guessing between "keep" and "clear".
			name:  "description changed",
			state: withDescription("antigo"),
			plan:  withDescription("novo"),
			expected: tagSDK.UpdateTagRequest{
				Description: ptr("novo"),
			},
		},
		{
			name:  "description removed from the config is cleared",
			state: withDescription("antigo"),
			plan: tagResourceModel{
				Name:        types.StringValue("tagfin"),
				Description: types.StringNull(),
				Color:       newCaseInsensitiveString("0086ff"),
				Kinds:       kindsSet("finops"),
			},
			expected: tagSDK.UpdateTagRequest{
				Description: ptr(""),
			},
		},
		{
			name:  "color changed",
			state: withDescription("mesmo"),
			plan: tagResourceModel{
				Name:        types.StringValue("tagfin"),
				Description: types.StringValue("mesmo"),
				Color:       newCaseInsensitiveString("ff0000"),
				Kinds:       kindsSet("finops"),
			},
			expected: tagSDK.UpdateTagRequest{
				Description: ptr("mesmo"),
				Color:       ptr("ff0000"),
			},
		},
		{
			// The API rejects a null color on update, so dropping it from the
			// config keeps whatever the tag has today.
			name:  "color removed from the config is not sent",
			state: withDescription("mesmo"),
			plan: tagResourceModel{
				Name:        types.StringValue("tagfin"),
				Description: types.StringValue("mesmo"),
				Color:       caseInsensitiveStringValue{StringValue: types.StringNull()},
				Kinds:       kindsSet("finops"),
			},
			expected: tagSDK.UpdateTagRequest{
				Description: ptr("mesmo"),
			},
		},
		{
			name:  "kinds cleared",
			state: withDescription("mesmo"),
			plan: tagResourceModel{
				Name:        types.StringValue("tagfin"),
				Description: types.StringValue("mesmo"),
				Color:       newCaseInsensitiveString("0086ff"),
				Kinds:       types.SetNull(types.StringType),
			},
			expected: tagSDK.UpdateTagRequest{
				Description: ptr("mesmo"),
				Kinds:       &[]tagSDK.TagKind{},
			},
		},
		{
			name: "kinds added",
			state: tagResourceModel{
				Name:        types.StringValue("tagfin"),
				Description: types.StringValue("mesmo"),
				Color:       newCaseInsensitiveString("0086ff"),
				Kinds:       types.SetNull(types.StringType),
			},
			plan: withDescription("mesmo"),
			expected: tagSDK.UpdateTagRequest{
				Description: ptr("mesmo"),
				Kinds:       &[]tagSDK.TagKind{"finops"},
			},
		},
		{
			name:  "nothing changed sends only the echoed description",
			state: withDescription("mesmo"),
			plan:  withDescription("mesmo"),
			expected: tagSDK.UpdateTagRequest{
				Description: ptr("mesmo"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.expected, buildUpdateTagRequest(tt.state, tt.plan))
		})
	}
}

// emptyState mirrors what the framework hands to ImportState: an object whose
// attributes are all null, not an uninitialized value.
func emptyState(ctx context.Context, tagSchema schema.Schema) tfsdk.State {
	objectType := tagSchema.Type().TerraformType(ctx).(tftypes.Object)

	attributes := make(map[string]tftypes.Value, len(objectType.AttributeTypes))
	for name, attributeType := range objectType.AttributeTypes {
		attributes[name] = tftypes.NewValue(attributeType, nil)
	}

	return tfsdk.State{Schema: tagSchema, Raw: tftypes.NewValue(objectType, attributes)}
}

func newTestTagResource(t *testing.T, svc tagSDK.TagService) (*tagResource, schema.Schema) {
	t.Helper()

	r := &tagResource{tags: svc}
	return r, testutils.GetResourceTestSchema(t, r).Schema
}

func TestTagResourceMetadata(t *testing.T) {
	t.Parallel()

	resp := &resource.MetadataResponse{}
	NewTagResource().Metadata(context.Background(), resource.MetadataRequest{ProviderTypeName: "mgc"}, resp)

	assert.Equal(t, "mgc_tag", resp.TypeName)
}

func TestTagResourceSchema(t *testing.T) {
	t.Parallel()

	_, tagSchema := newTestTagResource(t, nil)

	for _, attribute := range []string{"id", "name", "description", "color", "kinds", "created_at", "updated_at"} {
		assert.Contains(t, tagSchema.Attributes, attribute)
	}

	name := tagSchema.Attributes["name"].(schema.StringAttribute)
	assert.True(t, name.Required, "the API has no rename")
	assert.Len(t, name.PlanModifiers, 1, "name must require replacement")

	color := tagSchema.Attributes["color"].(schema.StringAttribute)
	assert.True(t, color.Optional)
	assert.True(t, color.Computed, "the API assigns a color when none is given")
	assert.Equal(t, caseInsensitiveStringType{}, color.CustomType)

	// updated_at changes on every update, so freezing it with the state value
	// would break an in-place update.
	updatedAt := tagSchema.Attributes["updated_at"].(schema.StringAttribute)
	assert.Empty(t, updatedAt.PlanModifiers)
}

func TestTagResourceCreate(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	created := tagFromJSON(t, `{
		"name": "tagfin",
		"description": "monitoramento",
		"color": "0086ff",
		"kinds": ["finops"],
		"created_at": "2026-08-03T00:57:47.908658",
		"updated_at": null,
		"values": []
	}`)

	mockSvc := new(mocks.TagService)
	mockSvc.On("Create", ctx, tagSDK.CreateTagRequest{
		Name:        "tagfin",
		Description: ptr("monitoramento"),
		Kinds:       []tagSDK.TagKind{"finops"},
	}).Return(&created, nil)

	r, tagSchema := newTestTagResource(t, mockSvc)

	plan := tfsdk.Plan{Schema: tagSchema}
	require.False(t, plan.Set(ctx, &tagResourceModel{
		ID:          types.StringUnknown(),
		Name:        types.StringValue("tagfin"),
		Description: types.StringValue("monitoramento"),
		Color:       caseInsensitiveStringValue{StringValue: types.StringUnknown()},
		Kinds:       kindsSet("finops"),
		CreatedAt:   types.StringUnknown(),
		UpdatedAt:   types.StringUnknown(),
	}).HasError())

	resp := &resource.CreateResponse{State: tfsdk.State{Schema: tagSchema}}
	r.Create(ctx, resource.CreateRequest{Plan: plan}, resp)

	require.False(t, resp.Diagnostics.HasError(), "Create returned errors: %v", resp.Diagnostics)

	var state tagResourceModel
	resp.State.Get(ctx, &state)

	assert.Equal(t, "tagfin", state.ID.ValueString())
	assert.Equal(t, "0086ff", state.Color.ValueString(), "the color assigned by the API lands in the state")
	assert.Equal(t, "2026-08-03T00:57:47Z", state.CreatedAt.ValueString())
	assert.True(t, state.UpdatedAt.IsNull())
	mockSvc.AssertExpectations(t)
}

func TestTagResourceCreateConflict(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	mockSvc := new(mocks.TagService)
	mockSvc.On("Create", ctx, mock.Anything).
		Return(nil, httpError(http.StatusConflict, `{"message":"Conflict","detail":"Entity already exists"}`))

	r, tagSchema := newTestTagResource(t, mockSvc)

	plan := tfsdk.Plan{Schema: tagSchema}
	require.False(t, plan.Set(ctx, &tagResourceModel{
		ID:          types.StringUnknown(),
		Name:        types.StringValue("tagfin"),
		Description: types.StringNull(),
		Color:       caseInsensitiveStringValue{StringValue: types.StringUnknown()},
		Kinds:       types.SetNull(types.StringType),
		CreatedAt:   types.StringUnknown(),
		UpdatedAt:   types.StringUnknown(),
	}).HasError())

	resp := &resource.CreateResponse{State: tfsdk.State{Schema: tagSchema}}
	r.Create(ctx, resource.CreateRequest{Plan: plan}, resp)

	require.True(t, resp.Diagnostics.HasError())
	assert.Contains(t, resp.Diagnostics.Errors()[0].Detail(), "terraform import", "the error has to say how to adopt the existing tag")
	mockSvc.AssertExpectations(t)
}

func TestTagResourceRead(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	// Someone changed the tag out of band; the refresh has to show it.
	current := tagFromJSON(t, `{
		"name": "tagfin",
		"description": "outro",
		"color": "ff0000",
		"kinds": [],
		"created_at": "2026-08-03T00:57:47.908658",
		"updated_at": "2026-08-03T01:14:56.834488",
		"values": []
	}`)

	mockSvc := new(mocks.TagService)
	mockSvc.On("Get", ctx, "tagfin").Return(&current, nil)

	r, tagSchema := newTestTagResource(t, mockSvc)

	priorState := tfsdk.State{Schema: tagSchema}
	require.False(t, priorState.Set(ctx, &tagResourceModel{
		ID:          types.StringValue("tagfin"),
		Name:        types.StringValue("tagfin"),
		Description: types.StringValue("monitoramento"),
		Color:       newCaseInsensitiveString("0086ff"),
		Kinds:       types.SetNull(types.StringType),
		CreatedAt:   types.StringValue("2026-08-03T00:57:47Z"),
		UpdatedAt:   types.StringNull(),
	}).HasError())

	resp := &resource.ReadResponse{State: tfsdk.State{Schema: tagSchema}}
	r.Read(ctx, resource.ReadRequest{State: priorState}, resp)

	require.False(t, resp.Diagnostics.HasError(), "Read returned errors: %v", resp.Diagnostics)

	var state tagResourceModel
	resp.State.Get(ctx, &state)

	assert.Equal(t, "outro", state.Description.ValueString())
	assert.Equal(t, "ff0000", state.Color.ValueString())
	assert.Equal(t, "2026-08-03T01:14:56Z", state.UpdatedAt.ValueString())
	mockSvc.AssertExpectations(t)
}

func TestTagResourceReadNotFound(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  error
	}{
		{
			name: "404",
			err:  httpError(http.StatusNotFound, `{"message":"Not Found","detail":"Tag not found"}`),
		},
		{
			// The API still answers 400 in some not-found cases.
			name: "400",
			err:  httpError(http.StatusBadRequest, `{"message":"Bad Request","detail":"Tag was not found"}`),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := context.Background()

			mockSvc := new(mocks.TagService)
			mockSvc.On("Get", ctx, "tagfin").Return(nil, tt.err)

			r, tagSchema := newTestTagResource(t, mockSvc)

			priorState := tfsdk.State{Schema: tagSchema}
			require.False(t, priorState.Set(ctx, &tagResourceModel{
				ID:          types.StringValue("tagfin"),
				Name:        types.StringValue("tagfin"),
				Description: types.StringNull(),
				Color:       newCaseInsensitiveString("0086ff"),
				Kinds:       types.SetNull(types.StringType),
				CreatedAt:   types.StringValue("2026-08-03T00:57:47Z"),
				UpdatedAt:   types.StringNull(),
			}).HasError())

			resp := &resource.ReadResponse{State: tfsdk.State{Schema: tagSchema}}
			r.Read(ctx, resource.ReadRequest{State: priorState}, resp)

			assert.False(t, resp.Diagnostics.HasError(), "a missing tag is not an error: %v", resp.Diagnostics)
			assert.Equal(t, 1, resp.Diagnostics.WarningsCount())
			assert.True(t, resp.State.Raw.IsNull(), "the tag must be dropped from the state so it gets recreated")
			mockSvc.AssertExpectations(t)
		})
	}
}

func TestTagResourceUpdate(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	updated := tagFromJSON(t, `{
		"name": "tagfin",
		"description": "novo",
		"color": "0086ff",
		"kinds": ["finops"],
		"created_at": "2026-08-03T00:57:47.908658",
		"updated_at": "2026-08-03T01:14:56.834488",
		"values": []
	}`)

	mockSvc := new(mocks.TagService)
	mockSvc.On("Update", ctx, "tagfin", tagSDK.UpdateTagRequest{Description: ptr("novo")}).Return(&updated, nil)

	r, tagSchema := newTestTagResource(t, mockSvc)

	priorState := tfsdk.State{Schema: tagSchema}
	require.False(t, priorState.Set(ctx, &tagResourceModel{
		ID:          types.StringValue("tagfin"),
		Name:        types.StringValue("tagfin"),
		Description: types.StringValue("antigo"),
		Color:       newCaseInsensitiveString("0086ff"),
		Kinds:       kindsSet("finops"),
		CreatedAt:   types.StringValue("2026-08-03T00:57:47Z"),
		UpdatedAt:   types.StringNull(),
	}).HasError())

	plan := tfsdk.Plan{Schema: tagSchema}
	require.False(t, plan.Set(ctx, &tagResourceModel{
		ID:          types.StringValue("tagfin"),
		Name:        types.StringValue("tagfin"),
		Description: types.StringValue("novo"),
		Color:       newCaseInsensitiveString("0086ff"),
		Kinds:       kindsSet("finops"),
		CreatedAt:   types.StringValue("2026-08-03T00:57:47Z"),
		UpdatedAt:   types.StringUnknown(),
	}).HasError())

	resp := &resource.UpdateResponse{State: tfsdk.State{Schema: tagSchema}}
	r.Update(ctx, resource.UpdateRequest{Plan: plan, State: priorState}, resp)

	require.False(t, resp.Diagnostics.HasError(), "Update returned errors: %v", resp.Diagnostics)

	var state tagResourceModel
	resp.State.Get(ctx, &state)

	assert.Equal(t, "novo", state.Description.ValueString())
	assert.Equal(t, "2026-08-03T01:14:56Z", state.UpdatedAt.ValueString(), "updated_at must not stay unknown")
	mockSvc.AssertExpectations(t)
}

func TestTagResourceDelete(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  error
	}{
		{name: "deleted", err: nil},
		{name: "already gone", err: httpError(http.StatusNotFound, "")},
		{name: "already gone with 400", err: httpError(http.StatusBadRequest, `{"detail":"Tag was not found"}`)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := context.Background()

			mockSvc := new(mocks.TagService)
			mockSvc.On("Delete", ctx, "tagfin").Return(tt.err)

			r, tagSchema := newTestTagResource(t, mockSvc)

			priorState := tfsdk.State{Schema: tagSchema}
			require.False(t, priorState.Set(ctx, &tagResourceModel{
				ID:          types.StringValue("tagfin"),
				Name:        types.StringValue("tagfin"),
				Description: types.StringNull(),
				Color:       newCaseInsensitiveString("0086ff"),
				Kinds:       types.SetNull(types.StringType),
				CreatedAt:   types.StringValue("2026-08-03T00:57:47Z"),
				UpdatedAt:   types.StringNull(),
			}).HasError())

			resp := &resource.DeleteResponse{State: tfsdk.State{Schema: tagSchema}}
			r.Delete(ctx, resource.DeleteRequest{State: priorState}, resp)

			assert.False(t, resp.Diagnostics.HasError(), "delete must be idempotent: %v", resp.Diagnostics)
			mockSvc.AssertExpectations(t)
		})
	}
}

func TestTagResourceDeleteError(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	mockSvc := new(mocks.TagService)
	mockSvc.On("Delete", ctx, "tagfin").Return(httpError(http.StatusInternalServerError, ""))

	r, tagSchema := newTestTagResource(t, mockSvc)

	priorState := tfsdk.State{Schema: tagSchema}
	require.False(t, priorState.Set(ctx, &tagResourceModel{
		ID:          types.StringValue("tagfin"),
		Name:        types.StringValue("tagfin"),
		Description: types.StringNull(),
		Color:       newCaseInsensitiveString("0086ff"),
		Kinds:       types.SetNull(types.StringType),
		CreatedAt:   types.StringValue("2026-08-03T00:57:47Z"),
		UpdatedAt:   types.StringNull(),
	}).HasError())

	resp := &resource.DeleteResponse{State: tfsdk.State{Schema: tagSchema}}
	r.Delete(ctx, resource.DeleteRequest{State: priorState}, resp)

	assert.True(t, resp.Diagnostics.HasError(), "a real failure must not be swallowed")
	mockSvc.AssertExpectations(t)
}

func TestTagResourceImportState(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	r, tagSchema := newTestTagResource(t, nil)

	resp := &resource.ImportStateResponse{State: emptyState(ctx, tagSchema)}
	r.ImportState(ctx, resource.ImportStateRequest{ID: "tagfin"}, resp)

	require.False(t, resp.Diagnostics.HasError(), "ImportState returned errors: %v", resp.Diagnostics)

	var name types.String
	resp.State.GetAttribute(ctx, path.Root("name"), &name)
	assert.Equal(t, "tagfin", name.ValueString())
}

func TestTagResourceImportStateEmpty(t *testing.T) {
	t.Parallel()

	r, tagSchema := newTestTagResource(t, nil)

	resp := &resource.ImportStateResponse{State: emptyState(context.Background(), tagSchema)}
	r.ImportState(context.Background(), resource.ImportStateRequest{ID: ""}, resp)

	assert.True(t, resp.Diagnostics.HasError())
}
