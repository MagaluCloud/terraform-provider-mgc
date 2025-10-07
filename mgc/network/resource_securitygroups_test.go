package network

import (
	"context"
	"errors"
	"testing"

	clientSDK "github.com/MagaluCloud/mgc-sdk-go/client"
	netSDK "github.com/MagaluCloud/mgc-sdk-go/network"

	"github.com/MagaluCloud/terraform-provider-mgc/mgc/utils"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type mockSecurityGroupService struct {
	mock.Mock
	netSDK.SecurityGroupService
}

func (m *mockSecurityGroupService) Get(ctx context.Context, id string) (*netSDK.SecurityGroupDetailResponse, error) {
	args := m.Called(ctx, id)
	result := args.Get(0)
	if result == nil {
		return nil, args.Error(1)
	}
	return result.(*netSDK.SecurityGroupDetailResponse), args.Error(1)
}

func (m *mockSecurityGroupService) Create(ctx context.Context, req netSDK.SecurityGroupCreateRequest) (string, error) {
	args := m.Called(ctx, req)
	return args.String(0), args.Error(1)
}

func (m *mockSecurityGroupService) Delete(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func TestNewNetworkSecurityGroupsResource(t *testing.T) {
	t.Parallel()

	resource := NewNetworkSecurityGroupsResource()
	require.NotNil(t, resource)
	require.IsType(t, &NetworkSecurityGroupsResource{}, resource)

	resource2 := NewNetworkSecurityGroupsResource()
	assert.NotSame(t, resource, resource2)

	sgResource, ok := resource.(*NetworkSecurityGroupsResource)
	require.True(t, ok)
	require.NotNil(t, sgResource)
}

func TestNetworkSecurityGroupsResource_Metadata(t *testing.T) {
	t.Parallel()

	r := &NetworkSecurityGroupsResource{}
	req := resource.MetadataRequest{
		ProviderTypeName: "mgc",
	}
	resp := &resource.MetadataResponse{}

	r.Metadata(context.Background(), req, resp)

	assert.Equal(t, "mgc_network_security_groups", resp.TypeName)
}

func TestNetworkSecurityGroupsResource_Configure(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		providerData any
		expectError  bool
		errorMessage string
	}{
		{
			name:         "nil provider data",
			providerData: nil,
			expectError:  false,
		},
		{
			name:         "invalid provider data type",
			providerData: "invalid",
			expectError:  true,
			errorMessage: "Failed to get provider data",
		},
		{
			name: "valid provider data",
			providerData: utils.DataConfig{
				ApiKey: "test-key",
				Region: "test-region",
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			r := &NetworkSecurityGroupsResource{}
			req := resource.ConfigureRequest{
				ProviderData: tt.providerData,
			}
			resp := &resource.ConfigureResponse{}

			r.Configure(context.Background(), req, resp)

			if tt.expectError {
				require.True(t, resp.Diagnostics.HasError())
				assert.Contains(t, resp.Diagnostics.Errors()[0].Summary(), tt.errorMessage)
			} else {
				assert.False(t, resp.Diagnostics.HasError())
			}
		})
	}
}

func TestNetworkSecurityGroupsResource_Schema(t *testing.T) {
	t.Parallel()

	r := &NetworkSecurityGroupsResource{}
	req := resource.SchemaRequest{}
	resp := &resource.SchemaResponse{}

	r.Schema(context.Background(), req, resp)

	require.NotNil(t, resp.Schema)
	assert.Equal(t, "Network Security Group", resp.Schema.Description)

	attrs := resp.Schema.Attributes
	require.Contains(t, attrs, "id")
	require.Contains(t, attrs, "name")
	require.Contains(t, attrs, "description")
	require.Contains(t, attrs, "disable_default_rules")

	idAttr, ok := attrs["id"].(schema.StringAttribute)
	require.True(t, ok, "id attribute should be StringAttribute")
	assert.True(t, idAttr.Computed)
	assert.False(t, idAttr.Required)
	assert.False(t, idAttr.Optional)

	nameAttr, ok := attrs["name"].(schema.StringAttribute)
	require.True(t, ok, "name attribute should be StringAttribute")
	assert.True(t, nameAttr.Required)
	assert.False(t, nameAttr.Computed)
	assert.False(t, nameAttr.Optional)

	descAttr, ok := attrs["description"].(schema.StringAttribute)
	require.True(t, ok, "description attribute should be StringAttribute")
	assert.True(t, descAttr.Optional)
	assert.False(t, descAttr.Required)
	assert.False(t, descAttr.Computed)

	disableAttr, ok := attrs["disable_default_rules"].(schema.BoolAttribute)
	require.True(t, ok, "disable_default_rules attribute should be BoolAttribute")
	assert.True(t, disableAttr.Optional)
	assert.True(t, disableAttr.Computed)
	assert.False(t, disableAttr.Required)
}

func TestNetworkSecurityGroupsResource_Read(t *testing.T) {
	tests := []struct {
		name           string
		mockSetup      func(*mockSecurityGroupService)
		inputData      NetworkSecurityGroupsModel
		expectError    bool
		expectedResult NetworkSecurityGroupsModel
	}{
		{
			name: "successful read",
			mockSetup: func(m *mockSecurityGroupService) {
				sg := &netSDK.SecurityGroupDetailResponse{
					SecurityGroupResponse: netSDK.SecurityGroupResponse{
						ID:          ptrTo("sg-123"),
						Name:        ptrTo("test-sg"),
						Description: ptrTo("test description"),
					},
				}
				m.On("Get", mock.Anything, "sg-123").Return(sg, nil)
			},
			inputData: NetworkSecurityGroupsModel{
				Id: types.StringValue("sg-123"),
			},
			expectError: false,
			expectedResult: NetworkSecurityGroupsModel{
				Id:          types.StringValue("sg-123"),
				Name:        types.StringValue("test-sg"),
				Description: types.StringValue("test description"),
			},
		},
		{
			name: "sdk error",
			mockSetup: func(m *mockSecurityGroupService) {
				m.On("Get", mock.Anything, "sg-123").Return(nil, errors.New("sdk error"))
			},
			inputData: NetworkSecurityGroupsModel{
				Id: types.StringValue("sg-123"),
			},
			expectError: true,
		},
		{
			name: "security group with nil values",
			mockSetup: func(m *mockSecurityGroupService) {
				sg := &netSDK.SecurityGroupDetailResponse{
					SecurityGroupResponse: netSDK.SecurityGroupResponse{
						ID:          ptrTo("sg-123"),
						Name:        nil,
						Description: nil,
					},
				}
				m.On("Get", mock.Anything, "sg-123").Return(sg, nil)
			},
			inputData: NetworkSecurityGroupsModel{
				Id: types.StringValue("sg-123"),
			},
			expectError: false,
			expectedResult: NetworkSecurityGroupsModel{
				Id:          types.StringValue("sg-123"),
				Name:        types.StringNull(),
				Description: types.StringNull(),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			r, mockSvc := setupTestResource(t)
			tt.mockSetup(mockSvc)

			state := createTestState(t, tt.inputData)

			req := resource.ReadRequest{
				State: state,
			}
			resp := &resource.ReadResponse{
				State: state,
			}

			r.Read(context.Background(), req, resp)

			if tt.expectError {
				require.True(t, resp.Diagnostics.HasError())
			} else {
				assert.False(t, resp.Diagnostics.HasError())

				var result NetworkSecurityGroupsModel
				resp.State.Get(context.Background(), &result)

				assert.Equal(t, tt.expectedResult.Id, result.Id)
				assert.Equal(t, tt.expectedResult.Name, result.Name)
				assert.Equal(t, tt.expectedResult.Description, result.Description)
			}

			mockSvc.AssertExpectations(t)
		})
	}
}

func TestNetworkSecurityGroupsResource_Create(t *testing.T) {
	tests := []struct {
		name        string
		mockSetup   func(*mockSecurityGroupService)
		inputData   NetworkSecurityGroupsModel
		expectError bool
		expectedID  string
	}{
		{
			name: "successful create",
			mockSetup: func(m *mockSecurityGroupService) {
				expectedReq := netSDK.SecurityGroupCreateRequest{
					Name:             "test-sg",
					Description:      ptrTo("test description"),
					SkipDefaultRules: ptrTo(false),
				}
				m.On("Create", mock.Anything, expectedReq).Return("sg-123", nil)
			},
			inputData: NetworkSecurityGroupsModel{
				Name:                types.StringValue("test-sg"),
				Description:         types.StringValue("test description"),
				DisableDefaultRules: types.BoolValue(false),
			},
			expectError: false,
			expectedID:  "sg-123",
		},
		{
			name: "create with nil description",
			mockSetup: func(m *mockSecurityGroupService) {
				expectedReq := netSDK.SecurityGroupCreateRequest{
					Name:             "test-sg",
					Description:      nil,
					SkipDefaultRules: ptrTo(true),
				}
				m.On("Create", mock.Anything, expectedReq).Return("sg-456", nil)
			},
			inputData: NetworkSecurityGroupsModel{
				Name:                types.StringValue("test-sg"),
				Description:         types.StringNull(),
				DisableDefaultRules: types.BoolValue(true),
			},
			expectError: false,
			expectedID:  "sg-456",
		},
		{
			name: "sdk error",
			mockSetup: func(m *mockSecurityGroupService) {
				m.On("Create", mock.Anything, mock.Anything).Return("", errors.New("sdk error"))
			},
			inputData: NetworkSecurityGroupsModel{
				Name: types.StringValue("test-sg"),
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			r, mockSvc := setupTestResource(t)
			tt.mockSetup(mockSvc)

			plan := createTestPlan(t, tt.inputData)

			req := resource.CreateRequest{
				Plan: plan,
			}
			resp := &resource.CreateResponse{
				State: tfsdk.State{
					Schema: getTestSchema(),
				},
			}

			r.Create(context.Background(), req, resp)

			if tt.expectError {
				require.True(t, resp.Diagnostics.HasError())
			} else {
				assert.False(t, resp.Diagnostics.HasError())

				var result NetworkSecurityGroupsModel
				resp.State.Get(context.Background(), &result)

				assert.Equal(t, tt.expectedID, result.Id.ValueString())
			}

			mockSvc.AssertExpectations(t)
		})
	}
}

func TestNetworkSecurityGroupsResource_Delete(t *testing.T) {
	tests := []struct {
		name        string
		mockSetup   func(*mockSecurityGroupService)
		inputData   NetworkSecurityGroupsModel
		expectError bool
	}{
		{
			name: "successful delete",
			mockSetup: func(m *mockSecurityGroupService) {
				m.On("Delete", mock.Anything, "sg-123").Return(nil)
			},
			inputData: NetworkSecurityGroupsModel{
				Id: types.StringValue("sg-123"),
			},
			expectError: false,
		},
		{
			name: "sdk error",
			mockSetup: func(m *mockSecurityGroupService) {
				m.On("Delete", mock.Anything, "sg-123").Return(errors.New("sdk error"))
			},
			inputData: NetworkSecurityGroupsModel{
				Id: types.StringValue("sg-123"),
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			r, mockSvc := setupTestResource(t)
			tt.mockSetup(mockSvc)

			state := createTestState(t, tt.inputData)

			req := resource.DeleteRequest{
				State: state,
			}
			resp := &resource.DeleteResponse{}

			r.Delete(context.Background(), req, resp)

			if tt.expectError {
				require.True(t, resp.Diagnostics.HasError())
			} else {
				assert.False(t, resp.Diagnostics.HasError())
			}

			mockSvc.AssertExpectations(t)
		})
	}
}

func TestNetworkSecurityGroupsResource_Update(t *testing.T) {
	t.Parallel()

	r := &NetworkSecurityGroupsResource{}
	req := resource.UpdateRequest{}
	resp := &resource.UpdateResponse{}

	r.Update(context.Background(), req, resp)

	require.True(t, resp.Diagnostics.HasError())
	assert.Contains(t, resp.Diagnostics.Errors()[0].Summary(), "Update is not supported for Security Group")
}

func TestNetworkSecurityGroupsResource_ImportState(t *testing.T) {
	t.Parallel()

	r := &NetworkSecurityGroupsResource{}

	testSchema := getTestSchema()

	state := tfsdk.State{
		Schema: testSchema,
	}

	emptyModel := NetworkSecurityGroupsModel{}
	state.Set(context.Background(), &emptyModel)

	req := resource.ImportStateRequest{
		ID: "sg-123",
	}
	resp := &resource.ImportStateResponse{
		State: state,
	}

	r.ImportState(context.Background(), req, resp)

	assert.False(t, resp.Diagnostics.HasError())

	var result NetworkSecurityGroupsModel
	resp.State.Get(context.Background(), &result)

	assert.Equal(t, "sg-123", result.Id.ValueString())
}

func TestNetworkSecurityGroupsModel_Validation(t *testing.T) {
	tests := []struct {
		name  string
		model NetworkSecurityGroupsModel
	}{
		{
			name: "valid model with all fields",
			model: NetworkSecurityGroupsModel{
				Id:                  types.StringValue("sg-123"),
				Name:                types.StringValue("test-sg"),
				Description:         types.StringValue("test description"),
				DisableDefaultRules: types.BoolValue(false),
			},
		},
		{
			name: "valid model with minimal fields",
			model: NetworkSecurityGroupsModel{
				Name: types.StringValue("test-sg"),
			},
		},
		{
			name: "valid model with null values",
			model: NetworkSecurityGroupsModel{
				Id:                  types.StringNull(),
				Name:                types.StringValue("test-sg"),
				Description:         types.StringNull(),
				DisableDefaultRules: types.BoolNull(),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.NotPanics(t, func() {
				_ = tt.model.Id.ValueString()
				_ = tt.model.Name.ValueString()
				_ = tt.model.Description.ValueString()
				_ = tt.model.DisableDefaultRules.ValueBool()
			})
		})
	}
}

// Test error handling with different SDK error types
func TestNetworkSecurityGroupsResource_ErrorHandling(t *testing.T) {
	tests := []struct {
		name      string
		sdkError  error
		operation string
	}{
		{
			name:      "http error",
			sdkError:  &clientSDK.HTTPError{StatusCode: 404},
			operation: "read",
		},
		{
			name:      "validation error",
			sdkError:  &clientSDK.ValidationError{Field: "name", Message: "required"},
			operation: "create",
		},
		{
			name:      "generic error",
			sdkError:  errors.New("generic error"),
			operation: "delete",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockSvc := &mockSecurityGroupService{}

			switch tt.operation {
			case "read":
				mockSvc.On("Get", mock.Anything, mock.Anything).Return(nil, tt.sdkError)
			case "create":
				mockSvc.On("Create", mock.Anything, mock.Anything).Return("", tt.sdkError)
			case "delete":
				mockSvc.On("Delete", mock.Anything, mock.Anything).Return(tt.sdkError)
			}

			r := &NetworkSecurityGroupsResource{
				networkSecurityGroups: mockSvc,
			}

			// Test that errors are handled gracefully
			switch tt.operation {
			case "read":
				state := tfsdk.State{Schema: getTestSchema()}
				state.Set(context.Background(), NetworkSecurityGroupsModel{Id: types.StringValue("sg-123")})

				req := resource.ReadRequest{State: state}
				resp := &resource.ReadResponse{State: state}

				r.Read(context.Background(), req, resp)
				require.True(t, resp.Diagnostics.HasError())

			case "create":
				plan := tfsdk.Plan{Schema: getTestSchema()}
				plan.Set(context.Background(), NetworkSecurityGroupsModel{Name: types.StringValue("test")})

				req := resource.CreateRequest{Plan: plan}
				resp := &resource.CreateResponse{State: tfsdk.State{Schema: getTestSchema()}}

				r.Create(context.Background(), req, resp)
				require.True(t, resp.Diagnostics.HasError())

			case "delete":
				state := tfsdk.State{Schema: getTestSchema()}
				state.Set(context.Background(), NetworkSecurityGroupsModel{Id: types.StringValue("sg-123")})

				req := resource.DeleteRequest{State: state}
				resp := &resource.DeleteResponse{}

				r.Delete(context.Background(), req, resp)
				require.True(t, resp.Diagnostics.HasError())
			}

			mockSvc.AssertExpectations(t)
		})
	}
}

