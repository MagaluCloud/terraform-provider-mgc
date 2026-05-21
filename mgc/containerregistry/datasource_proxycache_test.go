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

func TestProxyCacheDataSource_Metadata(t *testing.T) {
	t.Parallel()
	d := NewProxyCacheDataSource()
	req := datasource.MetadataRequest{
		ProviderTypeName: "mgc",
	}
	resp := &datasource.MetadataResponse{}

	d.Metadata(context.Background(), req, resp)

	assert.Equal(t, "mgc_container_registry_proxy_cache", resp.TypeName)
}

func TestProxyCacheDataSource_Schema(t *testing.T) {
	t.Parallel()
	d := NewProxyCacheDataSource()
	req := datasource.SchemaRequest{}
	resp := &datasource.SchemaResponse{}

	d.Schema(context.Background(), req, resp)

	assert.NotNil(t, resp.Schema)
	assert.NotEmpty(t, resp.Schema.Description)
	assert.Contains(t, resp.Schema.Attributes, "id")
	assert.Contains(t, resp.Schema.Attributes, "name")
	assert.Contains(t, resp.Schema.Attributes, "description")
	assert.Contains(t, resp.Schema.Attributes, "provider_name")
	assert.Contains(t, resp.Schema.Attributes, "url")
	assert.Contains(t, resp.Schema.Attributes, "created_at")
	assert.Contains(t, resp.Schema.Attributes, "updated_at")
}

func TestProxyCacheDataSource_Read(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	mockSvc := new(mocks.ProxyCachesService)
	mockSvc.On("Get", ctx, "proxy-123").Return(&crSDK.GetProxyCacheResponse{
		ProxyCache: crSDK.ProxyCache{
			ID:        "proxy-123",
			Name:      "my-proxy",
			Provider:  "dockerhub",
			URL:       "https://registry-1.docker.io",
			CreatedAt: "2024-01-01T00:00:00Z",
			UpdatedAt: "2024-01-02T00:00:00Z",
		},
		Description: "My proxy cache",
	}, nil)

	d := &ProxyCacheDataSource{
		proxyCacheService: mockSvc,
	}

	schemaResp := testutils.GetDataSourceTestSchema(t, d)

	configRaw := tftypes.NewValue(
		tftypes.Object{
			AttributeTypes: map[string]tftypes.Type{
				"id":            tftypes.String,
				"name":          tftypes.String,
				"description":   tftypes.String,
				"provider_name": tftypes.String,
				"url":           tftypes.String,
				"created_at":    tftypes.String,
				"updated_at":    tftypes.String,
			},
		},
		map[string]tftypes.Value{
			"id":            tftypes.NewValue(tftypes.String, "proxy-123"),
			"name":          tftypes.NewValue(tftypes.String, ""),
			"description":   tftypes.NewValue(tftypes.String, ""),
			"provider_name": tftypes.NewValue(tftypes.String, ""),
			"url":           tftypes.NewValue(tftypes.String, ""),
			"created_at":    tftypes.NewValue(tftypes.String, ""),
			"updated_at":    tftypes.NewValue(tftypes.String, ""),
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

	var finalState proxyCacheDataSourceModel
	resp.State.Get(ctx, &finalState)

	assert.Equal(t, "proxy-123", finalState.ID.ValueString())
	assert.Equal(t, "my-proxy", finalState.Name.ValueString())
	assert.Equal(t, "My proxy cache", finalState.Description.ValueString())
	assert.Equal(t, "dockerhub", finalState.ProviderName.ValueString())
	assert.Equal(t, "https://registry-1.docker.io", finalState.URL.ValueString())
	assert.Equal(t, "2024-01-01T00:00:00Z", finalState.CreatedAt.ValueString())
	assert.Equal(t, "2024-01-02T00:00:00Z", finalState.UpdatedAt.ValueString())

	mockSvc.AssertExpectations(t)
}

func TestProxyCacheDataSource_Read_APIError(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	mockSvc := new(mocks.ProxyCachesService)
	mockSvc.On("Get", ctx, "proxy-123").Return((*crSDK.GetProxyCacheResponse)(nil), errors.New("api connection error"))

	d := &ProxyCacheDataSource{
		proxyCacheService: mockSvc,
	}

	schemaResp := testutils.GetDataSourceTestSchema(t, d)

	configRaw := tftypes.NewValue(
		tftypes.Object{
			AttributeTypes: map[string]tftypes.Type{
				"id":            tftypes.String,
				"name":          tftypes.String,
				"description":   tftypes.String,
				"provider_name": tftypes.String,
				"url":           tftypes.String,
				"created_at":    tftypes.String,
				"updated_at":    tftypes.String,
			},
		},
		map[string]tftypes.Value{
			"id":            tftypes.NewValue(tftypes.String, "proxy-123"),
			"name":          tftypes.NewValue(tftypes.String, ""),
			"description":   tftypes.NewValue(tftypes.String, ""),
			"provider_name": tftypes.NewValue(tftypes.String, ""),
			"url":           tftypes.NewValue(tftypes.String, ""),
			"created_at":    tftypes.NewValue(tftypes.String, ""),
			"updated_at":    tftypes.NewValue(tftypes.String, ""),
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
