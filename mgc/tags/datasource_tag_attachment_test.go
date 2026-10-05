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

func newTestAttachmentDataSource(t *testing.T, service tagSDK.ResourceService) (*tagAttachmentDataSource, schema.Schema) {
	t.Helper()

	dataSource := &tagAttachmentDataSource{resources: service}
	return dataSource, testutils.GetDataSourceTestSchema(t, dataSource).Schema
}

func TestConvertAttachment(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		response string
		expected tagAttachmentModel
	}{
		{
			name: "every field filled",
			response: `{
				"id": "e1c4b1a0-0000-4000-8000-000000000001",
				"external_id": "b45659b5-2da2-4f60-9096-9d8dc7d96450",
				"resource_type": {"name": "net.vpc", "product": "network"},
				"region": "br-se1",
				"created_at": "2026-08-03T00:57:47.908658",
				"updated_at": "2026-08-04T10:00:00.000000",
				"last_tag_associated_at": "2026-08-05T11:00:00.000000",
				"tags": [{"name": "ambiente", "value": "producao"}]
			}`,
			expected: tagAttachmentModel{
				ID:                  types.StringValue("b45659b5-2da2-4f60-9096-9d8dc7d96450"),
				ResourceID:          types.StringValue("b45659b5-2da2-4f60-9096-9d8dc7d96450"),
				Tags:                tagsMap(map[string]string{"ambiente": "producao"}),
				ResourceType:        types.StringValue("net.vpc"),
				Product:             types.StringValue("network"),
				Region:              types.StringValue("br-se1"),
				CreatedAt:           types.StringValue("2026-08-03T00:57:47Z"),
				UpdatedAt:           types.StringValue("2026-08-04T10:00:00Z"),
				LastTagAssociatedAt: types.StringValue("2026-08-05T11:00:00Z"),
			},
		},
		{
			// The id of the tags service itself is deliberately left out: the
			// resource is identified by the id its own product uses for it.
			name: "external id is the identity, not the internal one",
			response: `{
				"id": "e1c4b1a0-0000-4000-8000-000000000001",
				"external_id": "b45659b5-2da2-4f60-9096-9d8dc7d96450",
				"resource_type": {"name": "net.vpc", "product": "network"},
				"region": "br-se1",
				"created_at": "2026-08-03T00:57:47.908658",
				"tags": [{"name": "ambiente", "value": "producao"}]
			}`,
			expected: tagAttachmentModel{
				ID:                  types.StringValue("b45659b5-2da2-4f60-9096-9d8dc7d96450"),
				ResourceID:          types.StringValue("b45659b5-2da2-4f60-9096-9d8dc7d96450"),
				Tags:                tagsMap(map[string]string{"ambiente": "producao"}),
				ResourceType:        types.StringValue("net.vpc"),
				Product:             types.StringValue("network"),
				Region:              types.StringValue("br-se1"),
				CreatedAt:           types.StringValue("2026-08-03T00:57:47Z"),
				UpdatedAt:           types.StringNull(),
				LastTagAssociatedAt: types.StringNull(),
			},
		},
		{
			name: "several tags become the whole map",
			response: `{
				"external_id": "b45659b5-2da2-4f60-9096-9d8dc7d96450",
				"resource_type": {"name": "un.unknown", "product": "unknown"},
				"region": "br-ne1",
				"created_at": "2026-08-03T00:57:47.908658",
				"tags": [
					{"name": "ambiente", "value": "producao"},
					{"name": "time", "value": "plataforma"}
				]
			}`,
			expected: tagAttachmentModel{
				ID:                  types.StringValue("b45659b5-2da2-4f60-9096-9d8dc7d96450"),
				ResourceID:          types.StringValue("b45659b5-2da2-4f60-9096-9d8dc7d96450"),
				Tags:                tagsMap(map[string]string{"ambiente": "producao", "time": "plataforma"}),
				ResourceType:        types.StringValue("un.unknown"),
				Product:             types.StringValue("unknown"),
				Region:              types.StringValue("br-ne1"),
				CreatedAt:           types.StringValue("2026-08-03T00:57:47Z"),
				UpdatedAt:           types.StringNull(),
				LastTagAssociatedAt: types.StringNull(),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.expected, convertAttachment(resourceFromJSON(t, tt.response)))
		})
	}
}

func TestAttachmentDataSourceMetadata(t *testing.T) {
	t.Parallel()

	resp := &datasource.MetadataResponse{}
	NewTagAttachmentDataSource().Metadata(context.Background(), datasource.MetadataRequest{ProviderTypeName: "mgc"}, resp)

	assert.Equal(t, "mgc_tag_attachment", resp.TypeName)
}

