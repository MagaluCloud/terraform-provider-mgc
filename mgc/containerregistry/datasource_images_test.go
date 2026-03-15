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
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/stretchr/testify/assert"
)

func TestDataSourceCRImages_Metadata(t *testing.T) {
	d := NewDataSourceCRImages()
	req := datasource.MetadataRequest{
		ProviderTypeName: "mgc",
	}
	resp := &datasource.MetadataResponse{}

	d.Metadata(context.Background(), req, resp)

	assert.Equal(t, "mgc_container_images", resp.TypeName)
}

func TestDataSourceCRImages_Schema(t *testing.T) {
	d := NewDataSourceCRImages()
	req := datasource.SchemaRequest{}
	resp := &datasource.SchemaResponse{}

	d.Schema(context.Background(), req, resp)

	assert.NotNil(t, resp.Schema)
	assert.NotEmpty(t, resp.Schema.Description)
	assert.Contains(t, resp.Schema.Attributes, "registry_id")
	assert.Contains(t, resp.Schema.Attributes, "repository_name")
	assert.Contains(t, resp.Schema.Attributes, "images")
}

func TestDataSourceCRImages_Read(t *testing.T) {
	ctx := context.Background()

	mockSvc := new(mocks.ImagesService)
	mockSvc.On("ListAll", ctx, "reg-123", "my-repo", crSDK.ImageFilterOptions{}).Return([]crSDK.ImageResponse{
		{
			Digest:    "sha256:123abc",
			SizeBytes: 1048576,
			PulledAt:  "2024-01-01T00:00:00Z",
			PushedAt:  "2024-01-02T00:00:00Z",
			Tags:      []string{"latest", "v1.0"},
		},
		{
			Digest:    "sha256:456def",
			SizeBytes: 2097152,
			PulledAt:  "2024-02-01T00:00:00Z",
			PushedAt:  "2024-02-02T00:00:00Z",
			Tags:      []string{"v2.0"},
		},
	}, nil)

	d := &DataSourceCRImages{
		crImages: mockSvc,
	}

	schemaResp := testutils.GetDataSourceTestSchema(t, d)

	imagesType := tftypes.Object{
		AttributeTypes: map[string]tftypes.Type{
			"digest":     tftypes.String,
			"size_bytes": tftypes.Number,
			"pullet_at":  tftypes.String,
			"pushed_at":  tftypes.String,
			"tags":       tftypes.List{ElementType: tftypes.String},
		},
	}
	listType := tftypes.List{ElementType: imagesType}

	configRaw := tftypes.NewValue(
		tftypes.Object{
			AttributeTypes: map[string]tftypes.Type{
				"registry_id":     tftypes.String,
				"repository_name": tftypes.String,
				"images":          listType,
			},
		},
		map[string]tftypes.Value{
			"registry_id":     tftypes.NewValue(tftypes.String, "reg-123"),
			"repository_name": tftypes.NewValue(tftypes.String, "my-repo"),
			"images":          tftypes.NewValue(listType, []tftypes.Value{}),
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

	var finalState crImagesList
	resp.State.Get(ctx, &finalState)

	assert.Equal(t, "reg-123", finalState.RegistryID.ValueString())
	assert.Equal(t, "my-repo", finalState.RepositoryName.ValueString())
	assert.Len(t, finalState.Images, 2)
	assert.Equal(t, "sha256:123abc", finalState.Images[0].Digest.ValueString())
	assert.Equal(t, int64(1048576), finalState.Images[0].SizeBytes.ValueInt64())
	assert.Equal(t, "2024-01-01T00:00:00Z", finalState.Images[0].PulledAt.ValueString())
	assert.Equal(t, "2024-01-02T00:00:00Z", finalState.Images[0].PushedAt.ValueString())
	assert.Len(t, finalState.Images[0].Tags, 2)
	assert.Equal(t, "latest", finalState.Images[0].Tags[0].ValueString())
	assert.Equal(t, "v1.0", finalState.Images[0].Tags[1].ValueString())

	assert.Equal(t, "sha256:456def", finalState.Images[1].Digest.ValueString())
	assert.Len(t, finalState.Images[1].Tags, 1)

	mockSvc.AssertExpectations(t)
}

func TestDataSourceCRImages_Read_EmptyList(t *testing.T) {
	ctx := context.Background()

	mockSvc := new(mocks.ImagesService)
	mockSvc.On("ListAll", ctx, "reg-123", "my-repo", crSDK.ImageFilterOptions{}).Return([]crSDK.ImageResponse{}, nil)

	d := &DataSourceCRImages{
		crImages: mockSvc,
	}

	schemaResp := testutils.GetDataSourceTestSchema(t, d)

	imagesType := tftypes.Object{
		AttributeTypes: map[string]tftypes.Type{
			"digest":     tftypes.String,
			"size_bytes": tftypes.Number,
			"pullet_at":  tftypes.String,
			"pushed_at":  tftypes.String,
			"tags":       tftypes.List{ElementType: tftypes.String},
		},
	}
	listType := tftypes.List{ElementType: imagesType}

	configRaw := tftypes.NewValue(
		tftypes.Object{
			AttributeTypes: map[string]tftypes.Type{
				"registry_id":     tftypes.String,
				"repository_name": tftypes.String,
				"images":          listType,
			},
		},
		map[string]tftypes.Value{
			"registry_id":     tftypes.NewValue(tftypes.String, "reg-123"),
			"repository_name": tftypes.NewValue(tftypes.String, "my-repo"),
			"images":          tftypes.NewValue(listType, []tftypes.Value{}),
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

	var finalState crImagesList
	resp.State.Get(ctx, &finalState)

	assert.Equal(t, "reg-123", finalState.RegistryID.ValueString())
	assert.Equal(t, "my-repo", finalState.RepositoryName.ValueString())
	assert.Len(t, finalState.Images, 0)
	mockSvc.AssertExpectations(t)
}

func TestDataSourceCRImages_Read_APIError(t *testing.T) {
	ctx := context.Background()

	mockSvc := new(mocks.ImagesService)
	mockSvc.On("ListAll", ctx, "reg-123", "my-repo", crSDK.ImageFilterOptions{}).Return([]crSDK.ImageResponse(nil), errors.New("api connection error"))

	d := &DataSourceCRImages{
		crImages: mockSvc,
	}

	schemaResp := testutils.GetDataSourceTestSchema(t, d)

	imagesType := tftypes.Object{
		AttributeTypes: map[string]tftypes.Type{
			"digest":     tftypes.String,
			"size_bytes": tftypes.Number,
			"pullet_at":  tftypes.String,
			"pushed_at":  tftypes.String,
			"tags":       tftypes.List{ElementType: tftypes.String},
		},
	}
	listType := tftypes.List{ElementType: imagesType}

	configRaw := tftypes.NewValue(
		tftypes.Object{
			AttributeTypes: map[string]tftypes.Type{
				"registry_id":     tftypes.String,
				"repository_name": tftypes.String,
				"images":          listType,
			},
		},
		map[string]tftypes.Value{
			"registry_id":     tftypes.NewValue(tftypes.String, "reg-123"),
			"repository_name": tftypes.NewValue(tftypes.String, "my-repo"),
			"images":          tftypes.NewValue(listType, []tftypes.Value{}),
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

func TestDataSourceCRImages_Model(t *testing.T) {
	image := crImage{
		Digest:    types.StringValue("sha256:test"),
		SizeBytes: types.Int64Value(1024),
		PulledAt:  types.StringValue("2024-01-01T00:00:00Z"),
		PushedAt:  types.StringValue("2024-01-02T00:00:00Z"),
		Tags:      []types.String{types.StringValue("latest")},
	}

	assert.Equal(t, "sha256:test", image.Digest.ValueString())
	assert.Equal(t, int64(1024), image.SizeBytes.ValueInt64())
	assert.Equal(t, "2024-01-01T00:00:00Z", image.PulledAt.ValueString())
	assert.Equal(t, "2024-01-02T00:00:00Z", image.PushedAt.ValueString())
	assert.Len(t, image.Tags, 1)
}
