package database

import (
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

func GetDataSources() []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewDBaaSClusterDataSource,
		NewDBaaSClustersDataSource,
		NewDataSourceDbaasEngines,
		NewDataSourceDbaasInstance,
		NewDataSourceDbaasInstances,
		NewDataSourceDbaasInstancesSnapshot,
		NewDataSourceDbaasInstancesSnapshots,
		NewDataSourceDbaasParameterGroup,
		NewDataSourceDbaasParameterGroups,
		NewDataSourceDbParametersList,
		NewDataSourceDbReplica,
		NewDataSourceDbReplicaList,
		NewDataSourceDbaasInstanceTypes,
	}
}

func GetResources() []func() resource.Resource {
	return []func() resource.Resource{
		NewDBaaSClusterResource,
		NewDBaaSInstanceResource,
		NewDBaaSInstanceSnapshotResource,
		NewDBaaSParameterGroupsResource,
		NewDBaaSParameterResource,
		NewDBaaSReplicaResource,
	}
}
