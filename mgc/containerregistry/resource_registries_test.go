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

func TestContainerRegistryResource_Metadata(t *testing.T) {
	r := NewContainerRegistryRegistriesResource()
	req := resource.MetadataRequest{
		ProviderTypeName: "mgc",
	}
	resp := &resource.MetadataResponse{}

	r.Metadata(context.Background(), req, resp)

	assert.Equal(t, "mgc_container_registries", resp.TypeName)
}

func TestContainerRegistryResource_Schema(t *testing.T) {
	r := NewContainerRegistryRegistriesResource()
	req := resource.SchemaRequest{}
	resp := &resource.SchemaResponse{}

	r.Schema(context.Background(), req, resp)

	assert.NotNil(t, resp.Schema)
	assert.NotEmpty(t, resp.Schema.Description)
	assert.Contains(t, resp.Schema.Attributes, "id")
	assert.Contains(t, resp.Schema.Attributes, "name")
	assert.Contains(t, resp.Schema.Attributes, "proxy_cache_id")
}

func TestContainerRegistryResource_Create(t *testing.T) {
	ctx := context.Background()

	mockSvc := new(mocks.RegistriesService)
	mockSvc.On("Create", ctx, &crSDK.RegistryRequest{
		Name:         "my-registry",
		ProxyCacheID: strPtr("proxy-123"),
	}).Return(&crSDK.RegistryResponse{
		ID:           "reg-123",
		Name:         "my-registry",
		ProxyCacheID: strPtr("proxy-123"),
	}, nil)

	r := &ContainerRegistryResource{
		registryService: mockSvc,
	}

	schemaResp := testutils.GetResourceTestSchema(t, r)

	plan := tfsdk.Plan{Schema: schemaResp.Schema}
	inputData := ContainerRegistryModel{
		Id:           types.StringUnknown(),
		Name:         types.StringValue("my-registry"),
		ProxyCacheID: types.StringValue("proxy-123"),
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

	var finalState ContainerRegistryModel
	resp.State.Get(ctx, &finalState)

	assert.Equal(t, "reg-123", finalState.Id.ValueString())
	assert.Equal(t, "my-registry", finalState.Name.ValueString())
	assert.Equal(t, "proxy-123", finalState.ProxyCacheID.ValueString())
	mockSvc.AssertExpectations(t)
}

func TestContainerRegistryResource_Create_WithoutProxyCache(t *testing.T) {
	ctx := context.Background()

	mockSvc := new(mocks.RegistriesService)
	mockSvc.On("Create", ctx, &crSDK.RegistryRequest{
		Name:         "my-registry",
		ProxyCacheID: (*string)(nil),
	}).Return(&crSDK.RegistryResponse{
		ID:           "reg-456",
		Name:         "my-registry",
		ProxyCacheID: nil,
	}, nil)

	r := &ContainerRegistryResource{
		registryService: mockSvc,
	}

	schemaResp := testutils.GetResourceTestSchema(t, r)

	plan := tfsdk.Plan{Schema: schemaResp.Schema}
	inputData := ContainerRegistryModel{
		Id:           types.StringUnknown(),
		Name:         types.StringValue("my-registry"),
		ProxyCacheID: types.StringNull(),
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

	var finalState ContainerRegistryModel
	resp.State.Get(ctx, &finalState)

	assert.Equal(t, "reg-456", finalState.Id.ValueString())
	assert.Equal(t, "my-registry", finalState.Name.ValueString())
	assert.True(t, finalState.ProxyCacheID.IsNull())
	mockSvc.AssertExpectations(t)
}

func TestContainerRegistryResource_Create_APIError(t *testing.T) {
	ctx := context.Background()

	mockSvc := new(mocks.RegistriesService)
	mockSvc.On("Create", ctx, &crSDK.RegistryRequest{
		Name:         "my-registry",
		ProxyCacheID: strPtr("proxy-123"),
	}).Return((*crSDK.RegistryResponse)(nil), errors.New("internal server error"))

	r := &ContainerRegistryResource{
		registryService: mockSvc,
	}

	schemaResp := testutils.GetResourceTestSchema(t, r)

	plan := tfsdk.Plan{Schema: schemaResp.Schema}
	inputData := ContainerRegistryModel{
		Id:           types.StringUnknown(),
		Name:         types.StringValue("my-registry"),
		ProxyCacheID: types.StringValue("proxy-123"),
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

func TestContainerRegistryResource_Read(t *testing.T) {
	ctx := context.Background()

	mockSvc := new(mocks.RegistriesService)
	mockSvc.On("Get", ctx, "reg-123").Return(&crSDK.RegistryResponse{
		ID:           "reg-123",
		Name:         "my-registry",
		ProxyCacheID: strPtr("proxy-123"),
	}, nil)

	r := &ContainerRegistryResource{
		registryService: mockSvc,
	}

	schemaResp := testutils.GetResourceTestSchema(t, r)

	state := tfsdk.State{Schema: schemaResp.Schema}
	inputData := ContainerRegistryModel{
		Id:           types.StringValue("reg-123"),
		Name:         types.StringValue("my-registry"),
		ProxyCacheID: types.StringValue("proxy-123"),
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

	var finalState ContainerRegistryModel
	resp.State.Get(ctx, &finalState)

	assert.Equal(t, "reg-123", finalState.Id.ValueString())
	assert.Equal(t, "my-registry", finalState.Name.ValueString())
	assert.Equal(t, "proxy-123", finalState.ProxyCacheID.ValueString())
	mockSvc.AssertExpectations(t)
}

func TestContainerRegistryResource_Read_APIError(t *testing.T) {
	ctx := context.Background()

	mockSvc := new(mocks.RegistriesService)
	mockSvc.On("Get", ctx, "reg-123").Return((*crSDK.RegistryResponse)(nil), errors.New("not found"))

	r := &ContainerRegistryResource{
		registryService: mockSvc,
	}

	schemaResp := testutils.GetResourceTestSchema(t, r)

	state := tfsdk.State{Schema: schemaResp.Schema}
	inputData := ContainerRegistryModel{
		Id:           types.StringValue("reg-123"),
		Name:         types.StringValue("my-registry"),
		ProxyCacheID: types.StringValue("proxy-123"),
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

func TestContainerRegistryResource_Update(t *testing.T) {
	ctx := context.Background()
	r := &ContainerRegistryResource{}

	req := resource.UpdateRequest{}
	resp := &resource.UpdateResponse{}

	r.Update(ctx, req, resp)

	assert.True(t, resp.Diagnostics.HasError(), "Update operation should return error since it is not supported")
	assert.Contains(t, resp.Diagnostics.Errors()[0].Detail(), "cannot be updated")
}

func TestContainerRegistryResource_Delete(t *testing.T) {
	ctx := context.Background()

	mockSvc := new(mocks.RegistriesService)
	mockSvc.On("Delete", ctx, "reg-123").Return(nil)

	r := &ContainerRegistryResource{
		registryService: mockSvc,
	}

	schemaResp := testutils.GetResourceTestSchema(t, r)

	state := tfsdk.State{Schema: schemaResp.Schema}
	inputData := ContainerRegistryModel{
		Id:           types.StringValue("reg-123"),
		Name:         types.StringValue("my-registry"),
		ProxyCacheID: types.StringValue("proxy-123"),
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

func TestContainerRegistryResource_Delete_APIError(t *testing.T) {
	ctx := context.Background()

	mockSvc := new(mocks.RegistriesService)
	mockSvc.On("Delete", ctx, "reg-123").Return(errors.New("delete error"))

	r := &ContainerRegistryResource{
		registryService: mockSvc,
	}

	schemaResp := testutils.GetResourceTestSchema(t, r)

	state := tfsdk.State{Schema: schemaResp.Schema}
	inputData := ContainerRegistryModel{
		Id:           types.StringValue("reg-123"),
		Name:         types.StringValue("my-registry"),
		ProxyCacheID: types.StringValue("proxy-123"),
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

func TestContainerRegistryResource_ImportState(t *testing.T) {
	ctx := context.Background()
	r := &ContainerRegistryResource{}

	schemaResp := testutils.GetResourceTestSchema(t, r)

	state := tfsdk.State{Schema: schemaResp.Schema}
	_ = state.Set(ctx, &ContainerRegistryModel{
		Id:           types.StringUnknown(),
		Name:         types.StringUnknown(),
		ProxyCacheID: types.StringUnknown(),
	})

	req := resource.ImportStateRequest{
		ID: "import-reg-123",
	}
	resp := &resource.ImportStateResponse{
		State: state,
	}

	r.ImportState(ctx, req, resp)

	assert.False(t, resp.Diagnostics.HasError(), "ImportState should not return errors: %v", resp.Diagnostics)

	var finalState ContainerRegistryModel
	resp.State.Get(ctx, &finalState)
	assert.Equal(t, "import-reg-123", finalState.Id.ValueString())
}