func TestNetworkSecurityGroupsResource_ReadWithEmptyID(t *testing.T) {
	mockSvc := &mockSecurityGroupService{}
	r := &NetworkSecurityGroupsResource{
		networkSecurityGroups: mockSvc,
	}

	// Test with empty ID should not call SDK
	data := NetworkSecurityGroupsModel{
		Id: types.StringValue(""),
	}

	state := tfsdk.State{Schema: getTestSchema()}
	state.Set(context.Background(), data)

	req := resource.ReadRequest{State: state}
	resp := &resource.ReadResponse{State: state}

	// Since empty ID will still be passed to SDK, mock should return error
	mockSvc.On("Get", mock.Anything, "").Return(nil, errors.New("invalid id"))

	r.Read(context.Background(), req, resp)

	assert.True(t, resp.Diagnostics.HasError())
	mockSvc.AssertExpectations(t)
}

func TestNetworkSecurityGroupsResource_CreateWithAllOptionalFields(t *testing.T) {
	r, mockSvc := setupTestResource(t)

	expectedReq := netSDK.SecurityGroupCreateRequest{
		Name:             "test-sg-full",
		Description:      ptrTo("comprehensive test description"),
		SkipDefaultRules: ptrTo(true),
	}
	mockSvc.On("Create", mock.Anything, expectedReq).Return("sg-full-123", nil)

	inputData := NetworkSecurityGroupsModel{
		Name:                types.StringValue("test-sg-full"),
		Description:         types.StringValue("comprehensive test description"),
		DisableDefaultRules: types.BoolValue(true),
	}

	plan := tfsdk.Plan{Schema: getTestSchema()}
	plan.Set(context.Background(), inputData)

	req := resource.CreateRequest{Plan: plan}
	resp := &resource.CreateResponse{
		State: tfsdk.State{Schema: getTestSchema()},
	}

	r.Create(context.Background(), req, resp)

	assert.False(t, resp.Diagnostics.HasError())

	var result NetworkSecurityGroupsModel
	resp.State.Get(context.Background(), &result)

	assert.Equal(t, "sg-full-123", result.Id.ValueString())
	assert.Equal(t, "test-sg-full", result.Name.ValueString())
	assert.Equal(t, "comprehensive test description", result.Description.ValueString())
	assert.True(t, result.DisableDefaultRules.ValueBool())

	mockSvc.AssertExpectations(t)
}

