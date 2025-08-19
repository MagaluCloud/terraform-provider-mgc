package objects

import (
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

func GetDataSources() []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewDatasourceBucket,
		NewDatasourceBuckets,
	}
}

func GetResources() []func() resource.Resource {
	return []func() resource.Resource{
		NewObjectStorageBucketsResource,
	}
}
