package tags

import (
	"context"
	"errors"
	"fmt"
	"testing"

	tagSDK "github.com/MagaluCloud/mgc-sdk-go/tag"
	"github.com/MagaluCloud/terraform-provider-mgc/mgc/internal/mocks"
	"github.com/MagaluCloud/terraform-provider-mgc/mgc/internal/testutils"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestAttachmentsDataSource(t *testing.T, service tagSDK.ResourceService) (*tagAttachmentsDataSource, schema.Schema) {
	t.Helper()

	dataSource := &tagAttachmentsDataSource{resources: service}
	return dataSource, testutils.GetDataSourceTestSchema(t, dataSource).Schema
}

func resourcePage(size int) []tagSDK.Resource {
	page := make([]tagSDK.Resource, size)
	for i := range page {
		page[i] = tagSDK.Resource{ExternalID: fmt.Sprintf("resource-%d", i)}
	}
	return page
}

func TestAttachmentsDataSourceMetadata(t *testing.T) {
	t.Parallel()

	resp := &datasource.MetadataResponse{}
	NewTagAttachmentsDataSource().Metadata(context.Background(), datasource.MetadataRequest{ProviderTypeName: "mgc"}, resp)

	assert.Equal(t, "mgc_tag_attachments", resp.TypeName)
}

func TestAttachmentsDataSourceRead(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	service := new(mocks.ResourceService)
	service.On("List", ctx, tagSDK.ListResourcesOptions{Limit: ptr(maxPageSize), Offset: ptr(0)}).
		Return([]tagSDK.Resource{
			resourceFromJSON(t, `{
				"external_id": "`+testExternalID+`",
				"resource_type": {"name": "net.vpc", "product": "network"},
				"region": "br-se1",
				"created_at": "2026-08-03T00:57:47.908658",
				"tags": [{"name": "ambiente", "value": "producao"}]
			}`),
			resourceFromJSON(t, `{
				"external_id": "0a1b2c3d-0000-4000-8000-000000000002",
				"resource_type": {"name": "k8s.cluster", "product": "kubernetes"},
				"region": "br-ne1",
				"created_at": "2026-08-03T01:00:00.000000",
				"tags": [{"name": "time", "value": "plataforma"}]
			}`),
		}, nil)

	dataSource, attachmentsSchema := newTestAttachmentsDataSource(t, service)
	resp := &datasource.ReadResponse{State: tfsdk.State{Schema: attachmentsSchema}}

	dataSource.Read(ctx, datasource.ReadRequest{
		Config: newDataSourceConfig(t, attachmentsSchema, nil),
	}, resp)

	require.False(t, resp.Diagnostics.HasError(), "%v", resp.Diagnostics)

	var state tagAttachmentsModel
	require.False(t, resp.State.Get(ctx, &state).HasError())

	require.Len(t, state.Attachments, 2)
	assert.Equal(t, testExternalID, state.Attachments[0].ResourceID.ValueString())
	assert.Equal(t, tagsMap(map[string]string{"ambiente": "producao"}), state.Attachments[0].Tags)
	assert.Equal(t, "net.vpc", state.Attachments[0].ResourceType.ValueString())
	assert.Equal(t, "k8s.cluster", state.Attachments[1].ResourceType.ValueString())
	assert.Equal(t, "kubernetes", state.Attachments[1].Product.ValueString())
	service.AssertExpectations(t)
}

func TestAttachmentsDataSourceReadPagesThroughEveryResource(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	service := new(mocks.ResourceService)
	service.On("List", ctx, tagSDK.ListResourcesOptions{Limit: ptr(maxPageSize), Offset: ptr(0)}).
		Return(resourcePage(maxPageSize), nil)
	service.On("List", ctx, tagSDK.ListResourcesOptions{Limit: ptr(maxPageSize), Offset: ptr(maxPageSize)}).
		Return(resourcePage(7), nil)

	dataSource, attachmentsSchema := newTestAttachmentsDataSource(t, service)
	resp := &datasource.ReadResponse{State: tfsdk.State{Schema: attachmentsSchema}}

	dataSource.Read(ctx, datasource.ReadRequest{
		Config: newDataSourceConfig(t, attachmentsSchema, nil),
	}, resp)

	require.False(t, resp.Diagnostics.HasError(), "%v", resp.Diagnostics)

	var state tagAttachmentsModel
	require.False(t, resp.State.Get(ctx, &state).HasError())

	assert.Len(t, state.Attachments, maxPageSize+7, "a full first page means there is a second one")
	service.AssertExpectations(t)
}

func TestAttachmentsDataSourceReadFilters(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	service := new(mocks.ResourceService)
	service.On("List", ctx, tagSDK.ListResourcesOptions{
		ResourceTypeName: ptr(tagSDK.ResourceTypeName("net.vpc")),
		Region:           ptr("br-se1"),
		Limit:            ptr(maxPageSize),
		Offset:           ptr(0),
	}).Return([]tagSDK.Resource{
		resourceFromJSON(t, `{
			"external_id": "`+testExternalID+`",
			"resource_type": {"name": "net.vpc", "product": "network"},
			"region": "br-se1",
			"created_at": "2026-08-03T00:57:47.908658",
			"tags": [{"name": "ambiente", "value": "producao"}]
		}`),
	}, nil)

	dataSource, attachmentsSchema := newTestAttachmentsDataSource(t, service)
	resp := &datasource.ReadResponse{State: tfsdk.State{Schema: attachmentsSchema}}

	dataSource.Read(ctx, datasource.ReadRequest{
		Config: newDataSourceConfig(t, attachmentsSchema, map[string]tftypes.Value{
			"resource_type": tftypes.NewValue(tftypes.String, "net.vpc"),
			"region":        tftypes.NewValue(tftypes.String, "br-se1"),
		}),
	}, resp)

	require.False(t, resp.Diagnostics.HasError(), "%v", resp.Diagnostics)

	var state tagAttachmentsModel
	require.False(t, resp.State.Get(ctx, &state).HasError())

	require.Len(t, state.Attachments, 1)
	assert.Equal(t, "net.vpc", state.ResourceType.ValueString(), "the filter stays in the state as the user wrote it")
	assert.Equal(t, "br-se1", state.Region.ValueString())
	service.AssertExpectations(t)
}

func TestAttachmentsDataSourceReadError(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	service := new(mocks.ResourceService)
	service.On("List", ctx, tagSDK.ListResourcesOptions{Limit: ptr(maxPageSize), Offset: ptr(0)}).
		Return(nil, errors.New("connection refused"))

	dataSource, attachmentsSchema := newTestAttachmentsDataSource(t, service)
	resp := &datasource.ReadResponse{State: tfsdk.State{Schema: attachmentsSchema}}

	dataSource.Read(ctx, datasource.ReadRequest{
		Config: newDataSourceConfig(t, attachmentsSchema, nil),
	}, resp)

	assert.True(t, resp.Diagnostics.HasError())
	service.AssertExpectations(t)
}
