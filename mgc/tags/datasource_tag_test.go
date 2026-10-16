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

func newTestTagDataSource(t *testing.T, service tagSDK.TagService) (*tagDataSource, schema.Schema) {
	t.Helper()

	dataSource := &tagDataSource{tags: service}
	return dataSource, testutils.GetDataSourceTestSchema(t, dataSource).Schema
}

func TestConvertTag(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		response string
		expected tagModel
	}{
		{
			name: "every field filled",
			response: `{
				"name": "tagfin",
				"description": "monitoramento",
				"color": "0086ff",
				"kinds": ["finops"],
				"created_at": "2026-08-03T00:57:47.908658",
				"updated_at": "2026-08-04T10:00:00.000000",
				"values": [
					{"name": "producao", "description": "ambiente produtivo", "created_at": "2026-08-03T01:00:00.000000", "updated_at": null}
				]
			}`,
			expected: tagModel{
				ID:          types.StringValue("tagfin"),
				Name:        types.StringValue("tagfin"),
				Description: types.StringValue("monitoramento"),
				Color:       types.StringValue("0086ff"),
				Kinds:       kindsSet("finops"),
				Values: []tagValueModel{{
					ID:          types.StringValue("tagfin,producao"),
					TagName:     types.StringValue("tagfin"),
					Name:        types.StringValue("producao"),
					Description: types.StringValue("ambiente produtivo"),
					CreatedAt:   types.StringValue("2026-08-03T01:00:00Z"),
					UpdatedAt:   types.StringNull(),
				}},
				CreatedAt: types.StringValue("2026-08-03T00:57:47Z"),
				UpdatedAt: types.StringValue("2026-08-04T10:00:00Z"),
			},
		},
		{
			// The tag has no values, which the listing endpoint reports by leaving
			// the field out and the tag endpoint by answering an empty array.
			name: "empty values stay an empty list",
			response: `{
				"name": "tagfin",
				"kinds": [],
				"values": [],
				"created_at": "2026-08-03T00:57:47.908658",
				"updated_at": null
			}`,
			expected: tagModel{
				ID:          types.StringValue("tagfin"),
				Name:        types.StringValue("tagfin"),
				Description: types.StringNull(),
				Color:       types.StringNull(),
				Kinds:       kindsSet(),
				Values:      []tagValueModel{},
				CreatedAt:   types.StringValue("2026-08-03T00:57:47Z"),
				UpdatedAt:   types.StringNull(),
			},
		},
		{
			name: "optional fields absent stay null",
			response: `{
				"name": "tagfin",
				"description": null,
				"color": null,
				"kinds": null,
				"created_at": "2026-08-03T00:57:47.908658",
				"updated_at": null
			}`,
			expected: tagModel{
				ID:          types.StringValue("tagfin"),
				Name:        types.StringValue("tagfin"),
				Description: types.StringNull(),
				Color:       types.StringNull(),
				Kinds:       types.SetNull(types.StringType),
				CreatedAt:   types.StringValue("2026-08-03T00:57:47Z"),
				UpdatedAt:   types.StringNull(),
			},
		},
		{
			// A tag without kinds answers with [], which is not the same as the
			// absent field above: the data source reports what the API said.
			name: "empty kinds stay an empty set",
			response: `{
				"name": "tagfin",
				"kinds": [],
				"created_at": "2026-08-03T00:57:47.908658",
				"updated_at": null
			}`,
			expected: tagModel{
				ID:          types.StringValue("tagfin"),
				Name:        types.StringValue("tagfin"),
				Description: types.StringNull(),
				Color:       types.StringNull(),
				Kinds:       kindsSet(),
				CreatedAt:   types.StringValue("2026-08-03T00:57:47Z"),
				UpdatedAt:   types.StringNull(),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.expected, convertTag(tagFromJSON(t, tt.response)))
		})
	}
}

func TestTagDataSourceMetadata(t *testing.T) {
	t.Parallel()

	resp := &datasource.MetadataResponse{}
	NewTagDataSource().Metadata(context.Background(), datasource.MetadataRequest{ProviderTypeName: "mgc"}, resp)

	assert.Equal(t, "mgc_tag", resp.TypeName)
}

