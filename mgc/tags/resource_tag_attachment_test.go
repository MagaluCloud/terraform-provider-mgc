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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

const testExternalID = "d4138294-2cc6-4c39-afcc-54ebadf0f9db"

func resourceFromJSON(t *testing.T, raw string) tagSDK.Resource {
	t.Helper()

	var taggedResource tagSDK.Resource
	require.NoError(t, json.Unmarshal([]byte(raw), &taggedResource))
	return taggedResource
}

func tagsMap(pairs map[string]string) types.Map {
	elements := make(map[string]attr.Value, len(pairs))
	for name, value := range pairs {
		elements[name] = types.StringValue(value)
	}
	return types.MapValueMust(types.StringType, elements)
}

func newTestAttachmentResource(t *testing.T, svc tagSDK.ResourceService) (*tagAttachmentResource, schema.Schema) {
	t.Helper()

	r := &tagAttachmentResource{resources: svc}
	return r, testutils.GetResourceTestSchema(t, r).Schema
}

// diffTags drives the whole resource: the API has no way to replace the set of
// tags in one call, and no way to overwrite the value of an attached tag.
func TestDiffTags(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		state      map[string]string
		plan       map[string]string
		wantAttach map[string]string
		wantDetach []string
	}{
		{
			name:       "nothing changed",
			state:      map[string]string{"ambiente": "producao"},
			plan:       map[string]string{"ambiente": "producao"},
			wantAttach: map[string]string{},
			wantDetach: nil,
		},
		{
			name:       "tag added",
			state:      map[string]string{"ambiente": "producao"},
			plan:       map[string]string{"ambiente": "producao", "time": "plataforma"},
			wantAttach: map[string]string{"time": "plataforma"},
			wantDetach: nil,
		},
		{
			name:       "tag removed from the config",
			state:      map[string]string{"ambiente": "producao", "time": "plataforma"},
			plan:       map[string]string{"ambiente": "producao"},
			wantAttach: map[string]string{},
			wantDetach: []string{"time"},
		},
		{
			// The API answers 409 when the tag is already attached, so changing a
			// value means detaching first.
			name:       "value changed",
			state:      map[string]string{"ambiente": "producao"},
			plan:       map[string]string{"ambiente": "homolog"},
			wantAttach: map[string]string{"ambiente": "homolog"},
			wantDetach: []string{"ambiente"},
		},
		{
			// This resource is authoritative: a tag attached by hand is removed.
			name:       "tag attached out of band",
			state:      map[string]string{"ambiente": "producao", "custo": "cc-42"},
			plan:       map[string]string{"ambiente": "producao"},
			wantAttach: map[string]string{},
			wantDetach: []string{"custo"},
		},
		{
			name:       "first attachment",
			state:      map[string]string{},
			plan:       map[string]string{"ambiente": "producao"},
			wantAttach: map[string]string{"ambiente": "producao"},
			wantDetach: nil,
		},
		{
			name:       "added, changed and removed at once",
			state:      map[string]string{"ambiente": "producao", "time": "plataforma", "custo": "cc-42"},
			plan:       map[string]string{"ambiente": "homolog", "time": "plataforma", "so": "linux"},
			wantAttach: map[string]string{"ambiente": "homolog", "so": "linux"},
			wantDetach: []string{"ambiente", "custo"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			attach, detach := diffTags(tt.state, tt.plan)

			assert.Equal(t, tt.wantAttach, attach)
			assert.Equal(t, tt.wantDetach, detach, "detach has to be sorted to keep the apply deterministic")
		})
	}
}

