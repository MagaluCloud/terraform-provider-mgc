package containerregistry

import (
	"context"
	"errors"
	"testing"

	crSDK "github.com/MagaluCloud/mgc-sdk-go/containerregistry"
	"github.com/MagaluCloud/terraform-provider-mgc/mgc/internal/mocks"
	"github.com/MagaluCloud/terraform-provider-mgc/mgc/internal/testutils"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
)

func TestProxyCacheResource_Metadata(t *testing.T) {
	r := NewContainerRegistryProxyCacheResource()
	req := resource.MetadataRequest{
		ProviderTypeName: "mgc",
	}
	resp := &resource.MetadataResponse{}

	r.Metadata(context.Background(), req, resp)

	assert.Equal(t, "mgc_container_registry_proxy_cache", resp.TypeName)
}

func TestProxyCacheResource_Schema(t *testing.T) {
	r := NewContainerRegistryProxyCacheResource()
	req := resource.SchemaRequest{}
	resp := &resource.SchemaResponse{}

	r.Schema(context.Background(), req, resp)

	assert.NotNil(t, resp.Schema)
	assert.NotEmpty(t, resp.Schema.Description)
	assert.Contains(t, resp.Schema.Attributes, "id")
	assert.Contains(t, resp.Schema.Attributes, "name")
	assert.Contains(t, resp.Schema.Attributes, "description")
	assert.Contains(t, resp.Schema.Attributes, "provider_name")
	assert.Contains(t, resp.Schema.Attributes, "url")
	assert.Contains(t, resp.Schema.Attributes, "access_key")
	assert.Contains(t, resp.Schema.Attributes, "access_secret")
	assert.Contains(t, resp.Schema.Attributes, "created_at")
	assert.Contains(t, resp.Schema.Attributes, "updated_at")

	accessKeyAttr := resp.Schema.Attributes["access_key"]
	assert.True(t, accessKeyAttr.IsSensitive())

	accessSecretAttr := resp.Schema.Attributes["access_secret"]
	assert.True(t, accessSecretAttr.IsSensitive())
}

func TestProxyCacheResource_Configure_NilProviderData(t *testing.T) {
	r := &ProxyCacheResource{}

	req := resource.ConfigureRequest{
		ProviderData: nil,
	}
	resp := &resource.ConfigureResponse{}

	r.Configure(context.Background(), req, resp)

	assert.False(t, resp.Diagnostics.HasError())
	assert.Nil(t, r.proxyCacheService)
}

func TestProxyCacheResource_Configure_InvalidProviderData(t *testing.T) {
	r := &ProxyCacheResource{}

	req := resource.ConfigureRequest{
		ProviderData: "invalid-type",
	}
	resp := &resource.ConfigureResponse{}

	r.Configure(context.Background(), req, resp)

	assert.True(t, resp.Diagnostics.HasError())
	assert.Nil(t, r.proxyCacheService)
}

