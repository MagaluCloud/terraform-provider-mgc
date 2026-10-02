package tags

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	tagSDK "github.com/MagaluCloud/mgc-sdk-go/tag"
	"github.com/MagaluCloud/terraform-provider-mgc/mgc/internal/mocks"
	"github.com/MagaluCloud/terraform-provider-mgc/mgc/internal/testutils"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func tagValueFromJSON(t *testing.T, raw string) tagSDK.TagValue {
	t.Helper()

	var value tagSDK.TagValue
	require.NoError(t, json.Unmarshal([]byte(raw), &value))
	return value
}

func newTestTagValueResource(t *testing.T, svc tagSDK.TagValueService) (*tagValueResource, schema.Schema) {
	t.Helper()

	r := &tagValueResource{values: svc}
	return r, testutils.GetResourceTestSchema(t, r).Schema
}

// The API name pattern accepts spaces and brackets but never a comma, which is
// what makes the comma safe as the separator of the composite id.
func TestParseTagValueID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		id          string
		wantTag     string
		wantValue   string
		wantFailure bool
	}{
		{
			name:      "tag and value",
			id:        "ambiente,producao",
			wantTag:   "ambiente",
			wantValue: "producao",
		},
		{
			name:      "names with spaces and brackets",
			id:        "centro de custo,time [plataforma]",
			wantTag:   "centro de custo",
			wantValue: "time [plataforma]",
		},
		{
			name:        "missing the value",
			id:          "ambiente",
			wantFailure: true,
		},
		{
			name:        "too many parts",
			id:          "ambiente,producao,extra",
			wantFailure: true,
		},
		{
			name:        "empty",
			id:          "",
			wantFailure: true,
		},
		{
			name:        "empty tag name",
			id:          ",producao",
			wantFailure: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tagName, valueName, err := parseTagValueID(tt.id)

			if tt.wantFailure {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.wantTag, tagName)
			assert.Equal(t, tt.wantValue, valueName)
			assert.Equal(t, tt.id, tagValueID(tagName, valueName), "the id has to round trip")
		})
	}
}

func TestFlattenTagValue(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		prior    tagValueResourceModel
		response string
		expected tagValueResourceModel
	}{
		{
			name: "api echoes the config",
			prior: tagValueResourceModel{
				TagName:     types.StringValue("ambiente"),
				Name:        types.StringValue("producao"),
				Description: types.StringValue("ambiente produtivo"),
			},
			response: `{
				"name": "producao",
				"description": "ambiente produtivo",
				"created_at": "2026-08-03T00:58:38.215088",
				"updated_at": null
			}`,
			expected: tagValueResourceModel{
				ID:          types.StringValue("ambiente,producao"),
				TagName:     types.StringValue("ambiente"),
				Name:        types.StringValue("producao"),
				Description: types.StringValue("ambiente produtivo"),
				CreatedAt:   types.StringValue("2026-08-03T00:58:38Z"),
				UpdatedAt:   types.StringNull(),
			},
		},
		{
			name: "empty description from the api keeps the config null",
			prior: tagValueResourceModel{
				TagName: types.StringValue("ambiente"),
				Name:    types.StringValue("producao"),
			},
			response: `{
				"name": "producao",
				"description": "",
				"created_at": "2026-08-03T00:58:38.215088",
				"updated_at": "2026-08-03T01:14:56.834488"
			}`,
			expected: tagValueResourceModel{
				ID:        types.StringValue("ambiente,producao"),
				TagName:   types.StringValue("ambiente"),
				Name:      types.StringValue("producao"),
				CreatedAt: types.StringValue("2026-08-03T00:58:38Z"),
				UpdatedAt: types.StringValue("2026-08-03T01:14:56Z"),
			},
		},
		{
			name: "description changed out of band",
			prior: tagValueResourceModel{
				TagName:     types.StringValue("ambiente"),
				Name:        types.StringValue("producao"),
				Description: types.StringValue("antigo"),
			},
			response: `{
				"name": "producao",
				"description": "outro",
				"created_at": "2026-08-03T00:58:38.215088",
				"updated_at": "2026-08-03T01:14:56.834488"
			}`,
			expected: tagValueResourceModel{
				ID:          types.StringValue("ambiente,producao"),
				TagName:     types.StringValue("ambiente"),
				Name:        types.StringValue("producao"),
				Description: types.StringValue("outro"),
				CreatedAt:   types.StringValue("2026-08-03T00:58:38Z"),
				UpdatedAt:   types.StringValue("2026-08-03T01:14:56Z"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := flattenTagValue(tt.prior, tagValueFromJSON(t, tt.response))

			assert.Equal(t, tt.expected, got)
		})
	}
}

