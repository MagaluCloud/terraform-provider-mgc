package objects

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/stretchr/testify/assert"
)

func TestObjectStorageObjectsDataSource_Metadata(t *testing.T) {
	ds := NewObjectStorageObjectsDataSource()

	ctx := context.Background()
	req := datasource.MetadataRequest{
		ProviderTypeName: "mgc",
	}
	resp := &datasource.MetadataResponse{}

	ds.Metadata(ctx, req, resp)

	assert.Equal(t, "mgc_object_storage_objects", resp.TypeName)
}

func TestObjectStorageObjectsDataSource_Configure_NilProviderData(t *testing.T) {
	ds := &ObjectStorageObjectsDataSource{}
	ctx := context.Background()
	req := datasource.ConfigureRequest{
		ProviderData: nil,
	}
	resp := &datasource.ConfigureResponse{}

	ds.Configure(ctx, req, resp)

	assert.False(t, resp.Diagnostics.HasError())
	assert.Nil(t, ds.objects)
}

func TestObjectStorageObjectsDataSource_Configure_InvalidProviderData(t *testing.T) {
	ds := &ObjectStorageObjectsDataSource{}
	ctx := context.Background()
	req := datasource.ConfigureRequest{
		ProviderData: "invalid",
	}
	resp := &datasource.ConfigureResponse{}

	ds.Configure(ctx, req, resp)

	assert.True(t, resp.Diagnostics.HasError())
	assert.Contains(t, resp.Diagnostics.Errors()[0].Summary(), "Failed to configure provider")
}

func TestObjectStorageObjectsDataSource_Configure_ValidProviderData(t *testing.T) {
	t.Skip("Skipping configure test - requires SDK integration")
}

func TestObjectStorageObjectsDataSource_Schema(t *testing.T) {
	ds := &ObjectStorageObjectsDataSource{}
	ctx := context.Background()
	req := datasource.SchemaRequest{}
	resp := &datasource.SchemaResponse{}

	ds.Schema(ctx, req, resp)

	assert.NotNil(t, resp.Schema)
	assert.NotEmpty(t, resp.Schema.Description)

	attributes := resp.Schema.Attributes
	assert.Contains(t, attributes, "bucket")
	assert.Contains(t, attributes, "objects")

	assert.True(t, attributes["bucket"].IsRequired())
	assert.False(t, attributes["bucket"].IsComputed())

	assert.True(t, attributes["objects"].IsComputed())
}

func TestNewObjectStorageObjectsDataSource(t *testing.T) {
	ds := NewObjectStorageObjectsDataSource()
	assert.NotNil(t, ds)
	assert.IsType(t, &ObjectStorageObjectsDataSource{}, ds)
}