func TestAttachmentDataSourceSchema(t *testing.T) {
	t.Parallel()

	_, attachmentSchema := newTestAttachmentDataSource(t, nil)

	require.Contains(t, attachmentSchema.Attributes, "resource_id")
	assert.True(t, attachmentSchema.Attributes["resource_id"].IsRequired(), "the resource id is what the data source searches by")

	for _, name := range []string{"id", "tags", "resource_type", "product", "region", "created_at", "updated_at", "last_tag_associated_at"} {
		require.Contains(t, attachmentSchema.Attributes, name)
		assert.True(t, attachmentSchema.Attributes[name].IsComputed(), "%s is read from the API", name)
	}
}

func TestAttachmentDataSourceRead(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	service := new(mocks.ResourceService)
	service.On("Get", ctx, testExternalID).Return(ptr(resourceFromJSON(t, `{
		"external_id": "`+testExternalID+`",
		"resource_type": {"name": "net.vpc", "product": "network"},
		"region": "br-se1",
		"created_at": "2026-08-03T00:57:47.908658",
		"last_tag_associated_at": "2026-08-05T11:00:00.000000",
		"tags": [{"name": "ambiente", "value": "producao"}]
	}`)), nil)

	dataSource, attachmentSchema := newTestAttachmentDataSource(t, service)
	resp := &datasource.ReadResponse{State: tfsdk.State{Schema: attachmentSchema}}

	dataSource.Read(ctx, datasource.ReadRequest{
		Config: newDataSourceConfig(t, attachmentSchema, map[string]tftypes.Value{
			"resource_id": tftypes.NewValue(tftypes.String, testExternalID),
		}),
	}, resp)

	require.False(t, resp.Diagnostics.HasError(), "%v", resp.Diagnostics)

	var state tagAttachmentModel
	require.False(t, resp.State.Get(ctx, &state).HasError())

	assert.Equal(t, testExternalID, state.ID.ValueString())
	assert.Equal(t, testExternalID, state.ResourceID.ValueString())
	assert.Equal(t, tagsMap(map[string]string{"ambiente": "producao"}), state.Tags)
	assert.Equal(t, "net.vpc", state.ResourceType.ValueString())
	assert.Equal(t, "network", state.Product.ValueString())
	assert.Equal(t, "br-se1", state.Region.ValueString())

	// The resource hides this one because it changes on every attach and detach,
	// which would be drift. A data source is read fresh, so there is none.
	assert.Equal(t, "2026-08-05T11:00:00Z", state.LastTagAssociatedAt.ValueString())
	service.AssertExpectations(t)
}

func TestAttachmentDataSourceReadNotFound(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	// The API reports a resource that carries no tag the same way as one that does
	// not exist, and answers 400 instead of 404 for both.
	for _, status := range []int{http.StatusNotFound, http.StatusBadRequest} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			t.Parallel()

			service := new(mocks.ResourceService)
			service.On("Get", ctx, testExternalID).Return(nil, httpError(status, "not found"))

			dataSource, attachmentSchema := newTestAttachmentDataSource(t, service)
			resp := &datasource.ReadResponse{State: tfsdk.State{Schema: attachmentSchema}}

			dataSource.Read(ctx, datasource.ReadRequest{
				Config: newDataSourceConfig(t, attachmentSchema, map[string]tftypes.Value{
					"resource_id": tftypes.NewValue(tftypes.String, testExternalID),
				}),
			}, resp)

			require.True(t, resp.Diagnostics.HasError(), "a data source that finds nothing has to fail the plan")
			assert.Contains(t, resp.Diagnostics.Errors()[0].Summary(), "Tagged resource not found")
			assert.Contains(t, resp.Diagnostics.Errors()[0].Detail(), testExternalID)
			assert.Contains(t, resp.Diagnostics.Errors()[0].Detail(), "carries no tag")
			service.AssertExpectations(t)
		})
	}
}

func TestAttachmentDataSourceReadError(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	service := new(mocks.ResourceService)
	service.On("Get", ctx, testExternalID).Return(nil, errors.New("connection refused"))

	dataSource, attachmentSchema := newTestAttachmentDataSource(t, service)
	resp := &datasource.ReadResponse{State: tfsdk.State{Schema: attachmentSchema}}

	dataSource.Read(ctx, datasource.ReadRequest{
		Config: newDataSourceConfig(t, attachmentSchema, map[string]tftypes.Value{
			"resource_id": tftypes.NewValue(tftypes.String, testExternalID),
		}),
	}, resp)

	assert.True(t, resp.Diagnostics.HasError())
	service.AssertExpectations(t)
}
