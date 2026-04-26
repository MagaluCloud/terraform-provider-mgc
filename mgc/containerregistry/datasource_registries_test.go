package containerregistry

import (
	"context"
	"errors"
	"testing"

	crSDK "github.com/MagaluCloud/mgc-sdk-go/containerregistry"
	"github.com/MagaluCloud/terraform-provider-mgc/mgc/internal/mocks"
	"github.com/MagaluCloud/terraform-provider-mgc/mgc/internal/testutils"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/stretchr/testify/assert"
)

func TestDataSourceCRRegistries_Metadata(t *testing.T) {
	t.Parallel()
	d := NewDataSourceCRRegistries()
	req := datasource.MetadataRequest{
		ProviderTypeName: "mgc",
	}
	resp := &datasource.MetadataResponse{}

	d.Metadata(context.Background(), req, resp)

	assert.Equal(t, "mgc_container_registries", resp.TypeName)
}

func TestDataSourceCRRegistries_Schema(t *testing.T) {
	t.Parallel()
	d := NewDataSourceCRRegistries()
	req := datasource.SchemaRequest{}
	resp := &datasource.SchemaResponse{}

	d.Schema(context.Background(), req, resp)

	assert.NotNil(t, resp.Schema)
	assert.NotEmpty(t, resp.Schema.Description)
	assert.Contains(t, resp.Schema.Attributes, "registries")
}

func TestDataSourceCRRegistries_Read(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	mockSvc := new(mocks.RegistriesService)
	mockSvc.On("ListAll", ctx, crSDK.RegistryFilterOptions{}).Return([]crSDK.RegistryResponse{
		{
			ID:           "reg-1",
			Name:         "my-registry-1",
			Storage:      1024000,
			ProxyCacheID: strPtr("proxy-1"),
			CreatedAt:    "2024-01-01T00:00:00Z",
			UpdatedAt:    "2024-01-02T00:00:00Z",
		},
		{
			ID:           "reg-2",
			Name:         "my-registry-2",
			Storage:      2048000,
			ProxyCacheID: nil,
			CreatedAt:    "2024-02-01T00:00:00Z",
			UpdatedAt:    "2024-02-02T00:00:00Z",
		},
	}, nil)

	d := &DataSourceCRRegistries{
		crRegistries: mockSvc,
	}

	schemaResp := testutils.GetDataSourceTestSchema(t, d)

	registriesType := tftypes.Object{
		AttributeTypes: map[string]tftypes.Type{
			"id":                  tftypes.String,
			"name":                tftypes.String,
			"storage_usage_bytes": tftypes.Number,
			"proxy_cache_id":      tftypes.String,
			"created_at":          tftypes.String,
			"updated_at":          tftypes.String,
		},
	}
	listType := tftypes.List{ElementType: registriesType}

	configRaw := tftypes.NewValue(
		tftypes.Object{
			AttributeTypes: map[string]tftypes.Type{
				"registries": listType,
			},
		},
		map[string]tftypes.Value{
			"registries": tftypes.NewValue(listType, []tftypes.Value{}),
		},
	)

	config := tfsdk.Config{
		Schema: schemaResp.Schema,
		Raw:    configRaw,
	}

	req := datasource.ReadRequest{
		Config: config,
	}

	resp := &datasource.ReadResponse{
		State: tfsdk.State{Schema: schemaResp.Schema},
	}

	d.Read(ctx, req, resp)

	assert.False(t, resp.Diagnostics.HasError(), "Read operation returned diagnostics errors: %v", resp.Diagnostics)

	var finalState crRegistriesList
	resp.State.Get(ctx, &finalState)

	assert.Len(t, finalState.Registries, 2)
	assert.Equal(t, "reg-1", finalState.Registries[0].ID.ValueString())
	assert.Equal(t, "my-registry-1", finalState.Registries[0].Name.ValueString())
	assert.Equal(t, int64(1024000), finalState.Registries[0].StorageUsageBytes.ValueInt64())
	assert.Equal(t, "proxy-1", finalState.Registries[0].ProxyCacheID.ValueString())
	assert.Equal(t, "2024-01-01T00:00:00Z", finalState.Registries[0].CreatedAt.ValueString())
	assert.Equal(t, "2024-01-02T00:00:00Z", finalState.Registries[0].UpdatedAt.ValueString())

	assert.Equal(t, "reg-2", finalState.Registries[1].ID.ValueString())
	assert.Equal(t, "my-registry-2", finalState.Registries[1].Name.ValueString())
	assert.Equal(t, int64(2048000), finalState.Registries[1].StorageUsageBytes.ValueInt64())
	assert.True(t, finalState.Registries[1].ProxyCacheID.IsNull())

	mockSvc.AssertExpectations(t)
}