func TestProxyCacheResource_Create(t *testing.T) {
	ctx := context.Background()

	mockSvc := new(mocks.ProxyCachesService)
	mockSvc.On("Create", ctx, crSDK.CreateProxyCacheRequest{
		Name:         "my-proxy",
		Provider:     "dockerhub",
		URL:          "https://registry-1.docker.io",
		Description:  strPtr("My proxy cache"),
		AccessKey:    strPtr("access-123"),
		AccessSecret: strPtr("secret-456"),
	}).Return(&crSDK.CreateProxyCacheResponse{
		ID:   "proxy-123",
		Name: "my-proxy",
	}, nil)

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

	r := &ProxyCacheResource{
		proxyCacheService: mockSvc,
	}

	schemaResp := testutils.GetResourceTestSchema(t, r)

	plan := tfsdk.Plan{Schema: schemaResp.Schema}
	inputData := ProxyCacheModel{
		ID:           types.StringUnknown(),
		Name:         types.StringValue("my-proxy"),
		Description:  types.StringValue("My proxy cache"),
		ProviderName: types.StringValue("dockerhub"),
		URL:          types.StringValue("https://registry-1.docker.io"),
		AccessKey:    types.StringValue("access-123"),
		AccessSecret: types.StringValue("secret-456"),
	}
	diags := plan.Set(ctx, &inputData)
	assert.False(t, diags.HasError())

	req := resource.CreateRequest{
		Plan: plan,
	}

	resp := &resource.CreateResponse{
		State: tfsdk.State{Schema: schemaResp.Schema},
	}

	r.Create(ctx, req, resp)

	assert.False(t, resp.Diagnostics.HasError(), "Create operation returned diagnostics errors: %v", resp.Diagnostics)

	var finalState ProxyCacheModel
	resp.State.Get(ctx, &finalState)

	assert.Equal(t, "proxy-123", finalState.ID.ValueString())
	assert.Equal(t, "my-proxy", finalState.Name.ValueString())
	assert.Equal(t, "My proxy cache", finalState.Description.ValueString())
	assert.Equal(t, "2024-01-01T00:00:00Z", finalState.CreatedAt.ValueString())
	assert.Equal(t, "2024-01-02T00:00:00Z", finalState.UpdatedAt.ValueString())
	mockSvc.AssertExpectations(t)
}

func TestProxyCacheResource_Create_APIError(t *testing.T) {
	ctx := context.Background()

	mockSvc := new(mocks.ProxyCachesService)
	mockSvc.On("Create", ctx, crSDK.CreateProxyCacheRequest{
		Name:     "my-proxy",
		Provider: "dockerhub",
		URL:      "https://registry-1.docker.io",
	}).Return((*crSDK.CreateProxyCacheResponse)(nil), errors.New("internal server error"))

	r := &ProxyCacheResource{
		proxyCacheService: mockSvc,
	}

	schemaResp := testutils.GetResourceTestSchema(t, r)

	plan := tfsdk.Plan{Schema: schemaResp.Schema}
	inputData := ProxyCacheModel{
		ID:           types.StringUnknown(),
		Name:         types.StringValue("my-proxy"),
		ProviderName: types.StringValue("dockerhub"),
		URL:          types.StringValue("https://registry-1.docker.io"),
	}
	_ = plan.Set(ctx, &inputData)

	req := resource.CreateRequest{
		Plan: plan,
	}

	resp := &resource.CreateResponse{
		State: tfsdk.State{Schema: schemaResp.Schema},
	}

	r.Create(ctx, req, resp)

	assert.True(t, resp.Diagnostics.HasError(), "Create operation should return diagnostics errors on API failure")
	mockSvc.AssertExpectations(t)
}