func TestNetworkSecurityGroupsResource_CreateWithEmptyName(t *testing.T) {

	r, mockSvc := setupTestResource(t)

	inputData := NetworkSecurityGroupsModel{
		Name:        types.StringValue(""),
		Description: types.StringValue("test"),
	}

	plan := createTestPlan(t, inputData)

	req := resource.CreateRequest{Plan: plan}
	resp := &resource.CreateResponse{
		State: tfsdk.State{Schema: getTestSchema()},
	}

	expectedReq := netSDK.SecurityGroupCreateRequest{
		Name:             "",
		Description:      ptrTo("test"),
		SkipDefaultRules: (*bool)(nil),
	}
	mockSvc.On("Create", mock.Anything, expectedReq).Return("", errors.New("name cannot be empty"))

	r.Create(context.Background(), req, resp)

	require.True(t, resp.Diagnostics.HasError())
	mockSvc.AssertExpectations(t)
}

func TestNetworkSecurityGroupsResource_DeleteWithEmptyID(t *testing.T) {
	r, mockSvc := setupTestResource(t)

	inputData := NetworkSecurityGroupsModel{
		Id: types.StringValue(""),
	}

	state := createTestState(t, inputData)

	req := resource.DeleteRequest{State: state}
	resp := &resource.DeleteResponse{}

	mockSvc.On("Delete", mock.Anything, "").Return(errors.New("id cannot be empty"))

	r.Delete(context.Background(), req, resp)

	require.True(t, resp.Diagnostics.HasError())
	mockSvc.AssertExpectations(t)
}