// Description always travels, and "" is what clears it. The name only travels
// when it changed, which renames the value.
func TestBuildUpdateTagValueRequest(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		state    tagValueResourceModel
		plan     tagValueResourceModel
		expected tagSDK.UpdateTagValueRequest
	}{
		{
			name:     "name changed renames the value",
			state:    tagValueResourceModel{Name: types.StringValue("producao")},
			plan:     tagValueResourceModel{Name: types.StringValue("prod"), Description: types.StringValue("x")},
			expected: tagSDK.UpdateTagValueRequest{Name: ptr("prod"), Description: ptr("x")},
		},
		{
			name:     "description set",
			plan:     tagValueResourceModel{Description: types.StringValue("ambiente produtivo")},
			expected: tagSDK.UpdateTagValueRequest{Description: ptr("ambiente produtivo")},
		},
		{
			name:     "description removed from the config is cleared",
			plan:     tagValueResourceModel{Description: types.StringNull()},
			expected: tagSDK.UpdateTagValueRequest{Description: ptr("")},
		},
		{
			name:     "explicit empty description",
			plan:     tagValueResourceModel{Description: types.StringValue("")},
			expected: tagSDK.UpdateTagValueRequest{Description: ptr("")},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.expected, buildUpdateTagValueRequest(tt.state, tt.plan))
		})
	}
}

func TestTagValueResourceMetadata(t *testing.T) {
	t.Parallel()

	resp := &resource.MetadataResponse{}
	NewTagValueResource().Metadata(context.Background(), resource.MetadataRequest{ProviderTypeName: "mgc"}, resp)

	assert.Equal(t, "mgc_tag_value", resp.TypeName)
}

func TestTagValueResourceSchema(t *testing.T) {
	t.Parallel()

	_, valueSchema := newTestTagValueResource(t, nil)

	for _, attribute := range []string{"id", "tag_name", "name", "description", "created_at", "updated_at"} {
		assert.Contains(t, valueSchema.Attributes, attribute)
	}

	// Renaming the tag or the value happens in place, so neither forces a
	// replacement, and the id follows both names.
	for _, attribute := range []string{"tag_name", "name"} {
		field := valueSchema.Attributes[attribute].(schema.StringAttribute)
		assert.True(t, field.Required, attribute)
		assert.Empty(t, field.PlanModifiers, attribute)
	}

	id := valueSchema.Attributes["id"].(schema.StringAttribute)
	require.Len(t, id.PlanModifiers, 1)
	assert.Equal(t, []string{"tag_name", "name"}, id.PlanModifiers[0].(idFromAttributes).attributes)

	updatedAt := valueSchema.Attributes["updated_at"].(schema.StringAttribute)
	assert.Empty(t, updatedAt.PlanModifiers)
}

func TestTagValueResourceCreate(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	created := tagValueFromJSON(t, `{
		"name": "producao",
		"description": "ambiente produtivo",
		"created_at": "2026-08-03T00:58:38.215088",
		"updated_at": null
	}`)

	mockSvc := new(mocks.TagValueService)
	mockSvc.On("Create", ctx, "ambiente", tagSDK.CreateTagValueRequest{
		Name:        "producao",
		Description: ptr("ambiente produtivo"),
	}).Return(&created, nil)

	r, valueSchema := newTestTagValueResource(t, mockSvc)

	plan := tfsdk.Plan{Schema: valueSchema}
	require.False(t, plan.Set(ctx, &tagValueResourceModel{
		ID:          types.StringUnknown(),
		TagName:     types.StringValue("ambiente"),
		Name:        types.StringValue("producao"),
		Description: types.StringValue("ambiente produtivo"),
		CreatedAt:   types.StringUnknown(),
		UpdatedAt:   types.StringUnknown(),
	}).HasError())

	resp := &resource.CreateResponse{State: tfsdk.State{Schema: valueSchema}}
	r.Create(ctx, resource.CreateRequest{Plan: plan}, resp)

	require.False(t, resp.Diagnostics.HasError(), "Create returned errors: %v", resp.Diagnostics)

	var state tagValueResourceModel
	resp.State.Get(ctx, &state)

	assert.Equal(t, "ambiente,producao", state.ID.ValueString())
	assert.Equal(t, "2026-08-03T00:58:38Z", state.CreatedAt.ValueString())
	mockSvc.AssertExpectations(t)
}

