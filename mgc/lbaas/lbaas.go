package lbaas

import (
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

func GetDataSources() []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewDataSourceLbaasNetwork,
		NewDataSourceLbaasNetworks,
		NewDataSourceLbaasNetworkBackend,
		NewDataSourceLbaasNetworkBackends,
		NewDataSourceLbaasNetworkHealthCheck,
		NewDataSourceLbaasNetworkHealthChecks,
		NewDataSourceLbaasNetworkListener,
		NewDataSourceLbaasNetworkListeners,
		NewDataSourceLbaasNetworkCertificate,
		NewDataSourceLbaasNetworkCertificates,
	}
}

func GetResources() []func() resource.Resource {
	return []func() resource.Resource{
		NewLoadBalancerResource,
	}
}