func TestNetworkSecurityGroupsModel_NullAndEmptyValues(t *testing.T) {
	tests := []struct {
		name  string
		model NetworkSecurityGroupsModel
	}{
		{
			name: "all null values",
			model: NetworkSecurityGroupsModel{
				Id:                  types.StringNull(),
				Name:                types.StringNull(),
				Description:         types.StringNull(),
				DisableDefaultRules: types.BoolNull(),
			},
		},
		{
			name: "all unknown values",
			model: NetworkSecurityGroupsModel{
				Id:                  types.StringUnknown(),
				Name:                types.StringUnknown(),
				Description:         types.StringUnknown(),
				DisableDefaultRules: types.BoolUnknown(),
			},
		},
		{
			name: "mixed null and value",
			model: NetworkSecurityGroupsModel{
				Id:                  types.StringValue("sg-123"),
				Name:                types.StringNull(),
				Description:         types.StringValue("test"),
				DisableDefaultRules: types.BoolNull(),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.NotPanics(t, func() {
				_ = tt.model.Id.IsNull()
				_ = tt.model.Id.IsUnknown()
				_ = tt.model.Name.IsNull()
				_ = tt.model.Name.IsUnknown()
				_ = tt.model.Description.IsNull()
				_ = tt.model.Description.IsUnknown()
				_ = tt.model.DisableDefaultRules.IsNull()
				_ = tt.model.DisableDefaultRules.IsUnknown()
			})
		})
	}
}