// Deleting the parent tag deletes its values, and the API reports it with its
// own message. The value must leave the state instead of blocking the refresh.
func TestTagValueResourceReadNotFound(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  error
	}{
		{
			name: "value deleted",
			err:  httpError(http.StatusNotFound, `{"message":"Not Found","detail":"tag value not found"}`),
		},
		{
			name: "parent tag deleted",
			err:  httpError(http.StatusNotFound, `{"message":"Not Found","detail":"Tag not found"}`),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := context.Background()

			mockSvc := new(mocks.TagValueService)
			mockSvc.On("Get", ctx, "ambiente", "producao").Return(nil, tt.err)

			r, valueSchema := newTestTagValueResource(t, mockSvc)

			priorState := tfsdk.State{Schema: valueSchema}
			require.False(t, priorState.Set(ctx, &tagValueResourceModel{
				ID:          types.StringValue("ambiente,producao"),
				TagName:     types.StringValue("ambiente"),
				Name:        types.StringValue("producao"),
				Description: types.StringNull(),
				CreatedAt:   types.StringValue("2026-08-03T00:58:38Z"),
				UpdatedAt:   types.StringNull(),
			}).HasError())

			resp := &resource.ReadResponse{State: tfsdk.State{Schema: valueSchema}}
			r.Read(ctx, resource.ReadRequest{State: priorState}, resp)

			assert.False(t, resp.Diagnostics.HasError(), "a missing value is not an error: %v", resp.Diagnostics)
			assert.True(t, resp.State.Raw.IsNull(), "the value must be dropped from the state")
			mockSvc.AssertExpectations(t)
		})
	}
}

func TestTagValueResourceUpdate(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	updated := tagValueFromJSON(t, `{
		"name": "producao",
		"description": "",
		"created_at": "2026-08-03T00:58:38.215088",
		"updated_at": "2026-08-03T01:14:56.834488"
	}`)

	mockSvc := new(mocks.TagValueService)
	mockSvc.On("Update", ctx, "ambiente", "producao", tagSDK.UpdateTagValueRequest{Description: ptr("")}).
		Return(&updated, nil)

	r, valueSchema := newTestTagValueResource(t, mockSvc)

	priorState := tfsdk.State{Schema: valueSchema}
	require.False(t, priorState.Set(ctx, &tagValueResourceModel{
		ID:          types.StringValue("ambiente,producao"),
		TagName:     types.StringValue("ambiente"),
		Name:        types.StringValue("producao"),
		Description: types.StringValue("antigo"),
		CreatedAt:   types.StringValue("2026-08-03T00:58:38Z"),
		UpdatedAt:   types.StringNull(),
	}).HasError())

	plan := tfsdk.Plan{Schema: valueSchema}
	require.False(t, plan.Set(ctx, &tagValueResourceModel{
		ID:          types.StringValue("ambiente,producao"),
		TagName:     types.StringValue("ambiente"),
		Name:        types.StringValue("producao"),
		Description: types.StringNull(),
		CreatedAt:   types.StringValue("2026-08-03T00:58:38Z"),
		UpdatedAt:   types.StringUnknown(),
	}).HasError())

	resp := &resource.UpdateResponse{State: tfsdk.State{Schema: valueSchema}}
	r.Update(ctx, resource.UpdateRequest{Plan: plan, State: priorState}, resp)

	require.False(t, resp.Diagnostics.HasError(), "Update returned errors: %v", resp.Diagnostics)

	var state tagValueResourceModel
	resp.State.Get(ctx, &state)

	assert.True(t, state.Description.IsNull(), "an emptied description stays null in the state")
	assert.Equal(t, "2026-08-03T01:14:56Z", state.UpdatedAt.ValueString())
	mockSvc.AssertExpectations(t)
}