func TestDataSourceCRRegistries_Read_EmptyList(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	mockSvc := new(mocks.RegistriesService)
	mockSvc.On("ListAll", ctx, crSDK.RegistryFilterOptions{}).Return([]crSDK.RegistryResponse{}, nil)

	d := &DataSourceCRRegistries{
		crRegistries: mockSvc,
	}

	schemaResp := testutils.GetDataSourceTestSchema(t, d)

	registriesType := tftypes.Object{
		AttributeTypes: map[string]tftypes.Type{
			"id":                  tftypes.String,
			"name":                tftypes.String,
			"storage_usage_bytes": tftypes.Number,
			"proxy_cache_id":      tftypes.String,
			"created_at":          tftypes.String,
			"updated_at":          tftypes.String,
		},
	}
	listType := tftypes.List{ElementType: registriesType}

	config := tfsdk.Config{
		Schema: schemaResp.Schema,
		Raw: tftypes.NewValue(
			tftypes.Object{
				AttributeTypes: map[string]tftypes.Type{
					"registries": listType,
				},
			},
			map[string]tftypes.Value{
				"registries": tftypes.NewValue(listType, []tftypes.Value{}),
			},
		),
	}

	req := datasource.ReadRequest{
		Config: config,
	}

	resp := &datasource.ReadResponse{
		State: tfsdk.State{Schema: schemaResp.Schema},
	}

	d.Read(ctx, req, resp)

	assert.False(t, resp.Diagnostics.HasError())

	var finalState crRegistriesList
	resp.State.Get(ctx, &finalState)

	assert.Len(t, finalState.Registries, 0)
	mockSvc.AssertExpectations(t)
}

func strPtr(s string) *string {
	return &s
}

func TestDataSourceCRRegistries_Read_APIError(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	mockSvc := new(mocks.RegistriesService)
	mockSvc.On("ListAll", ctx, crSDK.RegistryFilterOptions{}).Return([]crSDK.RegistryResponse(nil), errors.New("api connection error"))

	d := &DataSourceCRRegistries{
		crRegistries: mockSvc,
	}

	schemaResp := testutils.GetDataSourceTestSchema(t, d)

	registriesType := tftypes.Object{
		AttributeTypes: map[string]tftypes.Type{
			"id":                  tftypes.String,
			"name":                tftypes.String,
			"storage_usage_bytes": tftypes.Number,
			"proxy_cache_id":      tftypes.String,
			"created_at":          tftypes.String,
			"updated_at":          tftypes.String,
		},
	}
	listType := tftypes.List{ElementType: registriesType}

	config := tfsdk.Config{
		Schema: schemaResp.Schema,
		Raw: tftypes.NewValue(
			tftypes.Object{
				AttributeTypes: map[string]tftypes.Type{
					"registries": listType,
				},
			},
			map[string]tftypes.Value{
				"registries": tftypes.NewValue(listType, []tftypes.Value{}),
			},
		),
	}

	req := datasource.ReadRequest{
		Config: config,
	}

	resp := &datasource.ReadResponse{
		State: tfsdk.State{Schema: schemaResp.Schema},
	}

	d.Read(ctx, req, resp)

	assert.True(t, resp.Diagnostics.HasError(), "Read operation should return diagnostics errors on API failure")
	mockSvc.AssertExpectations(t)
}