func TestNetworkSecurityGroupsResource_ConcurrentOperations(t *testing.T) {
	t.Parallel()

	// Test that the resource can handle concurrent operations safely
	r, mockSvc := setupTestResource(t)

	sg := &netSDK.SecurityGroupDetailResponse{
		SecurityGroupResponse: netSDK.SecurityGroupResponse{
			ID:          ptrTo("sg-concurrent"),
			Name:        ptrTo("concurrent-sg"),
			Description: ptrTo("concurrent test"),
		},
	}
	mockSvc.On("Get", mock.Anything, "sg-concurrent").Return(sg, nil)

	const numGoroutines = 5
	results := make(chan bool, numGoroutines)

	for range numGoroutines {
		go func() {
			data := NetworkSecurityGroupsModel{
				Id: types.StringValue("sg-concurrent"),
			}

			state := createTestState(t, data)

			req := resource.ReadRequest{State: state}
			resp := &resource.ReadResponse{State: state}

			r.Read(context.Background(), req, resp)
			results <- !resp.Diagnostics.HasError()
		}()
	}

	for range numGoroutines {
		success := <-results
		assert.True(t, success)
	}

	mockSvc.AssertExpectations(t)
}

func BenchmarkNetworkSecurityGroupsResource_Create(b *testing.B) {
	r, mockSvc := setupTestResource(b)

	expectedReq := netSDK.SecurityGroupCreateRequest{
		Name:             "benchmark-sg",
		Description:      ptrTo("benchmark test"),
		SkipDefaultRules: ptrTo(false),
	}

	mockSvc.On("Create", mock.Anything, expectedReq).Return("sg-bench", nil).Maybe()

	inputData := NetworkSecurityGroupsModel{
		Name:                types.StringValue("benchmark-sg"),
		Description:         types.StringValue("benchmark test"),
		DisableDefaultRules: types.BoolValue(false),
	}

	b.ReportAllocs()

	for b.Loop() {
		plan := createTestPlan(b, inputData)
		req := resource.CreateRequest{Plan: plan}
		resp := &resource.CreateResponse{
			State: tfsdk.State{Schema: getTestSchema()},
		}

		r.Create(context.Background(), req, resp)
	}
}