func TestFlattenAttachment(t *testing.T) {
	t.Parallel()

	// Real payload: the API answers the attach with the whole resource.
	response := `{
		"created_at": "2026-08-03T01:19:31.655927",
		"external_id": "` + testExternalID + `",
		"id": "f6ddcf25-e4a9-4938-893f-da4d5975b8fc",
		"last_tag_associated_at": "2026-08-03T01:22:35.323684",
		"region": "br-se1",
		"resource_type": {"name": "k8s.cluster", "product": "kubernetes"},
		"tags": [
			{"name": "ambiente", "value": "producao"},
			{"name": "custo", "value": "cc-42"}
		],
		"updated_at": "2026-08-03T01:22:35.280244"
	}`

	prior := tagAttachmentResourceModel{
		ResourceID: types.StringValue(testExternalID),
		Tags:       tagsMap(map[string]string{"ambiente": "producao"}),
	}

	taggedResource := resourceFromJSON(t, response)
	got := flattenAttachment(prior, taggedResource)

	assert.Equal(t, testExternalID, got.ID.ValueString())
	assert.Equal(t, "k8s.cluster", got.ResourceType.ValueString())
	assert.Equal(t, "br-se1", got.Region.ValueString())

	// Tags are left to the caller: after an apply the state must hold what the
	// plan declared, or Terraform rejects the result as inconsistent.
	assert.Equal(t, prior.Tags, got.Tags)

	// Authoritative refresh: every tag the API reports lands in the state, so
	// the one attached out of band shows up in the next plan as a removal.
	assert.Equal(t, tagsMap(map[string]string{"ambiente": "producao", "custo": "cc-42"}), mirrorTags(taggedResource))
}

func TestAttachmentResourceMetadata(t *testing.T) {
	t.Parallel()

	resp := &resource.MetadataResponse{}
	NewTagAttachmentResource().Metadata(context.Background(), resource.MetadataRequest{ProviderTypeName: "mgc"}, resp)

	assert.Equal(t, "mgc_tag_attachment", resp.TypeName)
}

func TestAttachmentResourceSchema(t *testing.T) {
	t.Parallel()

	_, attachmentSchema := newTestAttachmentResource(t, nil)

	for _, attribute := range []string{"id", "resource_id", "tags", "resource_type", "region"} {
		assert.Contains(t, attachmentSchema.Attributes, attribute)
	}

	// It changes whenever any tag of the resource moves, including tags this
	// resource does not manage, so it would be permanent drift.
	assert.NotContains(t, attachmentSchema.Attributes, "last_tag_associated_at")

	resourceID := attachmentSchema.Attributes["resource_id"].(schema.StringAttribute)
	assert.True(t, resourceID.Required)
	assert.Len(t, resourceID.PlanModifiers, 1, "the attachment cannot move to another resource")

	// An empty map would attach nothing and leave a resource that the API
	// reports as missing, so the plan has to fail instead.
	tags := attachmentSchema.Attributes["tags"].(schema.MapAttribute)
	assert.True(t, tags.Required)
	assert.NotEmpty(t, tags.Validators)
}

func TestAttachmentResourceCreate(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	attached := resourceFromJSON(t, `{
		"created_at": "2026-08-03T01:19:31.655927",
		"external_id": "`+testExternalID+`",
		"id": "f6ddcf25-e4a9-4938-893f-da4d5975b8fc",
		"last_tag_associated_at": "2026-08-03T01:22:35.323684",
		"region": "br-se1",
		"resource_type": {"name": "k8s.cluster", "product": "kubernetes"},
		"tags": [
			{"name": "ambiente", "value": "producao"},
			{"name": "time", "value": "plataforma"}
		],
		"updated_at": "2026-08-03T01:22:35.280244"
	}`)

	mockSvc := new(mocks.ResourceService)
	// One call carries every pair, and the order is sorted so the request is
	// reproducible.
	mockSvc.On("AttachTags", ctx, testExternalID, tagSDK.AttachTagsRequest{
		Tags: []tagSDK.AttachTag{
			{Name: "ambiente", Value: "producao"},
			{Name: "time", Value: "plataforma"},
		},
	}).Return(&attached, nil)

	r, attachmentSchema := newTestAttachmentResource(t, mockSvc)

	plan := tfsdk.Plan{Schema: attachmentSchema}
	require.False(t, plan.Set(ctx, &tagAttachmentResourceModel{
		ID:           types.StringUnknown(),
		ResourceID:   types.StringValue(testExternalID),
		Tags:         tagsMap(map[string]string{"ambiente": "producao", "time": "plataforma"}),
		ResourceType: types.StringUnknown(),
		Region:       types.StringUnknown(),
	}).HasError())

	resp := &resource.CreateResponse{State: tfsdk.State{Schema: attachmentSchema}}
	r.Create(ctx, resource.CreateRequest{Plan: plan}, resp)

	require.False(t, resp.Diagnostics.HasError(), "Create returned errors: %v", resp.Diagnostics)

	var state tagAttachmentResourceModel
	resp.State.Get(ctx, &state)

	assert.Equal(t, testExternalID, state.ID.ValueString())
	assert.Equal(t, "k8s.cluster", state.ResourceType.ValueString())
	mockSvc.AssertExpectations(t)
}