func TestProxyCacheResource_Read(t *testing.T) {
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

	r := &ProxyCacheResource{
		proxyCacheService: mockSvc,
	}

	schemaResp := testutils.GetResourceTestSchema(t, r)

	state := tfsdk.State{Schema: schemaResp.Schema}
	inputData := ProxyCacheModel{
		ID:           types.StringValue("proxy-123"),
		Name:         types.StringValue("my-proxy"),
		Description:  types.StringValue("My proxy cache"),
		ProviderName: types.StringValue("dockerhub"),
		URL:          types.StringValue("https://registry-1.docker.io"),
		CreatedAt:    types.StringValue("2024-01-01T00:00:00Z"),
		UpdatedAt:    types.StringValue("2024-01-02T00:00:00Z"),
	}
	diags := state.Set(ctx, &inputData)
	assert.False(t, diags.HasError())

	req := resource.ReadRequest{
		State: state,
	}

	resp := &resource.ReadResponse{
		State: tfsdk.State{Schema: schemaResp.Schema},
	}

	r.Read(ctx, req, resp)

	assert.False(t, resp.Diagnostics.HasError(), "Read operation returned diagnostics errors")

	var finalState ProxyCacheModel
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

func TestProxyCacheResource_Read_APIError(t *testing.T) {
	ctx := context.Background()

	mockSvc := new(mocks.ProxyCachesService)
	mockSvc.On("Get", ctx, "proxy-123").Return((*crSDK.GetProxyCacheResponse)(nil), errors.New("not found"))

	r := &ProxyCacheResource{
		proxyCacheService: mockSvc,
	}

	schemaResp := testutils.GetResourceTestSchema(t, r)

	state := tfsdk.State{Schema: schemaResp.Schema}
	inputData := ProxyCacheModel{
		ID:           types.StringValue("proxy-123"),
		Name:         types.StringValue("my-proxy"),
		ProviderName: types.StringValue("dockerhub"),
		URL:          types.StringValue("https://registry-1.docker.io"),
	}
	_ = state.Set(ctx, &inputData)

	req := resource.ReadRequest{
		State: state,
	}

	resp := &resource.ReadResponse{
		State: tfsdk.State{Schema: schemaResp.Schema},
	}

	r.Read(ctx, req, resp)

	assert.True(t, resp.Diagnostics.HasError(), "Read operation should return diagnostics errors on API failure")
	mockSvc.AssertExpectations(t)
}

func TestProxyCacheResource_Update(t *testing.T) {
	ctx := context.Background()

	mockSvc := new(mocks.ProxyCachesService)
	mockSvc.On("Update", ctx, "proxy-123", crSDK.UpdateProxyCacheRequest{
		Name:        strPtr("my-proxy-updated"),
		Description: strPtr("Updated description"),
		URL:         strPtr("https://registry-2.docker.io"),
	}).Return(&crSDK.ProxyCache{
		ID:        "proxy-123",
		Name:      "my-proxy-updated",
		Provider:  "dockerhub",
		URL:       "https://registry-2.docker.io",
		CreatedAt: "2024-01-01T00:00:00Z",
		UpdatedAt: "2024-03-01T00:00:00Z",
	}, nil)

	r := &ProxyCacheResource{
		proxyCacheService: mockSvc,
	}

	schemaResp := testutils.GetResourceTestSchema(t, r)

	plan := tfsdk.Plan{Schema: schemaResp.Schema}
	planData := ProxyCacheModel{
		ID:           types.StringValue("proxy-123"),
		Name:         types.StringValue("my-proxy-updated"),
		Description:  types.StringValue("Updated description"),
		ProviderName: types.StringValue("dockerhub"),
		URL:          types.StringValue("https://registry-2.docker.io"),
		CreatedAt:    types.StringValue("2024-01-01T00:00:00Z"),
		UpdatedAt:    types.StringValue("2024-01-02T00:00:00Z"),
	}
	diags := plan.Set(ctx, &planData)
	assert.False(t, diags.HasError())

	state := tfsdk.State{Schema: schemaResp.Schema}
	stateData := ProxyCacheModel{
		ID:           types.StringValue("proxy-123"),
		Name:         types.StringValue("my-proxy"),
		Description:  types.StringValue("My proxy cache"),
		ProviderName: types.StringValue("dockerhub"),
		URL:          types.StringValue("https://registry-1.docker.io"),
		CreatedAt:    types.StringValue("2024-01-01T00:00:00Z"),
		UpdatedAt:    types.StringValue("2024-01-02T00:00:00Z"),
	}
	diags = state.Set(ctx, &stateData)
	assert.False(t, diags.HasError())

	req := resource.UpdateRequest{
		Plan:  plan,
		State: state,
	}

	resp := &resource.UpdateResponse{
		State: tfsdk.State{Schema: schemaResp.Schema},
	}

	r.Update(ctx, req, resp)

	assert.False(t, resp.Diagnostics.HasError(), "Update operation returned diagnostics errors: %v", resp.Diagnostics)

	var finalState ProxyCacheModel
	resp.State.Get(ctx, &finalState)

	assert.Equal(t, "proxy-123", finalState.ID.ValueString())
	assert.Equal(t, "my-proxy-updated", finalState.Name.ValueString())
	assert.Equal(t, "Updated description", finalState.Description.ValueString())
	assert.Equal(t, "https://registry-2.docker.io", finalState.URL.ValueString())
	assert.Equal(t, "2024-03-01T00:00:00Z", finalState.UpdatedAt.ValueString())
	mockSvc.AssertExpectations(t)
}

func TestProxyCacheResource_Update_OnlyName(t *testing.T) {
	ctx := context.Background()

	mockSvc := new(mocks.ProxyCachesService)
	mockSvc.On("Update", ctx, "proxy-123", crSDK.UpdateProxyCacheRequest{
		Name: strPtr("my-proxy-renamed"),
	}).Return(&crSDK.ProxyCache{
		ID:        "proxy-123",
		Name:      "my-proxy-renamed",
		Provider:  "dockerhub",
		URL:       "https://registry-1.docker.io",
		CreatedAt: "2024-01-01T00:00:00Z",
		UpdatedAt: "2024-03-01T00:00:00Z",
	}, nil)

	r := &ProxyCacheResource{
		proxyCacheService: mockSvc,
	}

	schemaResp := testutils.GetResourceTestSchema(t, r)

	plan := tfsdk.Plan{Schema: schemaResp.Schema}
	planData := ProxyCacheModel{
		ID:           types.StringValue("proxy-123"),
		Name:         types.StringValue("my-proxy-renamed"),
		Description:  types.StringValue("My proxy cache"),
		ProviderName: types.StringValue("dockerhub"),
		URL:          types.StringValue("https://registry-1.docker.io"),
		CreatedAt:    types.StringValue("2024-01-01T00:00:00Z"),
		UpdatedAt:    types.StringValue("2024-01-02T00:00:00Z"),
	}
	diags := plan.Set(ctx, &planData)
	assert.False(t, diags.HasError())

	state := tfsdk.State{Schema: schemaResp.Schema}
	stateData := ProxyCacheModel{
		ID:           types.StringValue("proxy-123"),
		Name:         types.StringValue("my-proxy"),
		Description:  types.StringValue("My proxy cache"),
		ProviderName: types.StringValue("dockerhub"),
		URL:          types.StringValue("https://registry-1.docker.io"),
		CreatedAt:    types.StringValue("2024-01-01T00:00:00Z"),
		UpdatedAt:    types.StringValue("2024-01-02T00:00:00Z"),
	}
	diags = state.Set(ctx, &stateData)
	assert.False(t, diags.HasError())

	req := resource.UpdateRequest{
		Plan:  plan,
		State: state,
	}

	resp := &resource.UpdateResponse{
		State: tfsdk.State{Schema: schemaResp.Schema},
	}

	r.Update(ctx, req, resp)

	assert.False(t, resp.Diagnostics.HasError(), "Update operation returned diagnostics errors: %v", resp.Diagnostics)

	var finalState ProxyCacheModel
	resp.State.Get(ctx, &finalState)

	assert.Equal(t, "my-proxy-renamed", finalState.Name.ValueString())
	assert.Equal(t, "My proxy cache", finalState.Description.ValueString())
	assert.Equal(t, "https://registry-1.docker.io", finalState.URL.ValueString())
	assert.Equal(t, "2024-03-01T00:00:00Z", finalState.UpdatedAt.ValueString())
	mockSvc.AssertExpectations(t)
}

func TestProxyCacheResource_Update_OnlyDescription(t *testing.T) {
	ctx := context.Background()

	mockSvc := new(mocks.ProxyCachesService)
	mockSvc.On("Update", ctx, "proxy-123", crSDK.UpdateProxyCacheRequest{
		Description: strPtr("New description"),
	}).Return(&crSDK.ProxyCache{
		ID:        "proxy-123",
		Name:      "my-proxy",
		Provider:  "dockerhub",
		URL:       "https://registry-1.docker.io",
		CreatedAt: "2024-01-01T00:00:00Z",
		UpdatedAt: "2024-03-01T00:00:00Z",
	}, nil)

	r := &ProxyCacheResource{
		proxyCacheService: mockSvc,
	}

	schemaResp := testutils.GetResourceTestSchema(t, r)

	plan := tfsdk.Plan{Schema: schemaResp.Schema}
	planData := ProxyCacheModel{
		ID:           types.StringValue("proxy-123"),
		Name:         types.StringValue("my-proxy"),
		Description:  types.StringValue("New description"),
		ProviderName: types.StringValue("dockerhub"),
		URL:          types.StringValue("https://registry-1.docker.io"),
		CreatedAt:    types.StringValue("2024-01-01T00:00:00Z"),
		UpdatedAt:    types.StringValue("2024-01-02T00:00:00Z"),
	}
	diags := plan.Set(ctx, &planData)
	assert.False(t, diags.HasError())

	state := tfsdk.State{Schema: schemaResp.Schema}
	stateData := ProxyCacheModel{
		ID:           types.StringValue("proxy-123"),
		Name:         types.StringValue("my-proxy"),
		Description:  types.StringValue("Old description"),
		ProviderName: types.StringValue("dockerhub"),
		URL:          types.StringValue("https://registry-1.docker.io"),
		CreatedAt:    types.StringValue("2024-01-01T00:00:00Z"),
		UpdatedAt:    types.StringValue("2024-01-02T00:00:00Z"),
	}
	diags = state.Set(ctx, &stateData)
	assert.False(t, diags.HasError())

	req := resource.UpdateRequest{
		Plan:  plan,
		State: state,
	}

	resp := &resource.UpdateResponse{
		State: tfsdk.State{Schema: schemaResp.Schema},
	}

	r.Update(ctx, req, resp)

	assert.False(t, resp.Diagnostics.HasError(), "Update operation returned diagnostics errors: %v", resp.Diagnostics)

	var finalState ProxyCacheModel
	resp.State.Get(ctx, &finalState)

	assert.Equal(t, "my-proxy", finalState.Name.ValueString())
	assert.Equal(t, "New description", finalState.Description.ValueString())
	assert.Equal(t, "2024-03-01T00:00:00Z", finalState.UpdatedAt.ValueString())
	mockSvc.AssertExpectations(t)
}

func TestProxyCacheResource_Update_OnlyURL(t *testing.T) {
	ctx := context.Background()

	mockSvc := new(mocks.ProxyCachesService)
	mockSvc.On("Update", ctx, "proxy-123", crSDK.UpdateProxyCacheRequest{
		URL: strPtr("https://registry-2.docker.io"),
	}).Return(&crSDK.ProxyCache{
		ID:        "proxy-123",
		Name:      "my-proxy",
		Provider:  "dockerhub",
		URL:       "https://registry-2.docker.io",
		CreatedAt: "2024-01-01T00:00:00Z",
		UpdatedAt: "2024-03-01T00:00:00Z",
	}, nil)

	r := &ProxyCacheResource{
		proxyCacheService: mockSvc,
	}

	schemaResp := testutils.GetResourceTestSchema(t, r)

	plan := tfsdk.Plan{Schema: schemaResp.Schema}
	planData := ProxyCacheModel{
		ID:           types.StringValue("proxy-123"),
		Name:         types.StringValue("my-proxy"),
		Description:  types.StringValue("My proxy cache"),
		ProviderName: types.StringValue("dockerhub"),
		URL:          types.StringValue("https://registry-2.docker.io"),
		CreatedAt:    types.StringValue("2024-01-01T00:00:00Z"),
		UpdatedAt:    types.StringValue("2024-01-02T00:00:00Z"),
	}
	diags := plan.Set(ctx, &planData)
	assert.False(t, diags.HasError())

	state := tfsdk.State{Schema: schemaResp.Schema}
	stateData := ProxyCacheModel{
		ID:           types.StringValue("proxy-123"),
		Name:         types.StringValue("my-proxy"),
		Description:  types.StringValue("My proxy cache"),
		ProviderName: types.StringValue("dockerhub"),
		URL:          types.StringValue("https://registry-1.docker.io"),
		CreatedAt:    types.StringValue("2024-01-01T00:00:00Z"),
		UpdatedAt:    types.StringValue("2024-01-02T00:00:00Z"),
	}
	diags = state.Set(ctx, &stateData)
	assert.False(t, diags.HasError())

	req := resource.UpdateRequest{
		Plan:  plan,
		State: state,
	}

	resp := &resource.UpdateResponse{
		State: tfsdk.State{Schema: schemaResp.Schema},
	}

	r.Update(ctx, req, resp)

	assert.False(t, resp.Diagnostics.HasError(), "Update operation returned diagnostics errors: %v", resp.Diagnostics)

	var finalState ProxyCacheModel
	resp.State.Get(ctx, &finalState)

	assert.Equal(t, "my-proxy", finalState.Name.ValueString())
	assert.Equal(t, "https://registry-2.docker.io", finalState.URL.ValueString())
	assert.Equal(t, "2024-03-01T00:00:00Z", finalState.UpdatedAt.ValueString())
	mockSvc.AssertExpectations(t)
}

func TestProxyCacheResource_Update_NoChanges(t *testing.T) {
	ctx := context.Background()

	mockSvc := new(mocks.ProxyCachesService)
	// No Update call expected since there are no changes

	r := &ProxyCacheResource{
		proxyCacheService: mockSvc,
	}

	schemaResp := testutils.GetResourceTestSchema(t, r)

	plan := tfsdk.Plan{Schema: schemaResp.Schema}
	planData := ProxyCacheModel{
		ID:           types.StringValue("proxy-123"),
		Name:         types.StringValue("my-proxy"),
		Description:  types.StringValue("My proxy cache"),
		ProviderName: types.StringValue("dockerhub"),
		URL:          types.StringValue("https://registry-1.docker.io"),
		CreatedAt:    types.StringValue("2024-01-01T00:00:00Z"),
		UpdatedAt:    types.StringValue("2024-01-02T00:00:00Z"),
	}
	diags := plan.Set(ctx, &planData)
	assert.False(t, diags.HasError())

	state := tfsdk.State{Schema: schemaResp.Schema}
	stateData := ProxyCacheModel{
		ID:           types.StringValue("proxy-123"),
		Name:         types.StringValue("my-proxy"),
		Description:  types.StringValue("My proxy cache"),
		ProviderName: types.StringValue("dockerhub"),
		URL:          types.StringValue("https://registry-1.docker.io"),
		CreatedAt:    types.StringValue("2024-01-01T00:00:00Z"),
		UpdatedAt:    types.StringValue("2024-01-02T00:00:00Z"),
	}
	diags = state.Set(ctx, &stateData)
	assert.False(t, diags.HasError())

	req := resource.UpdateRequest{
		Plan:  plan,
		State: state,
	}

	resp := &resource.UpdateResponse{
		State: tfsdk.State{Schema: schemaResp.Schema},
	}

	r.Update(ctx, req, resp)

	assert.False(t, resp.Diagnostics.HasError())

	var finalState ProxyCacheModel
	resp.State.Get(ctx, &finalState)

	assert.Equal(t, "proxy-123", finalState.ID.ValueString())
	assert.Equal(t, "my-proxy", finalState.Name.ValueString())
}

func TestProxyCacheResource_Update_APIError(t *testing.T) {
	ctx := context.Background()

	mockSvc := new(mocks.ProxyCachesService)
	mockSvc.On("Update", ctx, "proxy-123", crSDK.UpdateProxyCacheRequest{
		Name: strPtr("my-proxy-updated"),
	}).Return((*crSDK.ProxyCache)(nil), errors.New("update error"))

	r := &ProxyCacheResource{
		proxyCacheService: mockSvc,
	}

	schemaResp := testutils.GetResourceTestSchema(t, r)

	plan := tfsdk.Plan{Schema: schemaResp.Schema}
	planData := ProxyCacheModel{
		ID:           types.StringValue("proxy-123"),
		Name:         types.StringValue("my-proxy-updated"),
		ProviderName: types.StringValue("dockerhub"),
		URL:          types.StringValue("https://registry-1.docker.io"),
	}
	_ = plan.Set(ctx, &planData)

	state := tfsdk.State{Schema: schemaResp.Schema}
	stateData := ProxyCacheModel{
		ID:           types.StringValue("proxy-123"),
		Name:         types.StringValue("my-proxy"),
		ProviderName: types.StringValue("dockerhub"),
		URL:          types.StringValue("https://registry-1.docker.io"),
	}
	_ = state.Set(ctx, &stateData)

	req := resource.UpdateRequest{
		Plan:  plan,
		State: state,
	}

	resp := &resource.UpdateResponse{
		State: tfsdk.State{Schema: schemaResp.Schema},
	}

	r.Update(ctx, req, resp)

	assert.True(t, resp.Diagnostics.HasError(), "Update operation should return diagnostics errors on API failure")
	mockSvc.AssertExpectations(t)
}

func TestProxyCacheResource_Delete(t *testing.T) {
	ctx := context.Background()

	mockSvc := new(mocks.ProxyCachesService)
	mockSvc.On("Delete", ctx, "proxy-123").Return(nil)

	r := &ProxyCacheResource{
		proxyCacheService: mockSvc,
	}

	schemaResp := testutils.GetResourceTestSchema(t, r)

	state := tfsdk.State{Schema: schemaResp.Schema}
	inputData := ProxyCacheModel{
		ID:           types.StringValue("proxy-123"),
		Name:         types.StringValue("my-proxy"),
		ProviderName: types.StringValue("dockerhub"),
		URL:          types.StringValue("https://registry-1.docker.io"),
	}
	diags := state.Set(ctx, &inputData)
	assert.False(t, diags.HasError())

	req := resource.DeleteRequest{
		State: state,
	}

	resp := &resource.DeleteResponse{
		State: tfsdk.State{Schema: schemaResp.Schema},
	}

	r.Delete(ctx, req, resp)

	assert.False(t, resp.Diagnostics.HasError(), "Delete operation returned diagnostics errors")
	mockSvc.AssertExpectations(t)
}

func TestProxyCacheResource_Delete_APIError(t *testing.T) {
	ctx := context.Background()

	mockSvc := new(mocks.ProxyCachesService)
	mockSvc.On("Delete", ctx, "proxy-123").Return(errors.New("delete error"))

	r := &ProxyCacheResource{
		proxyCacheService: mockSvc,
	}

	schemaResp := testutils.GetResourceTestSchema(t, r)

	state := tfsdk.State{Schema: schemaResp.Schema}
	inputData := ProxyCacheModel{
		ID:           types.StringValue("proxy-123"),
		Name:         types.StringValue("my-proxy"),
		ProviderName: types.StringValue("dockerhub"),
		URL:          types.StringValue("https://registry-1.docker.io"),
	}
	_ = state.Set(ctx, &inputData)

	req := resource.DeleteRequest{
		State: state,
	}

	resp := &resource.DeleteResponse{
		State: tfsdk.State{Schema: schemaResp.Schema},
	}

	r.Delete(ctx, req, resp)

	assert.True(t, resp.Diagnostics.HasError(), "Delete operation should return diagnostics errors on API failure")
	mockSvc.AssertExpectations(t)
}

func TestProxyCacheResource_ImportState(t *testing.T) {
	ctx := context.Background()
	r := &ProxyCacheResource{}

	schemaResp := testutils.GetResourceTestSchema(t, r)

	state := tfsdk.State{Schema: schemaResp.Schema}
	_ = state.Set(ctx, &ProxyCacheModel{
		ID: types.StringUnknown(),
	})

	req := resource.ImportStateRequest{
		ID: "import-proxy-123",
	}
	resp := &resource.ImportStateResponse{
		State: state,
	}

	r.ImportState(ctx, req, resp)

	assert.False(t, resp.Diagnostics.HasError(), "ImportState should not return errors: %v", resp.Diagnostics)

	var finalState ProxyCacheModel
	resp.State.Get(ctx, &finalState)
	assert.Equal(t, "import-proxy-123", finalState.ID.ValueString())
}