func BenchmarkNetworkSecurityGroupsResource_Read(b *testing.B) {
	r, mockSvc := setupTestResource(b)

	sg := &netSDK.SecurityGroupDetailResponse{
		SecurityGroupResponse: netSDK.SecurityGroupResponse{
			ID:          ptrTo("sg-bench"),
			Name:        ptrTo("benchmark-sg"),
			Description: ptrTo("benchmark test"),
		},
	}

	mockSvc.On("Get", mock.Anything, "sg-bench").Return(sg, nil).Maybe()

	inputData := NetworkSecurityGroupsModel{
		Id: types.StringValue("sg-bench"),
	}

	b.ReportAllocs()

	for b.Loop() {
		state := createTestState(b, inputData)
		req := resource.ReadRequest{State: state}
		resp := &resource.ReadResponse{State: state}

		r.Read(context.Background(), req, resp)
	}
}

func BenchmarkNetworkSecurityGroupsResource_Delete(b *testing.B) {
	r, mockSvc := setupTestResource(b)

	mockSvc.On("Delete", mock.Anything, "sg-bench").Return(nil).Maybe()

	inputData := NetworkSecurityGroupsModel{
		Id: types.StringValue("sg-bench"),
	}

	b.ReportAllocs()

	for b.Loop() {
		state := createTestState(b, inputData)
		req := resource.DeleteRequest{State: state}
		resp := &resource.DeleteResponse{}

		r.Delete(context.Background(), req, resp)
	}
}

