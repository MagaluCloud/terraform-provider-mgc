package ssh

import (
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

func GetDataSources() []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewDataSourceSSH,
	}
}

func GetResources() []func() resource.Resource {
	return []func() resource.Resource{
		NewSshKeysResource,
	}
}
