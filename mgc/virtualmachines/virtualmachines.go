package virtualmachines

import (
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

func GetDataSources() []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewDataSourceVMIMages,
		NewDataSourceVmInstance,
		NewDataSourceVmInstances,
		NewDataSourceVmMachineType,
		NewDataSourceVmSnapshots,
	}
}

func GetResources() []func() resource.Resource {
	return []func() resource.Resource{
		NewVirtualMachineInstancesResource,
		NewVirtualMachineInterfaceAttachResource,
		NewVirtualMachineSnapshotsResource,
	}
}