func TestAttachmentResourceCreateConflict(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	mockSvc := new(mocks.ResourceService)
	mockSvc.On("AttachTags", ctx, testExternalID, mock.Anything).
		Return(nil, httpError(http.StatusConflict, `{"message":"Conflict","detail":"Tag at index 0 is already attached to the resource"}`))

	r, attachmentSchema := newTestAttachmentResource(t, mockSvc)

	plan := tfsdk.Plan{Schema: attachmentSchema}
	require.False(t, plan.Set(ctx, &tagAttachmentResourceModel{
		ID:           types.StringUnknown(),
		ResourceID:   types.StringValue(testExternalID),
		Tags:         tagsMap(map[string]string{"ambiente": "producao"}),
		ResourceType: types.StringUnknown(),
		Region:       types.StringUnknown(),
	}).HasError())

	resp := &resource.CreateResponse{State: tfsdk.State{Schema: attachmentSchema}}
	r.Create(ctx, resource.CreateRequest{Plan: plan}, resp)

	require.True(t, resp.Diagnostics.HasError())
	assert.Contains(t, resp.Diagnostics.Errors()[0].Detail(), "terraform import")
	mockSvc.AssertExpectations(t)
}

func TestAttachmentResourceReadNotFound(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  error
	}{
		{
			name: "404",
			err:  httpError(http.StatusNotFound, ""),
		},
		{
			// What the API answers today when the resource carries no tag.
			name: "400",
			err:  httpError(http.StatusBadRequest, `{"message":"Bad Request","detail":"Tag was not found"}`),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := context.Background()

			mockSvc := new(mocks.ResourceService)
			mockSvc.On("Get", ctx, testExternalID).Return(nil, tt.err)

			r, attachmentSchema := newTestAttachmentResource(t, mockSvc)

			priorState := tfsdk.State{Schema: attachmentSchema}
			require.False(t, priorState.Set(ctx, &tagAttachmentResourceModel{
				ID:           types.StringValue(testExternalID),
				ResourceID:   types.StringValue(testExternalID),
				Tags:         tagsMap(map[string]string{"ambiente": "producao"}),
				ResourceType: types.StringValue("k8s.cluster"),
				Region:       types.StringValue("br-se1"),
			}).HasError())

			resp := &resource.ReadResponse{State: tfsdk.State{Schema: attachmentSchema}}
			r.Read(ctx, resource.ReadRequest{State: priorState}, resp)

			assert.False(t, resp.Diagnostics.HasError(), "detached out of band is not an error: %v", resp.Diagnostics)
			assert.True(t, resp.State.Raw.IsNull(), "the attachment must be dropped from the state")
			mockSvc.AssertExpectations(t)
		})
	}
}

