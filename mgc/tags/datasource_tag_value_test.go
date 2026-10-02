package tags

import (
	"context"
	"errors"
	"net/http"
	"testing"

	tagSDK "github.com/MagaluCloud/mgc-sdk-go/tag"
	"github.com/MagaluCloud/terraform-provider-mgc/mgc/internal/mocks"
	"github.com/MagaluCloud/terraform-provider-mgc/mgc/internal/testutils"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestTagValueDataSource(t *testing.T, service tagSDK.TagValueService) (*tagValueDataSource, schema.Schema) {
	t.Helper()

	dataSource := &tagValueDataSource{values: service}
	return dataSource, testutils.GetDataSourceTestSchema(t, dataSource).Schema
}

func TestConvertTagValue(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		tagName  string
		response string
		expected tagValueModel
	}{
		{
			name:    "every field filled",
			tagName: "tagfin",
			response: `{
				"name": "producao",
				"description": "ambiente produtivo",
				"created_at": "2026-08-03T00:57:47.908658",
				"updated_at": "2026-08-04T10:00:00.000000"
			}`,
			expected: tagValueModel{
				ID:          types.StringValue("tagfin,producao"),
				TagName:     types.StringValue("tagfin"),
				Name:        types.StringValue("producao"),
				Description: types.StringValue("ambiente produtivo"),
				CreatedAt:   types.StringValue("2026-08-03T00:57:47Z"),
				UpdatedAt:   types.StringValue("2026-08-04T10:00:00Z"),
			},
		},
		{
			name:    "optional fields absent stay null",
			tagName: "tagfin",
			response: `{
				"name": "producao",
				"description": null,
				"created_at": "2026-08-03T00:57:47.908658",
				"updated_at": null
			}`,
			expected: tagValueModel{
				ID:          types.StringValue("tagfin,producao"),
				TagName:     types.StringValue("tagfin"),
				Name:        types.StringValue("producao"),
				Description: types.StringNull(),
				CreatedAt:   types.StringValue("2026-08-03T00:57:47Z"),
				UpdatedAt:   types.StringNull(),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.expected, convertTagValue(tt.tagName, tagValueFromJSON(t, tt.response)))
		})
	}
}

func TestTagValueDataSourceMetadata(t *testing.T) {
	t.Parallel()

	resp := &datasource.MetadataResponse{}
	NewTagValueDataSource().Metadata(context.Background(), datasource.MetadataRequest{ProviderTypeName: "mgc"}, resp)

	assert.Equal(t, "mgc_tag_value", resp.TypeName)
}

func TestTagValueDataSourceSchema(t *testing.T) {
	t.Parallel()

	_, valueSchema := newTestTagValueDataSource(t, nil)

	for _, name := range []string{"tag_name", "name"} {
		require.Contains(t, valueSchema.Attributes, name)
		assert.True(t, valueSchema.Attributes[name].IsRequired(), "%s is part of the search key", name)
	}

	for _, name := range []string{"id", "description", "created_at", "updated_at"} {
		require.Contains(t, valueSchema.Attributes, name)
		assert.True(t, valueSchema.Attributes[name].IsComputed(), "%s is read from the API", name)
	}
}

func TestTagValueDataSourceRead(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	service := new(mocks.TagValueService)
	service.On("Get", ctx, "tagfin", "producao").Return(ptr(tagValueFromJSON(t, `{
		"name": "producao",
		"description": "ambiente produtivo",
		"created_at": "2026-08-03T00:57:47.908658",
		"updated_at": null
	}`)), nil)

	dataSource, valueSchema := newTestTagValueDataSource(t, service)
	resp := &datasource.ReadResponse{State: tfsdk.State{Schema: valueSchema}}

	dataSource.Read(ctx, datasource.ReadRequest{
		Config: newDataSourceConfig(t, valueSchema, map[string]tftypes.Value{
			"tag_name": tftypes.NewValue(tftypes.String, "tagfin"),
			"name":     tftypes.NewValue(tftypes.String, "producao"),
		}),
	}, resp)

	require.False(t, resp.Diagnostics.HasError(), "%v", resp.Diagnostics)

	var state tagValueModel
	require.False(t, resp.State.Get(ctx, &state).HasError())

	assert.Equal(t, "tagfin,producao", state.ID.ValueString())
	assert.Equal(t, "tagfin", state.TagName.ValueString())
	assert.Equal(t, "producao", state.Name.ValueString())
	assert.Equal(t, "ambiente produtivo", state.Description.ValueString())
	assert.Equal(t, "2026-08-03T00:57:47Z", state.CreatedAt.ValueString())
	assert.True(t, state.UpdatedAt.IsNull())
	service.AssertExpectations(t)
}

func TestTagValueDataSourceReadNotFound(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	// A missing tag and a missing value are reported the same way, and 400 stands
	// for not found in this API, so all of them have to land on the same message.
	for _, status := range []int{http.StatusNotFound, http.StatusBadRequest} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			t.Parallel()

			service := new(mocks.TagValueService)
			service.On("Get", ctx, "tagfin", "ghost").Return(nil, httpError(status, "not found"))

			dataSource, valueSchema := newTestTagValueDataSource(t, service)
			resp := &datasource.ReadResponse{State: tfsdk.State{Schema: valueSchema}}

			dataSource.Read(ctx, datasource.ReadRequest{
				Config: newDataSourceConfig(t, valueSchema, map[string]tftypes.Value{
					"tag_name": tftypes.NewValue(tftypes.String, "tagfin"),
					"name":     tftypes.NewValue(tftypes.String, "ghost"),
				}),
			}, resp)

			require.True(t, resp.Diagnostics.HasError(), "a data source that finds nothing has to fail the plan")
			assert.Contains(t, resp.Diagnostics.Errors()[0].Summary(), "Tag value not found")
			assert.Contains(t, resp.Diagnostics.Errors()[0].Detail(), "ghost")
			assert.Contains(t, resp.Diagnostics.Errors()[0].Detail(), "tagfin")
			service.AssertExpectations(t)
		})
	}
}

func TestTagValueDataSourceReadError(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	service := new(mocks.TagValueService)
	service.On("Get", ctx, "tagfin", "producao").Return(nil, errors.New("connection refused"))

	dataSource, valueSchema := newTestTagValueDataSource(t, service)
	resp := &datasource.ReadResponse{State: tfsdk.State{Schema: valueSchema}}

	dataSource.Read(ctx, datasource.ReadRequest{
		Config: newDataSourceConfig(t, valueSchema, map[string]tftypes.Value{
			"tag_name": tftypes.NewValue(tftypes.String, "tagfin"),
			"name":     tftypes.NewValue(tftypes.String, "producao"),
		}),
	}, resp)

	assert.True(t, resp.Diagnostics.HasError())
	service.AssertExpectations(t)
}
