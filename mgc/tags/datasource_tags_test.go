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

func newTestTagsDataSource(t *testing.T, service tagSDK.TagService) (*tagsDataSource, schema.Schema) {
	t.Helper()

	dataSource := &tagsDataSource{tags: service}
	return dataSource, testutils.GetDataSourceTestSchema(t, dataSource).Schema
}

// tagPage builds a page of the listing, big enough to be the answer the API gives
// when there is more to fetch.
func tagPage(size int) []tagSDK.Tag {
	page := make([]tagSDK.Tag, size)
	for i := range page {
		page[i] = tagSDK.Tag{Name: fmt.Sprintf("tag-%d", i)}
	}
	return page
}

func TestTagsDataSourceMetadata(t *testing.T) {
	t.Parallel()

	resp := &datasource.MetadataResponse{}
	NewTagsDataSource().Metadata(context.Background(), datasource.MetadataRequest{ProviderTypeName: "mgc"}, resp)

	assert.Equal(t, "mgc_tags", resp.TypeName)
}

func TestTagsDataSourceRead(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	service := new(mocks.TagService)
	service.On("List", ctx, tagSDK.ListTagsOptions{Limit: ptr(maxPageSize), Offset: ptr(0)}).
		Return([]tagSDK.Tag{
			tagFromJSON(t, `{"name": "tagfin", "color": "0086ff", "kinds": ["finops"], "created_at": "2026-08-03T00:57:47.908658"}`),
			tagFromJSON(t, `{"name": "ambiente", "created_at": "2026-08-03T00:58:00.000000"}`),
		}, nil)

	dataSource, tagsSchema := newTestTagsDataSource(t, service)
	resp := &datasource.ReadResponse{State: tfsdk.State{Schema: tagsSchema}}

	dataSource.Read(ctx, datasource.ReadRequest{
		Config: newDataSourceConfig(t, tagsSchema, nil),
	}, resp)

	require.False(t, resp.Diagnostics.HasError(), "%v", resp.Diagnostics)

	var state tagsModel
	require.False(t, resp.State.Get(ctx, &state).HasError())

	require.Len(t, state.Tags, 2)
	assert.Equal(t, "tagfin", state.Tags[0].Name.ValueString())
	assert.Equal(t, "tagfin", state.Tags[0].ID.ValueString())
	assert.Equal(t, "0086ff", state.Tags[0].Color.ValueString())
	assert.Equal(t, kindsSet("finops"), state.Tags[0].Kinds)
	assert.Equal(t, "ambiente", state.Tags[1].Name.ValueString())
	assert.True(t, state.Tags[1].Color.IsNull())
	assert.Empty(t, state.Tags[0].Values, "the listing does not embed the values of each tag")
	service.AssertExpectations(t)
}

func TestTagsDataSourceReadPagesThroughEveryTag(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	service := new(mocks.TagService)
	service.On("List", ctx, tagSDK.ListTagsOptions{Limit: ptr(maxPageSize), Offset: ptr(0)}).
		Return(tagPage(maxPageSize), nil)
	service.On("List", ctx, tagSDK.ListTagsOptions{Limit: ptr(maxPageSize), Offset: ptr(maxPageSize)}).
		Return(tagPage(3), nil)

	dataSource, tagsSchema := newTestTagsDataSource(t, service)
	resp := &datasource.ReadResponse{State: tfsdk.State{Schema: tagsSchema}}

	dataSource.Read(ctx, datasource.ReadRequest{
		Config: newDataSourceConfig(t, tagsSchema, nil),
	}, resp)

	require.False(t, resp.Diagnostics.HasError(), "%v", resp.Diagnostics)

	var state tagsModel
	require.False(t, resp.State.Get(ctx, &state).HasError())

	assert.Len(t, state.Tags, maxPageSize+3, "a full first page means there is a second one")
	service.AssertExpectations(t)
}

func TestTagsDataSourceReadFilters(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	service := new(mocks.TagService)
	service.On("List", ctx, tagSDK.ListTagsOptions{
		Color:  ptr("0086ff"),
		Kinds:  []tagSDK.TagKind{tagSDK.TagKindFinops},
		Limit:  ptr(maxPageSize),
		Offset: ptr(0),
	}).Return([]tagSDK.Tag{
		tagFromJSON(t, `{"name": "tagfin", "color": "0086ff", "kinds": ["finops"], "created_at": "2026-08-03T00:57:47.908658"}`),
	}, nil)

	dataSource, tagsSchema := newTestTagsDataSource(t, service)
	resp := &datasource.ReadResponse{State: tfsdk.State{Schema: tagsSchema}}

	kindsType := tftypes.Set{ElementType: tftypes.String}
	dataSource.Read(ctx, datasource.ReadRequest{
		Config: newDataSourceConfig(t, tagsSchema, map[string]tftypes.Value{
			"color": tftypes.NewValue(tftypes.String, "0086ff"),
			"kinds": tftypes.NewValue(kindsType, []tftypes.Value{tftypes.NewValue(tftypes.String, "finops")}),
		}),
	}, resp)

	require.False(t, resp.Diagnostics.HasError(), "%v", resp.Diagnostics)

	var state tagsModel
	require.False(t, resp.State.Get(ctx, &state).HasError())

	require.Len(t, state.Tags, 1)
	assert.Equal(t, "0086ff", state.Color.ValueString(), "the filter stays in the state as the user wrote it")
	service.AssertExpectations(t)
}

func TestTagsDataSourceReadError(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	service := new(mocks.TagService)
	service.On("List", ctx, tagSDK.ListTagsOptions{Limit: ptr(maxPageSize), Offset: ptr(0)}).
		Return(nil, errors.New("connection refused"))

	dataSource, tagsSchema := newTestTagsDataSource(t, service)
	resp := &datasource.ReadResponse{State: tfsdk.State{Schema: tagsSchema}}

	dataSource.Read(ctx, datasource.ReadRequest{
		Config: newDataSourceConfig(t, tagsSchema, nil),
	}, resp)

	assert.True(t, resp.Diagnostics.HasError())
	service.AssertExpectations(t)
}