func TestAttachmentResourceUpdate(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	final := resourceFromJSON(t, `{
		"created_at": "2026-08-03T01:19:31.655927",
		"external_id": "`+testExternalID+`",
		"id": "f6ddcf25-e4a9-4938-893f-da4d5975b8fc",
		"last_tag_associated_at": "2026-08-03T01:22:35.323684",
		"region": "br-se1",
		"resource_type": {"name": "k8s.cluster", "product": "kubernetes"},
		"tags": [
			{"name": "ambiente", "value": "homolog"},
			{"name": "so", "value": "linux"}
		],
		"updated_at": "2026-08-03T01:22:35.280244"
	}`)

	mockSvc := new(mocks.ResourceService)
	// A changed value is a detach followed by an attach, and the removed tag is
	// detached as well.
	mockSvc.On("DetachTag", ctx, testExternalID, "ambiente").Return(nil)
	mockSvc.On("DetachTag", ctx, testExternalID, "time").Return(nil)
	mockSvc.On("AttachTags", ctx, testExternalID, tagSDK.AttachTagsRequest{
		Tags: []tagSDK.AttachTag{
			{Name: "ambiente", Value: "homolog"},
			{Name: "so", Value: "linux"},
		},
	}).Return(&final, nil)

	r, attachmentSchema := newTestAttachmentResource(t, mockSvc)

	priorState := tfsdk.State{Schema: attachmentSchema}
	require.False(t, priorState.Set(ctx, &tagAttachmentResourceModel{
		ID:           types.StringValue(testExternalID),
		ResourceID:   types.StringValue(testExternalID),
		Tags:         tagsMap(map[string]string{"ambiente": "producao", "time": "plataforma"}),
		ResourceType: types.StringValue("k8s.cluster"),
		Region:       types.StringValue("br-se1"),
	}).HasError())

	plan := tfsdk.Plan{Schema: attachmentSchema}
	require.False(t, plan.Set(ctx, &tagAttachmentResourceModel{
		ID:           types.StringValue(testExternalID),
		ResourceID:   types.StringValue(testExternalID),
		Tags:         tagsMap(map[string]string{"ambiente": "homolog", "so": "linux"}),
		ResourceType: types.StringValue("k8s.cluster"),
		Region:       types.StringValue("br-se1"),
	}).HasError())

	resp := &resource.UpdateResponse{State: tfsdk.State{Schema: attachmentSchema}}
	r.Update(ctx, resource.UpdateRequest{Plan: plan, State: priorState}, resp)

	require.False(t, resp.Diagnostics.HasError(), "Update returned errors: %v", resp.Diagnostics)

	var state tagAttachmentResourceModel
	resp.State.Get(ctx, &state)

	assert.Equal(t, tagsMap(map[string]string{"ambiente": "homolog", "so": "linux"}), state.Tags)
	mockSvc.AssertExpectations(t)
}

// A failed attach after a successful detach must not leave the state lying about
// what the resource carries.
func TestAttachmentResourceUpdatePartialFailure(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	afterFailure := resourceFromJSON(t, `{
		"created_at": "2026-08-03T01:19:31.655927",
		"external_id": "`+testExternalID+`",
		"id": "f6ddcf25-e4a9-4938-893f-da4d5975b8fc",
		"last_tag_associated_at": "2026-08-03T01:22:35.323684",
		"region": "br-se1",
		"resource_type": {"name": "k8s.cluster", "product": "kubernetes"},
		"tags": [],
		"updated_at": "2026-08-03T01:22:35.280244"
	}`)

	mockSvc := new(mocks.ResourceService)
	mockSvc.On("DetachTag", ctx, testExternalID, "ambiente").Return(nil)
	mockSvc.On("AttachTags", ctx, testExternalID, mock.Anything).
		Return(nil, httpError(http.StatusUnprocessableEntity, `{"detail":"value not found"}`))
	mockSvc.On("Get", ctx, testExternalID).Return(&afterFailure, nil)

	r, attachmentSchema := newTestAttachmentResource(t, mockSvc)

	priorState := tfsdk.State{Schema: attachmentSchema}
	require.False(t, priorState.Set(ctx, &tagAttachmentResourceModel{
		ID:           types.StringValue(testExternalID),
		ResourceID:   types.StringValue(testExternalID),
		Tags:         tagsMap(map[string]string{"ambiente": "producao"}),
		ResourceType: types.StringValue("k8s.cluster"),
		Region:       types.StringValue("br-se1"),
	}).HasError())

	plan := tfsdk.Plan{Schema: attachmentSchema}
	require.False(t, plan.Set(ctx, &tagAttachmentResourceModel{
		ID:           types.StringValue(testExternalID),
		ResourceID:   types.StringValue(testExternalID),
		Tags:         tagsMap(map[string]string{"ambiente": "inexistente"}),
		ResourceType: types.StringValue("k8s.cluster"),
		Region:       types.StringValue("br-se1"),
	}).HasError())

	resp := &resource.UpdateResponse{State: tfsdk.State{Schema: attachmentSchema}}
	r.Update(ctx, resource.UpdateRequest{Plan: plan, State: priorState}, resp)

	assert.True(t, resp.Diagnostics.HasError(), "the failure has to reach the user")

	var state tagAttachmentResourceModel
	resp.State.Get(ctx, &state)
	assert.Equal(t, tagsMap(map[string]string{}), state.Tags, "the state has to show what the resource really carries")
	mockSvc.AssertExpectations(t)
}

