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

func TestDataSourceCRRepositories_Metadata(t *testing.T) {
	t.Parallel()
	d := NewDataSourceCRRepositories()
	req := datasource.MetadataRequest{
		ProviderTypeName: "mgc",
	}
	resp := &datasource.MetadataResponse{}

	d.Metadata(context.Background(), req, resp)

	assert.Equal(t, "mgc_container_repositories", resp.TypeName)
}

func TestDataSourceCRRepositories_Schema(t *testing.T) {
	t.Parallel()
	d := NewDataSourceCRRepositories()
	req := datasource.SchemaRequest{}
	resp := &datasource.SchemaResponse{}

	d.Schema(context.Background(), req, resp)

	assert.NotNil(t, resp.Schema)
	assert.NotEmpty(t, resp.Schema.Description)
	assert.Contains(t, resp.Schema.Attributes, "registry_id")
	assert.Contains(t, resp.Schema.Attributes, "repositories")
}

func TestDataSourceCRRepositories_Read(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	mockSvc := new(mocks.RepositoriesService)
	mockSvc.On("ListAll", ctx, "reg-123", crSDK.RepositoryFilterOptions{}).Return([]crSDK.RepositoryResponse{
		{
			Name:         "repo-1",
			RegistryName: "my-registry",
			ImageCount:   5,
			CreatedAt:    "2024-01-01T00:00:00Z",
			UpdatedAt:    "2024-01-02T00:00:00Z",
		},
		{
			Name:         "repo-2",
			RegistryName: "my-registry",
			ImageCount:   10,
			CreatedAt:    "2024-02-01T00:00:00Z",
			UpdatedAt:    "2024-02-02T00:00:00Z",
		},
	}, nil)

	d := &DataSourceCRRepositories{
		crRepositories: mockSvc,
	}

	schemaResp := testutils.GetDataSourceTestSchema(t, d)

	repositoriesType := tftypes.Object{
		AttributeTypes: map[string]tftypes.Type{
			"name":          tftypes.String,
			"registry_name": tftypes.String,
			"image_count":   tftypes.Number,
			"created_at":    tftypes.String,
			"updated_at":    tftypes.String,
		},
	}
	listType := tftypes.List{ElementType: repositoriesType}

	configRaw := tftypes.NewValue(
		tftypes.Object{
			AttributeTypes: map[string]tftypes.Type{
				"registry_id":  tftypes.String,
				"repositories": listType,
			},
		},
		map[string]tftypes.Value{
			"registry_id":  tftypes.NewValue(tftypes.String, "reg-123"),
			"repositories": tftypes.NewValue(listType, []tftypes.Value{}),
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

	var finalState crRepositoriesList
	resp.State.Get(ctx, &finalState)

	assert.Equal(t, "reg-123", finalState.RegistryID.ValueString())
	assert.Len(t, finalState.Repositories, 2)
	assert.Equal(t, "repo-1", finalState.Repositories[0].Name.ValueString())
	assert.Equal(t, "my-registry", finalState.Repositories[0].RegistryName.ValueString())
	assert.Equal(t, int64(5), finalState.Repositories[0].ImageCount.ValueInt64())
	assert.Equal(t, "2024-01-01T00:00:00Z", finalState.Repositories[0].CreatedAt.ValueString())
	assert.Equal(t, "2024-01-02T00:00:00Z", finalState.Repositories[0].UpdatedAt.ValueString())

	assert.Equal(t, "repo-2", finalState.Repositories[1].Name.ValueString())
	assert.Equal(t, int64(10), finalState.Repositories[1].ImageCount.ValueInt64())

	mockSvc.AssertExpectations(t)
}

func TestDataSourceCRRepositories_Read_EmptyList(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	mockSvc := new(mocks.RepositoriesService)
	mockSvc.On("ListAll", ctx, "reg-123", crSDK.RepositoryFilterOptions{}).Return([]crSDK.RepositoryResponse{}, nil)

	d := &DataSourceCRRepositories{
		crRepositories: mockSvc,
	}

	schemaResp := testutils.GetDataSourceTestSchema(t, d)

	repositoriesType := tftypes.Object{
		AttributeTypes: map[string]tftypes.Type{
			"name":          tftypes.String,
			"registry_name": tftypes.String,
			"image_count":   tftypes.Number,
			"created_at":    tftypes.String,
			"updated_at":    tftypes.String,
		},
	}
	listType := tftypes.List{ElementType: repositoriesType}

	configRaw := tftypes.NewValue(
		tftypes.Object{
			AttributeTypes: map[string]tftypes.Type{
				"registry_id":  tftypes.String,
				"repositories": listType,
			},
		},
		map[string]tftypes.Value{
			"registry_id":  tftypes.NewValue(tftypes.String, "reg-123"),
			"repositories": tftypes.NewValue(listType, []tftypes.Value{}),
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

	var finalState crRepositoriesList
	resp.State.Get(ctx, &finalState)

	assert.Equal(t, "reg-123", finalState.RegistryID.ValueString())
	assert.Len(t, finalState.Repositories, 0)
	mockSvc.AssertExpectations(t)
}

func TestDataSourceCRRepositories_Read_APIError(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	mockSvc := new(mocks.RepositoriesService)
	mockSvc.On("ListAll", ctx, "reg-123", crSDK.RepositoryFilterOptions{}).Return([]crSDK.RepositoryResponse(nil), errors.New("api connection error"))

	d := &DataSourceCRRepositories{
		crRepositories: mockSvc,
	}

	schemaResp := testutils.GetDataSourceTestSchema(t, d)

	repositoriesType := tftypes.Object{
		AttributeTypes: map[string]tftypes.Type{
			"name":          tftypes.String,
			"registry_name": tftypes.String,
			"image_count":   tftypes.Number,
			"created_at":    tftypes.String,
			"updated_at":    tftypes.String,
		},
	}
	listType := tftypes.List{ElementType: repositoriesType}

	configRaw := tftypes.NewValue(
		tftypes.Object{
			AttributeTypes: map[string]tftypes.Type{
				"registry_id":  tftypes.String,
				"repositories": listType,
			},
		},
		map[string]tftypes.Value{
			"registry_id":  tftypes.NewValue(tftypes.String, "reg-123"),
			"repositories": tftypes.NewValue(listType, []tftypes.Value{}),
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
