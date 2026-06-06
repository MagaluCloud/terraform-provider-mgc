package objects

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/stretchr/testify/assert"
)

func TestObjectStorageObjectDataSource_Metadata(t *testing.T) {
	ds := NewObjectStorageObjectDataSource()

	ctx := context.Background()
	req := datasource.MetadataRequest{
		ProviderTypeName: "mgc",
	}
	resp := &datasource.MetadataResponse{}

	ds.Metadata(ctx, req, resp)

	assert.Equal(t, "mgc_object_storage_object", resp.TypeName)
}

func TestObjectStorageObjectDataSource_Configure_NilProviderData(t *testing.T) {
	ds := &ObjectStorageObjectDataSource{}
	ctx := context.Background()
	req := datasource.ConfigureRequest{
		ProviderData: nil,
	}
	resp := &datasource.ConfigureResponse{}

	ds.Configure(ctx, req, resp)

	assert.False(t, resp.Diagnostics.HasError())
	assert.Nil(t, ds.objects)
}

func TestObjectStorageObjectDataSource_Configure_InvalidProviderData(t *testing.T) {
	ds := &ObjectStorageObjectDataSource{}
	ctx := context.Background()
	req := datasource.ConfigureRequest{
		ProviderData: "invalid",
	}
	resp := &datasource.ConfigureResponse{}

	ds.Configure(ctx, req, resp)

	assert.True(t, resp.Diagnostics.HasError())
	assert.Contains(t, resp.Diagnostics.Errors()[0].Summary(), "Failed to configure provider")
}

func TestObjectStorageObjectDataSource_Configure_ValidProviderData(t *testing.T) {
	t.Skip("Skipping configure test - requires SDK integration")
}

func TestObjectStorageObjectDataSource_Schema(t *testing.T) {
	ds := &ObjectStorageObjectDataSource{}
	ctx := context.Background()
	req := datasource.SchemaRequest{}
	resp := &datasource.SchemaResponse{}

	ds.Schema(ctx, req, resp)

	assert.NotNil(t, resp.Schema)
	assert.NotEmpty(t, resp.Schema.Description)

	attributes := resp.Schema.Attributes
	assert.Contains(t, attributes, "bucket")
	assert.Contains(t, attributes, "key")
	assert.Contains(t, attributes, "content_type")
	assert.Contains(t, attributes, "etag")
	assert.Contains(t, attributes, "size")
	assert.Contains(t, attributes, "last_modified")
	assert.Contains(t, attributes, "lock_status")

	assert.True(t, attributes["bucket"].IsRequired())
	assert.False(t, attributes["bucket"].IsComputed())

	assert.True(t, attributes["key"].IsRequired())
	assert.False(t, attributes["key"].IsComputed())

	assert.True(t, attributes["content_type"].IsComputed())
	assert.True(t, attributes["etag"].IsComputed())
	assert.True(t, attributes["size"].IsComputed())
	assert.True(t, attributes["last_modified"].IsComputed())
	assert.True(t, attributes["lock_status"].IsComputed())

	lockStatusAttr, ok := attributes["lock_status"].(schema.BoolAttribute)
	assert.True(t, ok)
	assert.True(t, lockStatusAttr.Computed)
}

func TestNewObjectStorageObjectDataSource(t *testing.T) {
	ds := NewObjectStorageObjectDataSource()
	assert.NotNil(t, ds)
	assert.IsType(t, &ObjectStorageObjectDataSource{}, ds)
}