func TestAttachmentResourceDelete(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	mockSvc := new(mocks.ResourceService)
	mockSvc.On("DetachTag", ctx, testExternalID, "ambiente").Return(nil)
	// The owning product may have deleted the resource already, taking its tags
	// with it: that is a successful destroy.
	mockSvc.On("DetachTag", ctx, testExternalID, "time").
		Return(httpError(http.StatusBadRequest, `{"message":"Bad Request","detail":"Tag was not found"}`))

	r, attachmentSchema := newTestAttachmentResource(t, mockSvc)

	priorState := tfsdk.State{Schema: attachmentSchema}
	require.False(t, priorState.Set(ctx, &tagAttachmentResourceModel{
		ID:           types.StringValue(testExternalID),
		ResourceID:   types.StringValue(testExternalID),
		Tags:         tagsMap(map[string]string{"ambiente": "producao", "time": "plataforma"}),
		ResourceType: types.StringValue("k8s.cluster"),
		Region:       types.StringValue("br-se1"),
	}).HasError())

	resp := &resource.DeleteResponse{State: tfsdk.State{Schema: attachmentSchema}}
	r.Delete(ctx, resource.DeleteRequest{State: priorState}, resp)

	assert.False(t, resp.Diagnostics.HasError(), "detaching what is already gone is a success: %v", resp.Diagnostics)
	mockSvc.AssertExpectations(t)
}

func TestAttachmentResourceImportState(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	r, attachmentSchema := newTestAttachmentResource(t, nil)

	resp := &resource.ImportStateResponse{State: emptyState(ctx, attachmentSchema)}
	r.ImportState(ctx, resource.ImportStateRequest{ID: testExternalID}, resp)

	require.False(t, resp.Diagnostics.HasError(), "ImportState returned errors: %v", resp.Diagnostics)

	var resourceID types.String
	resp.State.GetAttribute(ctx, path.Root("resource_id"), &resourceID)
	assert.Equal(t, testExternalID, resourceID.ValueString())
}

// Removing a tag without adding any leaves no attach response to reuse, so the
// state comes from a read.
func TestAttachmentResourceUpdateDetachOnly(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	remaining := resourceFromJSON(t, `{
		"created_at": "2026-08-03T01:19:31.655927",
		"external_id": "`+testExternalID+`",
		"id": "f6ddcf25-e4a9-4938-893f-da4d5975b8fc",
		"last_tag_associated_at": "2026-08-03T01:22:35.323684",
		"region": "br-se1",
		"resource_type": {"name": "k8s.cluster", "product": "kubernetes"},
		"tags": [{"name": "ambiente", "value": "producao"}],
		"updated_at": "2026-08-03T01:22:35.280244"
	}`)

	mockSvc := new(mocks.ResourceService)
	mockSvc.On("DetachTag", ctx, testExternalID, "time").Return(nil)
	mockSvc.On("Get", ctx, testExternalID).Return(&remaining, nil)

	r, attachmentSchema := newTestAttachmentResource(t, mockSvc)

	priorState := tfsdk.State{Schema: attachmentSchema}
	require.False(t, priorState.Set(ctx, &tagAttachmentResourceModel{
		ID:           types.StringValue(testExternalID),
		ResourceID:   types.StringValue(testExternalID),
		Tags:         tagsMap(map[string]string{"ambiente": "producao", "time": "plataforma"}),
		ResourceType: types.StringValue("k8s.cluster"),
		Region:       types.StringValue("br-se1"),
	}).HasError())

	plan := tfsdk.Plan{Schema: attachmentSchema}
	require.False(t, plan.Set(ctx, &tagAttachmentResourceModel{
		ID:           types.StringValue(testExternalID),
		ResourceID:   types.StringValue(testExternalID),
		Tags:         tagsMap(map[string]string{"ambiente": "producao"}),
		ResourceType: types.StringValue("k8s.cluster"),
		Region:       types.StringValue("br-se1"),
	}).HasError())

	resp := &resource.UpdateResponse{State: tfsdk.State{Schema: attachmentSchema}}
	r.Update(ctx, resource.UpdateRequest{Plan: plan, State: priorState}, resp)

	require.False(t, resp.Diagnostics.HasError(), "Update returned errors: %v", resp.Diagnostics)

	var state tagAttachmentResourceModel
	resp.State.Get(ctx, &state)

	assert.Equal(t, tagsMap(map[string]string{"ambiente": "producao"}), state.Tags)
	mockSvc.AssertExpectations(t)
}

