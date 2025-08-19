package blockstorage

import (
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

func GetDataSources() []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewDataSourceBSSnapshotDatasource,
		NewDataSourceBSSnapshots,
		NewDataSourceBsVolume,
		NewDataSourceBsVolumeTypes,
		NewDataSourceBsVolumes,
	}
}

func GetResources() []func() resource.Resource {
	return []func() resource.Resource{
		NewBlockStorageSnapshotsResource,
		NewVolumeAttachResource,
		NewBlockStorageVolumesResource,
	}
}
