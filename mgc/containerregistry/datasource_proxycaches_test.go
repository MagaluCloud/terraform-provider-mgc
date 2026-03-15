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

func TestProxyCacheListDataSource_Metadata(t *testing.T) {
	d := NewProxyCacheListDataSource()
	req := datasource.MetadataRequest{
		ProviderTypeName: "mgc",
	}
	resp := &datasource.MetadataResponse{}

	d.Metadata(context.Background(), req, resp)

	assert.Equal(t, "mgc_container_registry_proxy_caches", resp.TypeName)
}

func TestProxyCacheListDataSource_Schema(t *testing.T) {
	d := NewProxyCacheListDataSource()
	req := datasource.SchemaRequest{}
	resp := &datasource.SchemaResponse{}

	d.Schema(context.Background(), req, resp)

	assert.NotNil(t, resp.Schema)
	assert.NotEmpty(t, resp.Schema.Description)
	assert.Contains(t, resp.Schema.Attributes, "proxy_caches")
}

func TestProxyCacheListDataSource_Read(t *testing.T) {
	ctx := context.Background()

	mockSvc := new(mocks.ProxyCachesService)
	mockSvc.On("ListAll", ctx, crSDK.ProxyCacheListAllOptions{}).Return([]crSDK.ProxyCache{
		{
			ID:        "proxy-1",
			Name:      "dockerhub-proxy",
			Provider:  "dockerhub",
			URL:       "https://registry-1.docker.io",
			CreatedAt: "2024-01-01T00:00:00Z",
			UpdatedAt: "2024-01-02T00:00:00Z",
		},
		{
			ID:        "proxy-2",
			Name:      "ghcr-proxy",
			Provider:  "ghcr",
			URL:       "https://ghcr.io",
			CreatedAt: "2024-02-01T00:00:00Z",
			UpdatedAt: "2024-02-02T00:00:00Z",
		},
	}, nil)

	d := &ProxyCacheListDataSource{
		proxyCacheService: mockSvc,
	}

	schemaResp := testutils.GetDataSourceTestSchema(t, d)

	proxyCachesType := tftypes.Object{
		AttributeTypes: map[string]tftypes.Type{
			"id":            tftypes.String,
			"name":          tftypes.String,
			"provider_name": tftypes.String,
			"url":           tftypes.String,
			"created_at":    tftypes.String,
			"updated_at":    tftypes.String,
		},
	}
	listType := tftypes.List{ElementType: proxyCachesType}

	configRaw := tftypes.NewValue(
		tftypes.Object{
			AttributeTypes: map[string]tftypes.Type{
				"proxy_caches": listType,
			},
		},
		map[string]tftypes.Value{
			"proxy_caches": tftypes.NewValue(listType, []tftypes.Value{}),
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

	var finalState proxyCacheListDataSourceModel
	resp.State.Get(ctx, &finalState)

	assert.Len(t, finalState.ProxyCaches, 2)
	assert.Equal(t, "proxy-1", finalState.ProxyCaches[0].ID.ValueString())
	assert.Equal(t, "dockerhub-proxy", finalState.ProxyCaches[0].Name.ValueString())
	assert.Equal(t, "dockerhub", finalState.ProxyCaches[0].ProviderName.ValueString())
	assert.Equal(t, "https://registry-1.docker.io", finalState.ProxyCaches[0].URL.ValueString())
	assert.Equal(t, "2024-01-01T00:00:00Z", finalState.ProxyCaches[0].CreatedAt.ValueString())
	assert.Equal(t, "2024-01-02T00:00:00Z", finalState.ProxyCaches[0].UpdatedAt.ValueString())

	assert.Equal(t, "proxy-2", finalState.ProxyCaches[1].ID.ValueString())
	assert.Equal(t, "ghcr-proxy", finalState.ProxyCaches[1].Name.ValueString())
	assert.Equal(t, "ghcr", finalState.ProxyCaches[1].ProviderName.ValueString())

	mockSvc.AssertExpectations(t)
}

func TestProxyCacheListDataSource_Read_EmptyList(t *testing.T) {
	ctx := context.Background()

	mockSvc := new(mocks.ProxyCachesService)
	mockSvc.On("ListAll", ctx, crSDK.ProxyCacheListAllOptions{}).Return([]crSDK.ProxyCache{}, nil)

	d := &ProxyCacheListDataSource{
		proxyCacheService: mockSvc,
	}

	schemaResp := testutils.GetDataSourceTestSchema(t, d)

	proxyCachesType := tftypes.Object{
		AttributeTypes: map[string]tftypes.Type{
			"id":            tftypes.String,
			"name":          tftypes.String,
			"provider_name": tftypes.String,
			"url":           tftypes.String,
			"created_at":    tftypes.String,
			"updated_at":    tftypes.String,
		},
	}
	listType := tftypes.List{ElementType: proxyCachesType}

	configRaw := tftypes.NewValue(
		tftypes.Object{
			AttributeTypes: map[string]tftypes.Type{
				"proxy_caches": listType,
			},
		},
		map[string]tftypes.Value{
			"proxy_caches": tftypes.NewValue(listType, []tftypes.Value{}),
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

	assert.False(t, resp.Diagnostics.HasError())

	var finalState proxyCacheListDataSourceModel
	resp.State.Get(ctx, &finalState)

	assert.Len(t, finalState.ProxyCaches, 0)
	mockSvc.AssertExpectations(t)
}

func TestProxyCacheListDataSource_Read_APIError(t *testing.T) {
	ctx := context.Background()

	mockSvc := new(mocks.ProxyCachesService)
	mockSvc.On("ListAll", ctx, crSDK.ProxyCacheListAllOptions{}).Return([]crSDK.ProxyCache(nil), errors.New("api connection error"))

	d := &ProxyCacheListDataSource{
		proxyCacheService: mockSvc,
	}

	schemaResp := testutils.GetDataSourceTestSchema(t, d)

	proxyCachesType := tftypes.Object{
		AttributeTypes: map[string]tftypes.Type{
			"id":            tftypes.String,
			"name":          tftypes.String,
			"provider_name": tftypes.String,
			"url":           tftypes.String,
			"created_at":    tftypes.String,
			"updated_at":    tftypes.String,
		},
	}
	listType := tftypes.List{ElementType: proxyCachesType}

	configRaw := tftypes.NewValue(
		tftypes.Object{
			AttributeTypes: map[string]tftypes.Type{
				"proxy_caches": listType,
			},
		},
		map[string]tftypes.Value{
			"proxy_caches": tftypes.NewValue(listType, []tftypes.Value{}),
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

	assert.True(t, resp.Diagnostics.HasError(), "Read operation should return diagnostics errors on API failure")
	mockSvc.AssertExpectations(t)
}