// The attach response carries every tag of the resource, including one attached
// by hand. Writing those into the state would fail the apply with "inconsistent
// result after apply", so the state keeps the planned tags and the extra one
// only surfaces on the next refresh.
func TestAttachmentResourceCreateKeepsPlannedTags(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	attached := resourceFromJSON(t, `{
		"created_at": "2026-08-03T01:19:31.655927",
		"external_id": "`+testExternalID+`",
		"id": "f6ddcf25-e4a9-4938-893f-da4d5975b8fc",
		"last_tag_associated_at": "2026-08-03T01:22:35.323684",
		"region": "br-se1",
		"resource_type": {"name": "k8s.cluster", "product": "kubernetes"},
		"tags": [
			{"name": "ambiente", "value": "producao"},
			{"name": "custo", "value": "cc-42"}
		],
		"updated_at": "2026-08-03T01:22:35.280244"
	}`)

	mockSvc := new(mocks.ResourceService)
	mockSvc.On("AttachTags", ctx, testExternalID, tagSDK.AttachTagsRequest{
		Tags: []tagSDK.AttachTag{{Name: "ambiente", Value: "producao"}},
	}).Return(&attached, nil)

	r, attachmentSchema := newTestAttachmentResource(t, mockSvc)

	plan := tfsdk.Plan{Schema: attachmentSchema}
	require.False(t, plan.Set(ctx, &tagAttachmentResourceModel{
		ID:           types.StringUnknown(),
		ResourceID:   types.StringValue(testExternalID),
		Tags:         tagsMap(map[string]string{"ambiente": "producao"}),
		ResourceType: types.StringUnknown(),
		Region:       types.StringUnknown(),
	}).HasError())

	resp := &resource.CreateResponse{State: tfsdk.State{Schema: attachmentSchema}}
	r.Create(ctx, resource.CreateRequest{Plan: plan}, resp)

	require.False(t, resp.Diagnostics.HasError(), "Create returned errors: %v", resp.Diagnostics)

	var state tagAttachmentResourceModel
	resp.State.Get(ctx, &state)

	assert.Equal(t, tagsMap(map[string]string{"ambiente": "producao"}), state.Tags)
	mockSvc.AssertExpectations(t)
}

// On refresh the same response has to land whole in the state: that is what
// turns the tag attached by hand into a removal in the next plan.
func TestAttachmentResourceReadIsAuthoritative(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	current := resourceFromJSON(t, `{
		"created_at": "2026-08-03T01:19:31.655927",
		"external_id": "`+testExternalID+`",
		"id": "f6ddcf25-e4a9-4938-893f-da4d5975b8fc",
		"last_tag_associated_at": "2026-08-03T01:22:35.323684",
		"region": "br-se1",
		"resource_type": {"name": "k8s.cluster", "product": "kubernetes"},
		"tags": [
			{"name": "ambiente", "value": "producao"},
			{"name": "custo", "value": "cc-42"}
		],
		"updated_at": "2026-08-03T01:22:35.280244"
	}`)

	mockSvc := new(mocks.ResourceService)
	mockSvc.On("Get", ctx, testExternalID).Return(&current, nil)

	r, attachmentSchema := newTestAttachmentResource(t, mockSvc)

	priorState := tfsdk.State{Schema: attachmentSchema}
	require.False(t, priorState.Set(ctx, &tagAttachmentResourceModel{
		ID:           types.StringValue(testExternalID),
		ResourceID:   types.StringValue(testExternalID),
		Tags:         tagsMap(map[string]string{"ambiente": "producao"}),
		ResourceType: types.StringValue("k8s.cluster"),
		Region:       types.StringValue("br-se1"),
	}).HasError())

	resp := &resource.ReadResponse{State: tfsdk.State{Schema: attachmentSchema}}
	r.Read(ctx, resource.ReadRequest{State: priorState}, resp)

	require.False(t, resp.Diagnostics.HasError(), "Read returned errors: %v", resp.Diagnostics)

	var state tagAttachmentResourceModel
	resp.State.Get(ctx, &state)

	assert.Equal(t, tagsMap(map[string]string{"ambiente": "producao", "custo": "cc-42"}), state.Tags)
	mockSvc.AssertExpectations(t)
}
