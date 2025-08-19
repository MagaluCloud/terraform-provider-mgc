package kubernetes

import (
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

func GetDataSources() []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewDataSourceKubernetesCluster,
		NewDataSourceKubernetesClusterList,
		NewDataSourceKubernetesClusterKubeConfig,
		NewDataSourceKubernetesFlavor,
		NewDataSourceKubernetesNode,
		NewDataSourceKubernetesNodepool,
		NewDataSourceKubernetesVersion,
	}
}

func GetResources() []func() resource.Resource {
	return []func() resource.Resource{
		NewK8sClusterResource,
		NewNewNodePoolResource,
	}
}
