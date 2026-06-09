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

func TestDataSourceCRCredentials_Metadata(t *testing.T) {
	t.Parallel()
	d := NewDataSourceCRCredentials()
	req := datasource.MetadataRequest{
		ProviderTypeName: "mgc",
	}
	resp := &datasource.MetadataResponse{}

	d.Metadata(context.Background(), req, resp)

	assert.Equal(t, "mgc_container_credentials", resp.TypeName)
}

func TestDataSourceCRCredentials_Schema(t *testing.T) {
	t.Parallel()
	d := NewDataSourceCRCredentials()
	req := datasource.SchemaRequest{}
	resp := &datasource.SchemaResponse{}

	d.Schema(context.Background(), req, resp)

	assert.NotNil(t, resp.Schema)
	assert.NotEmpty(t, resp.Schema.Description)
	assert.Contains(t, resp.Schema.Attributes, "email")
	assert.Contains(t, resp.Schema.Attributes, "password")
	assert.Contains(t, resp.Schema.Attributes, "username")

	passwordAttr := resp.Schema.Attributes["password"]
	assert.True(t, passwordAttr.IsSensitive())
}

func TestDataSourceCRCredentials_Configure_NilProviderData(t *testing.T) {
	t.Parallel()
	d := &DataSourceCRCredentials{}

	req := datasource.ConfigureRequest{
		ProviderData: nil,
	}
	resp := &datasource.ConfigureResponse{}

	d.Configure(context.Background(), req, resp)

	assert.False(t, resp.Diagnostics.HasError())
	assert.Nil(t, d.crCredentials)
}

func TestDataSourceCRCredentials_Configure_InvalidProviderData(t *testing.T) {
	t.Parallel()
	d := &DataSourceCRCredentials{}

	req := datasource.ConfigureRequest{
		ProviderData: "invalid-type",
	}
	resp := &datasource.ConfigureResponse{}

	d.Configure(context.Background(), req, resp)

	assert.True(t, resp.Diagnostics.HasError())
	assert.Nil(t, d.crCredentials)
}

func TestDataSourceCRCredentials_Read(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	mockSvc := new(mocks.CredentialsService)
	mockSvc.On("Get", ctx).Return(&crSDK.CredentialsResponse{
		Email:    "test@example.com",
		Password: "secret-password-123",
		Username: "testuser",
	}, nil)

	d := &DataSourceCRCredentials{
		crCredentials: mockSvc,
	}

	schemaResp := testutils.GetDataSourceTestSchema(t, d)

	configRaw := tftypes.NewValue(
		tftypes.Object{
			AttributeTypes: map[string]tftypes.Type{
				"email":    tftypes.String,
				"password": tftypes.String,
				"username": tftypes.String,
			},
		},
		map[string]tftypes.Value{
			"email":    tftypes.NewValue(tftypes.String, ""),
			"password": tftypes.NewValue(tftypes.String, ""),
			"username": tftypes.NewValue(tftypes.String, ""),
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

	var finalState crCredentials
	resp.State.Get(ctx, &finalState)

	assert.Equal(t, "test@example.com", finalState.Email.ValueString())
	assert.Equal(t, "secret-password-123", finalState.Password.ValueString())
	assert.Equal(t, "testuser", finalState.Username.ValueString())

	mockSvc.AssertExpectations(t)
}

func TestDataSourceCRCredentials_Read_APIError(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	mockSvc := new(mocks.CredentialsService)
	mockSvc.On("Get", ctx).Return((*crSDK.CredentialsResponse)(nil), errors.New("api connection error"))

	d := &DataSourceCRCredentials{
		crCredentials: mockSvc,
	}

	schemaResp := testutils.GetDataSourceTestSchema(t, d)

	configRaw := tftypes.NewValue(
		tftypes.Object{
			AttributeTypes: map[string]tftypes.Type{
				"email":    tftypes.String,
				"password": tftypes.String,
				"username": tftypes.String,
			},
		},
		map[string]tftypes.Value{
			"email":    tftypes.NewValue(tftypes.String, ""),
			"password": tftypes.NewValue(tftypes.String, ""),
			"username": tftypes.NewValue(tftypes.String, ""),
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