func FuzzNetworkSecurityGroupsModel_ValueString(f *testing.F) {

	f.Add("sg-123", "test-sg", "test description", true)
	f.Add("", "", "", false)
	f.Add("sg-very-long-id-with-special-chars", "test-sg-name", "", true)

	f.Fuzz(func(t *testing.T, id, name, description string, disableRules bool) {
		model := NetworkSecurityGroupsModel{
			Id:                  types.StringValue(id),
			Name:                types.StringValue(name),
			Description:         types.StringValue(description),
			DisableDefaultRules: types.BoolValue(disableRules),
		}

		assert.NotPanics(t, func() {
			_ = model.Id.ValueString()
			_ = model.Name.ValueString()
			_ = model.Description.ValueString()
			_ = model.DisableDefaultRules.ValueBool()
		})

		assert.Equal(t, id, model.Id.ValueString())
		assert.Equal(t, name, model.Name.ValueString())
		assert.Equal(t, description, model.Description.ValueString())
		assert.Equal(t, disableRules, model.DisableDefaultRules.ValueBool())
	})
}

func ptrTo[T any](v T) *T {
	return &v
}

func setupTestResource(t testing.TB) (*NetworkSecurityGroupsResource, *mockSecurityGroupService) {
	t.Helper()

	mockSvc := &mockSecurityGroupService{}
	resource := &NetworkSecurityGroupsResource{
		networkSecurityGroups: mockSvc,
	}

	return resource, mockSvc
}

func createTestState(t testing.TB, data NetworkSecurityGroupsModel) tfsdk.State {
	t.Helper()

	state := tfsdk.State{Schema: getTestSchema()}
	diags := state.Set(context.Background(), data)
	require.False(t, diags.HasError(), "Failed to set test state")

	return state
}

func createTestPlan(t testing.TB, data NetworkSecurityGroupsModel) tfsdk.Plan {
	t.Helper()

	plan := tfsdk.Plan{Schema: getTestSchema()}
	diags := plan.Set(context.Background(), data)
	require.False(t, diags.HasError(), "Failed to set test plan")

	return plan
}

func getTestSchema() schema.Schema {
	return schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
			},
			"name": schema.StringAttribute{
				Required: true,
			},
			"description": schema.StringAttribute{
				Optional: true,
			},
			"disable_default_rules": schema.BoolAttribute{
				Optional: true,
				Computed: true,
			},
		},
	}
}
