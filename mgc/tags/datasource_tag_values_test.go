package tags

import (
	"context"
	"errors"
	"fmt"
	"net/http"
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

func newTestTagValuesDataSource(t *testing.T, service tagSDK.TagValueService) (*tagValuesDataSource, schema.Schema) {
	t.Helper()

	dataSource := &tagValuesDataSource{values: service}
	return dataSource, testutils.GetDataSourceTestSchema(t, dataSource).Schema
}

func tagValuePage(size int) []tagSDK.TagValue {
	page := make([]tagSDK.TagValue, size)
	for i := range page {
		page[i] = tagSDK.TagValue{Name: fmt.Sprintf("value-%d", i)}
	}
	return page
}

func TestTagValuesDataSourceMetadata(t *testing.T) {
	t.Parallel()

	resp := &datasource.MetadataResponse{}
	NewTagValuesDataSource().Metadata(context.Background(), datasource.MetadataRequest{ProviderTypeName: "mgc"}, resp)

	assert.Equal(t, "mgc_tag_values", resp.TypeName)
}

func TestTagValuesDataSourceRead(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	service := new(mocks.TagValueService)
	service.On("List", ctx, "tagfin", tagSDK.ListTagValuesOptions{Limit: ptr(maxPageSize), Offset: ptr(0)}).
		Return([]tagSDK.TagValue{
			tagValueFromJSON(t, `{"name": "producao", "description": "ambiente produtivo", "created_at": "2026-08-03T00:57:47.908658"}`),
			tagValueFromJSON(t, `{"name": "homologacao", "created_at": "2026-08-03T00:58:00.000000"}`),
		}, nil)

	dataSource, valuesSchema := newTestTagValuesDataSource(t, service)
	resp := &datasource.ReadResponse{State: tfsdk.State{Schema: valuesSchema}}

	dataSource.Read(ctx, datasource.ReadRequest{
		Config: newDataSourceConfig(t, valuesSchema, map[string]tftypes.Value{
			"tag_name": tftypes.NewValue(tftypes.String, "tagfin"),
		}),
	}, resp)

	require.False(t, resp.Diagnostics.HasError(), "%v", resp.Diagnostics)

	var state tagValuesModel
	require.False(t, resp.State.Get(ctx, &state).HasError())

	assert.Equal(t, "tagfin", state.TagName.ValueString())
	require.Len(t, state.Values, 2)
	assert.Equal(t, "tagfin,producao", state.Values[0].ID.ValueString())
	assert.Equal(t, "tagfin", state.Values[0].TagName.ValueString(), "each value carries the tag it belongs to")
	assert.Equal(t, "producao", state.Values[0].Name.ValueString())
	assert.Equal(t, "ambiente produtivo", state.Values[0].Description.ValueString())
	assert.Equal(t, "homologacao", state.Values[1].Name.ValueString())
	assert.True(t, state.Values[1].Description.IsNull())
	service.AssertExpectations(t)
}

func TestTagValuesDataSourceReadPagesThroughEveryValue(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	service := new(mocks.TagValueService)
	service.On("List", ctx, "tagfin", tagSDK.ListTagValuesOptions{Limit: ptr(maxPageSize), Offset: ptr(0)}).
		Return(tagValuePage(maxPageSize), nil)
	service.On("List", ctx, "tagfin", tagSDK.ListTagValuesOptions{Limit: ptr(maxPageSize), Offset: ptr(maxPageSize)}).
		Return(tagValuePage(5), nil)

	dataSource, valuesSchema := newTestTagValuesDataSource(t, service)
	resp := &datasource.ReadResponse{State: tfsdk.State{Schema: valuesSchema}}

	dataSource.Read(ctx, datasource.ReadRequest{
		Config: newDataSourceConfig(t, valuesSchema, map[string]tftypes.Value{
			"tag_name": tftypes.NewValue(tftypes.String, "tagfin"),
		}),
	}, resp)

	require.False(t, resp.Diagnostics.HasError(), "%v", resp.Diagnostics)

	var state tagValuesModel
	require.False(t, resp.State.Get(ctx, &state).HasError())

	assert.Len(t, state.Values, maxPageSize+5, "a full first page means there is a second one")
	service.AssertExpectations(t)
}

func TestTagValuesDataSourceReadTagNotFound(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	service := new(mocks.TagValueService)
	service.On("List", ctx, "ghost", tagSDK.ListTagValuesOptions{Limit: ptr(maxPageSize), Offset: ptr(0)}).
		Return(nil, httpError(http.StatusBadRequest, "not found"))

	dataSource, valuesSchema := newTestTagValuesDataSource(t, service)
	resp := &datasource.ReadResponse{State: tfsdk.State{Schema: valuesSchema}}

	dataSource.Read(ctx, datasource.ReadRequest{
		Config: newDataSourceConfig(t, valuesSchema, map[string]tftypes.Value{
			"tag_name": tftypes.NewValue(tftypes.String, "ghost"),
		}),
	}, resp)

	require.True(t, resp.Diagnostics.HasError())
	assert.Contains(t, resp.Diagnostics.Errors()[0].Summary(), "Tag not found")
	assert.Contains(t, resp.Diagnostics.Errors()[0].Detail(), "ghost")
	service.AssertExpectations(t)
}

func TestTagValuesDataSourceReadError(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	service := new(mocks.TagValueService)
	service.On("List", ctx, "tagfin", tagSDK.ListTagValuesOptions{Limit: ptr(maxPageSize), Offset: ptr(0)}).
		Return(nil, errors.New("connection refused"))

	dataSource, valuesSchema := newTestTagValuesDataSource(t, service)
	resp := &datasource.ReadResponse{State: tfsdk.State{Schema: valuesSchema}}

	dataSource.Read(ctx, datasource.ReadRequest{
		Config: newDataSourceConfig(t, valuesSchema, map[string]tftypes.Value{
			"tag_name": tftypes.NewValue(tftypes.String, "tagfin"),
		}),
	}, resp)

	assert.True(t, resp.Diagnostics.HasError())
	service.AssertExpectations(t)
}
