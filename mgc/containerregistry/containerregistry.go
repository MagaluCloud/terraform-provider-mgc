package containerregistry

import (
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

func GetDataSources() []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewDataSourceCRCredentials,
		NewDataSourceCRImages,
		NewDataSourceCRRegistries,
		NewDataSourceCRRepositories,
		NewDataSourceCRRepository,
		NewProxyCacheDataSource,
		NewProxyCacheListDataSource,
		NewDataSourceCRUser,
		NewDataSourceCRMember,
		NewDataSourceCRMembers,
		NewDataSourceCRScans,
		NewDataSourceCRScanStatus,
		NewDataSourceCRScanVulnerabilities,
	}
}

func GetResources() []func() resource.Resource {
	return []func() resource.Resource{
		NewContainerRegistryRegistriesResource,
		NewContainerRegistryProxyCacheResource,
		NewContainerRegistryUserResource,
		NewContainerRegistryMemberResource,
		NewContainerRegistryScanResource,
	}
}