func TestTagDataSourceSchema(t *testing.T) {
	t.Parallel()

	_, tagSchema := newTestTagDataSource(t, nil)

	require.Contains(t, tagSchema.Attributes, "name")
	assert.True(t, tagSchema.Attributes["name"].IsRequired(), "the name is what the data source searches by")

	for _, name := range []string{"id", "description", "color", "kinds", "values", "created_at", "updated_at"} {
		require.Contains(t, tagSchema.Attributes, name)
		assert.True(t, tagSchema.Attributes[name].IsComputed(), "%s is read from the API", name)
	}
}

func TestTagDataSourceRead(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	service := new(mocks.TagService)
	service.On("Get", ctx, "tagfin").Return(ptr(tagFromJSON(t, `{
		"name": "tagfin",
		"description": "monitoramento",
		"color": "0086ff",
		"kinds": ["finops"],
		"values": [
			{"name": "producao", "created_at": "2026-08-03T01:00:00.000000", "updated_at": null}
		],
		"created_at": "2026-08-03T00:57:47.908658",
		"updated_at": null
	}`)), nil)

	dataSource, tagSchema := newTestTagDataSource(t, service)
	resp := &datasource.ReadResponse{State: tfsdk.State{Schema: tagSchema}}

	dataSource.Read(ctx, datasource.ReadRequest{
		Config: newDataSourceConfig(t, tagSchema, map[string]tftypes.Value{
			"name": tftypes.NewValue(tftypes.String, "tagfin"),
		}),
	}, resp)

	require.False(t, resp.Diagnostics.HasError(), "%v", resp.Diagnostics)

	var state tagModel
	require.False(t, resp.State.Get(ctx, &state).HasError())

	assert.Equal(t, "tagfin", state.ID.ValueString())
	assert.Equal(t, "tagfin", state.Name.ValueString())
	assert.Equal(t, "monitoramento", state.Description.ValueString())
	assert.Equal(t, "0086ff", state.Color.ValueString())
	assert.Equal(t, kindsSet("finops"), state.Kinds)
	assert.Equal(t, "2026-08-03T00:57:47Z", state.CreatedAt.ValueString())
	assert.True(t, state.UpdatedAt.IsNull())

	require.Len(t, state.Values, 1, "the tag endpoint embeds the values of the tag")
	assert.Equal(t, "producao", state.Values[0].Name.ValueString())
	assert.Equal(t, "tagfin,producao", state.Values[0].ID.ValueString())
	service.AssertExpectations(t)
}

func TestTagDataSourceReadNotFound(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	// The tags API answers 400 for a tag that does not exist, so both codes have
	// to reach the user as the same "not found" message.
	for _, status := range []int{http.StatusNotFound, http.StatusBadRequest} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			t.Parallel()

			service := new(mocks.TagService)
			service.On("Get", ctx, "ghost").Return(nil, httpError(status, "not found"))

			dataSource, tagSchema := newTestTagDataSource(t, service)
			resp := &datasource.ReadResponse{State: tfsdk.State{Schema: tagSchema}}

			dataSource.Read(ctx, datasource.ReadRequest{
				Config: newDataSourceConfig(t, tagSchema, map[string]tftypes.Value{
					"name": tftypes.NewValue(tftypes.String, "ghost"),
				}),
			}, resp)

			require.True(t, resp.Diagnostics.HasError(), "a data source that finds nothing has to fail the plan")
			assert.Contains(t, resp.Diagnostics.Errors()[0].Summary(), "Tag not found")
			assert.Contains(t, resp.Diagnostics.Errors()[0].Detail(), "ghost")
			service.AssertExpectations(t)
		})
	}
}

func TestTagDataSourceReadError(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	service := new(mocks.TagService)
	service.On("Get", ctx, "tagfin").Return(nil, errors.New("connection refused"))

	dataSource, tagSchema := newTestTagDataSource(t, service)
	resp := &datasource.ReadResponse{State: tfsdk.State{Schema: tagSchema}}

	dataSource.Read(ctx, datasource.ReadRequest{
		Config: newDataSourceConfig(t, tagSchema, map[string]tftypes.Value{
			"name": tftypes.NewValue(tftypes.String, "tagfin"),
		}),
	}, resp)

	assert.True(t, resp.Diagnostics.HasError())
	service.AssertExpectations(t)
}