// valueUpdateRequest builds the Update request of a value moving from one pair of
// names to another, with the description kept.
func valueUpdateRequest(t *testing.T, valueSchema schema.Schema, from, to [2]string) resource.UpdateRequest {
	t.Helper()
	ctx := context.Background()

	model := func(names [2]string) *tagValueResourceModel {
		return &tagValueResourceModel{
			ID:          types.StringValue(tagValueID(names[0], names[1])),
			TagName:     types.StringValue(names[0]),
			Name:        types.StringValue(names[1]),
			Description: types.StringValue("mesma"),
			CreatedAt:   types.StringValue("2026-08-03T00:58:38Z"),
			UpdatedAt:   types.StringNull(),
		}
	}

	state := tfsdk.State{Schema: valueSchema}
	require.False(t, state.Set(ctx, model(from)).HasError())
	plan := tfsdk.Plan{Schema: valueSchema}
	require.False(t, plan.Set(ctx, model(to)).HasError())

	return resource.UpdateRequest{Plan: plan, State: state}
}

func TestTagValueResourceUpdateRename(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	renamed := tagValueFromJSON(t, `{
		"name": "prod",
		"description": "mesma",
		"created_at": "2026-08-03T00:58:38.215088",
		"updated_at": "2026-08-03T01:14:56.834488"
	}`)

	mockSvc := new(mocks.TagValueService)
	mockSvc.On("Update", ctx, "ambiente", "producao", tagSDK.UpdateTagValueRequest{Name: ptr("prod"), Description: ptr("mesma")}).
		Return(&renamed, nil)

	r, valueSchema := newTestTagValueResource(t, mockSvc)
	resp := &resource.UpdateResponse{State: tfsdk.State{Schema: valueSchema}}
	r.Update(ctx, valueUpdateRequest(t, valueSchema, [2]string{"ambiente", "producao"}, [2]string{"ambiente", "prod"}), resp)

	require.False(t, resp.Diagnostics.HasError(), "Update returned errors: %v", resp.Diagnostics)

	var state tagValueResourceModel
	resp.State.Get(ctx, &state)
	assert.Equal(t, "ambiente,prod", state.ID.ValueString())
	mockSvc.AssertExpectations(t)
}

// Renaming the tag takes its values along: the value is found under the new tag
// name and is only updated there, never recreated.
func TestTagValueResourceUpdateFollowsTagRename(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	value := tagValueFromJSON(t, `{
		"name": "producao",
		"description": "mesma",
		"created_at": "2026-08-03T00:58:38.215088",
		"updated_at": null
	}`)

	mockSvc := new(mocks.TagValueService)
	mockSvc.On("Get", ctx, "env", "producao").Return(&value, nil)
	mockSvc.On("Update", ctx, "env", "producao", tagSDK.UpdateTagValueRequest{Description: ptr("mesma")}).
		Return(&value, nil)

	r, valueSchema := newTestTagValueResource(t, mockSvc)
	resp := &resource.UpdateResponse{State: tfsdk.State{Schema: valueSchema}}
	r.Update(ctx, valueUpdateRequest(t, valueSchema, [2]string{"ambiente", "producao"}, [2]string{"env", "producao"}), resp)

	require.False(t, resp.Diagnostics.HasError(), "Update returned errors: %v", resp.Diagnostics)

	var state tagValueResourceModel
	resp.State.Get(ctx, &state)
	assert.Equal(t, "env,producao", state.ID.ValueString())
	assert.Equal(t, "env", state.TagName.ValueString())
	mockSvc.AssertNotCalled(t, "Create", mock.Anything, mock.Anything, mock.Anything)
	mockSvc.AssertNotCalled(t, "Delete", mock.Anything, mock.Anything, mock.Anything)
	mockSvc.AssertExpectations(t)
}

// Pointing the value to a tag that does not have it moves the value: created in
// the new tag, deleted from the old one.
func TestTagValueResourceUpdateMovesToAnotherTag(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	created := tagValueFromJSON(t, `{
		"name": "producao",
		"description": "mesma",
		"created_at": "2026-08-03T02:00:00.000000",
		"updated_at": null
	}`)

	mockSvc := new(mocks.TagValueService)
	mockSvc.On("Get", ctx, "time", "producao").Return(nil, httpError(http.StatusNotFound, ""))
	mockSvc.On("Create", ctx, "time", tagSDK.CreateTagValueRequest{Name: "producao", Description: ptr("mesma")}).
		Return(&created, nil)
	mockSvc.On("Delete", ctx, "ambiente", "producao").Return(nil)

	r, valueSchema := newTestTagValueResource(t, mockSvc)
	resp := &resource.UpdateResponse{State: tfsdk.State{Schema: valueSchema}}
	r.Update(ctx, valueUpdateRequest(t, valueSchema, [2]string{"ambiente", "producao"}, [2]string{"time", "producao"}), resp)

	require.False(t, resp.Diagnostics.HasError(), "Update returned errors: %v", resp.Diagnostics)

	var state tagValueResourceModel
	resp.State.Get(ctx, &state)
	assert.Equal(t, "time,producao", state.ID.ValueString())
	assert.Equal(t, "2026-08-03T02:00:00Z", state.CreatedAt.ValueString(), "the moved value is a new one")
	mockSvc.AssertExpectations(t)
}

