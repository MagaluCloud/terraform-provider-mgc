package tags

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
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

func newTestResourceTypesDataSource(t *testing.T, service tagSDK.ResourceTypeService) (*tagResourceTypesDataSource, schema.Schema) {
	t.Helper()

	dataSource := &tagResourceTypesDataSource{resourceTypes: service}
	return dataSource, testutils.GetDataSourceTestSchema(t, dataSource).Schema
}

func resourceTypeFromJSON(t *testing.T, raw string) tagSDK.ResourceType {
	t.Helper()

	var resourceType tagSDK.ResourceType
	require.NoError(t, json.Unmarshal([]byte(raw), &resourceType))
	return resourceType
}

func resourceTypePage(size int) []tagSDK.ResourceType {
	page := make([]tagSDK.ResourceType, size)
	for i := range page {
		page[i] = tagSDK.ResourceType{Name: tagSDK.ResourceTypeName(fmt.Sprintf("prod.type-%d", i))}
	}
	return page
}

func TestConvertResourceType(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		response string
		expected tagResourceTypeModel
	}{
		{
			name: "every field filled",
			response: `{
				"name": "net.vpc",
				"product": "network",
				"created_at": "2026-08-03T00:57:47.908658",
				"updated_at": "2026-08-04T10:00:00.000000"
			}`,
			expected: tagResourceTypeModel{
				Name:      types.StringValue("net.vpc"),
				Product:   types.StringValue("network"),
				CreatedAt: types.StringValue("2026-08-03T00:57:47Z"),
				UpdatedAt: types.StringValue("2026-08-04T10:00:00Z"),
			},
		},
		{
			name: "never updated stays null",
			response: `{
				"name": "k8s.cluster",
				"product": "kubernetes",
				"created_at": "2026-08-03T00:57:47.908658",
				"updated_at": null
			}`,
			expected: tagResourceTypeModel{
				Name:      types.StringValue("k8s.cluster"),
				Product:   types.StringValue("kubernetes"),
				CreatedAt: types.StringValue("2026-08-03T00:57:47Z"),
				UpdatedAt: types.StringNull(),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.expected, convertResourceType(resourceTypeFromJSON(t, tt.response)))
		})
	}
}

func TestResourceTypesDataSourceMetadata(t *testing.T) {
	t.Parallel()

	resp := &datasource.MetadataResponse{}
	NewTagResourceTypesDataSource().Metadata(context.Background(), datasource.MetadataRequest{ProviderTypeName: "mgc"}, resp)

	assert.Equal(t, "mgc_tag_resource_types", resp.TypeName)
}

func TestResourceTypesDataSourceRead(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	service := new(mocks.ResourceTypeService)
	service.On("List", ctx, tagSDK.ListResourceTypesOptions{Limit: ptr(maxPageSize), Offset: ptr(0)}).
		Return([]tagSDK.ResourceType{
			resourceTypeFromJSON(t, `{"name": "net.vpc", "product": "network", "created_at": "2026-08-03T00:57:47.908658"}`),
			resourceTypeFromJSON(t, `{"name": "k8s.cluster", "product": "kubernetes", "created_at": "2026-08-03T01:00:00.000000"}`),
		}, nil)

	dataSource, typesSchema := newTestResourceTypesDataSource(t, service)
	resp := &datasource.ReadResponse{State: tfsdk.State{Schema: typesSchema}}

	dataSource.Read(ctx, datasource.ReadRequest{
		Config: newDataSourceConfig(t, typesSchema, nil),
	}, resp)

	require.False(t, resp.Diagnostics.HasError(), "%v", resp.Diagnostics)

	var state tagResourceTypesModel
	require.False(t, resp.State.Get(ctx, &state).HasError())

	require.Len(t, state.ResourceTypes, 2)
	assert.Equal(t, "net.vpc", state.ResourceTypes[0].Name.ValueString())
	assert.Equal(t, "network", state.ResourceTypes[0].Product.ValueString())
	assert.Equal(t, "k8s.cluster", state.ResourceTypes[1].Name.ValueString())
	service.AssertExpectations(t)
}

func TestResourceTypesDataSourceReadPagesThroughEveryType(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	service := new(mocks.ResourceTypeService)
	service.On("List", ctx, tagSDK.ListResourceTypesOptions{Limit: ptr(maxPageSize), Offset: ptr(0)}).
		Return(resourceTypePage(maxPageSize), nil)
	service.On("List", ctx, tagSDK.ListResourceTypesOptions{Limit: ptr(maxPageSize), Offset: ptr(maxPageSize)}).
		Return(resourceTypePage(2), nil)

	dataSource, typesSchema := newTestResourceTypesDataSource(t, service)
	resp := &datasource.ReadResponse{State: tfsdk.State{Schema: typesSchema}}

	dataSource.Read(ctx, datasource.ReadRequest{
		Config: newDataSourceConfig(t, typesSchema, nil),
	}, resp)

	require.False(t, resp.Diagnostics.HasError(), "%v", resp.Diagnostics)

	var state tagResourceTypesModel
	require.False(t, resp.State.Get(ctx, &state).HasError())

	assert.Len(t, state.ResourceTypes, maxPageSize+2, "a full first page means there is a second one")
	service.AssertExpectations(t)
}

func TestResourceTypesDataSourceReadFilters(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	service := new(mocks.ResourceTypeService)
	service.On("List", ctx, tagSDK.ListResourceTypesOptions{
		Product: ptr(tagSDK.Product("network")),
		Limit:   ptr(maxPageSize),
		Offset:  ptr(0),
	}).Return([]tagSDK.ResourceType{
		resourceTypeFromJSON(t, `{"name": "net.vpc", "product": "network", "created_at": "2026-08-03T00:57:47.908658"}`),
	}, nil)

	dataSource, typesSchema := newTestResourceTypesDataSource(t, service)
	resp := &datasource.ReadResponse{State: tfsdk.State{Schema: typesSchema}}

	dataSource.Read(ctx, datasource.ReadRequest{
		Config: newDataSourceConfig(t, typesSchema, map[string]tftypes.Value{
			"product": tftypes.NewValue(tftypes.String, "network"),
		}),
	}, resp)

	require.False(t, resp.Diagnostics.HasError(), "%v", resp.Diagnostics)

	var state tagResourceTypesModel
	require.False(t, resp.State.Get(ctx, &state).HasError())

	require.Len(t, state.ResourceTypes, 1)
	assert.Equal(t, "network", state.Product.ValueString(), "the filter stays in the state as the user wrote it")
	service.AssertExpectations(t)
}

func TestResourceTypesDataSourceReadError(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	service := new(mocks.ResourceTypeService)
	service.On("List", ctx, tagSDK.ListResourceTypesOptions{Limit: ptr(maxPageSize), Offset: ptr(0)}).
		Return(nil, errors.New("connection refused"))

	dataSource, typesSchema := newTestResourceTypesDataSource(t, service)
	resp := &datasource.ReadResponse{State: tfsdk.State{Schema: typesSchema}}

	dataSource.Read(ctx, datasource.ReadRequest{
		Config: newDataSourceConfig(t, typesSchema, nil),
	}, resp)

	assert.True(t, resp.Diagnostics.HasError())
	service.AssertExpectations(t)
}
