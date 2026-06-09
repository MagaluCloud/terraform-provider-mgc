package objects

import (
	objSdk "github.com/MagaluCloud/mgc-sdk-go/objectstorage"
	"github.com/MagaluCloud/terraform-provider-mgc/mgc/utils"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

// resolveS3Endpoint returns the S3 endpoint for object storage.
// It uses the custom endpoint from DataConfig when configured,
// otherwise resolves the default regional endpoint.
func resolveS3Endpoint(dataConfig utils.DataConfig) (objSdk.Endpoint, error) {
	if customURL, hasCustom := dataConfig.ObjectStorageEndpoint(); hasCustom {
		return objSdk.Endpoint(customURL), nil
	}
	return utils.RegionToS3Url(dataConfig.Region, dataConfig.Env)
}

func GetDataSources() []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewObjectStorageBucketDataSource,
		NewObjectStorageBucketsDataSource,
		NewObjectStorageObjectDataSource,
		NewObjectStorageObjectsDataSource,
	}
}

func GetResources() []func() resource.Resource {
	return []func() resource.Resource{
		NewObjectStorageBucketsResource,
		NewObjectStorageObjectsResource,
	}
}