// A failed delete of the old value still records the new one, which exists.
func TestTagValueResourceUpdateMoveDeleteFails(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	created := tagValueFromJSON(t, `{
		"name": "producao",
		"description": "mesma",
		"created_at": "2026-08-03T02:00:00.000000",
		"updated_at": null
	}`)

	mockSvc := new(mocks.TagValueService)
	mockSvc.On("Get", ctx, "time", "producao").Return(nil, httpError(http.StatusNotFound, ""))
	mockSvc.On("Create", ctx, "time", mock.Anything).Return(&created, nil)
	mockSvc.On("Delete", ctx, "ambiente", "producao").Return(httpError(http.StatusInternalServerError, ""))

	r, valueSchema := newTestTagValueResource(t, mockSvc)
	resp := &resource.UpdateResponse{State: tfsdk.State{Schema: valueSchema}}
	r.Update(ctx, valueUpdateRequest(t, valueSchema, [2]string{"ambiente", "producao"}, [2]string{"time", "producao"}), resp)

	assert.True(t, resp.Diagnostics.HasError(), "the failed delete has to reach the user")

	var state tagValueResourceModel
	resp.State.Get(ctx, &state)
	assert.Equal(t, "time,producao", state.ID.ValueString())
	mockSvc.AssertExpectations(t)
}

func TestTagValueResourceDelete(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		err       error
		wantError bool
	}{
		{name: "deleted", err: nil},
		{name: "already gone", err: httpError(http.StatusNotFound, "")},
		{name: "server error is propagated", err: httpError(http.StatusInternalServerError, ""), wantError: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := context.Background()

			mockSvc := new(mocks.TagValueService)
			mockSvc.On("Delete", ctx, "ambiente", "producao").Return(tt.err)

			r, valueSchema := newTestTagValueResource(t, mockSvc)

			priorState := tfsdk.State{Schema: valueSchema}
			require.False(t, priorState.Set(ctx, &tagValueResourceModel{
				ID:          types.StringValue("ambiente,producao"),
				TagName:     types.StringValue("ambiente"),
				Name:        types.StringValue("producao"),
				Description: types.StringNull(),
				CreatedAt:   types.StringValue("2026-08-03T00:58:38Z"),
				UpdatedAt:   types.StringNull(),
			}).HasError())

			resp := &resource.DeleteResponse{State: tfsdk.State{Schema: valueSchema}}
			r.Delete(ctx, resource.DeleteRequest{State: priorState}, resp)

			assert.Equal(t, tt.wantError, resp.Diagnostics.HasError(), "%v", resp.Diagnostics)
			mockSvc.AssertExpectations(t)
		})
	}
}

func TestTagValueResourceImportState(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	r, valueSchema := newTestTagValueResource(t, nil)

	resp := &resource.ImportStateResponse{State: emptyState(ctx, valueSchema)}
	r.ImportState(ctx, resource.ImportStateRequest{ID: "ambiente,producao"}, resp)

	require.False(t, resp.Diagnostics.HasError(), "ImportState returned errors: %v", resp.Diagnostics)

	var tagName, name types.String
	resp.State.GetAttribute(ctx, path.Root("tag_name"), &tagName)
	resp.State.GetAttribute(ctx, path.Root("name"), &name)

	assert.Equal(t, "ambiente", tagName.ValueString())
	assert.Equal(t, "producao", name.ValueString())
}

func TestTagValueResourceImportStateInvalid(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	r, valueSchema := newTestTagValueResource(t, nil)

	resp := &resource.ImportStateResponse{State: emptyState(ctx, valueSchema)}
	r.ImportState(ctx, resource.ImportStateRequest{ID: "ambiente"}, resp)

	assert.True(t, resp.Diagnostics.HasError())
}
