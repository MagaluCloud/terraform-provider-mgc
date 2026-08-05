package tags

import (
	clientSDK "github.com/MagaluCloud/mgc-sdk-go/client"
	tagSDK "github.com/MagaluCloud/mgc-sdk-go/tag"
	"github.com/MagaluCloud/terraform-provider-mgc/mgc/utils"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

func GetResources() []func() resource.Resource {
	return []func() resource.Resource{
		NewTagResource,
		NewTagValueResource,
		NewTagAttachmentResource,
	}
}

func GetDataSources() []func() datasource.DataSource {
	return []func() datasource.DataSource{}
}

// newTagClient builds the tags client. Tags are a global service, and the client
// overrides the base URL of the core client with the global one, so a custom
// endpoint has to be passed through the option instead of through CoreFor.
func newTagClient(dataConfig utils.DataConfig) *tagSDK.TagClient {
	var opts []tagSDK.ClientOption
	if endpoint, ok := dataConfig.EndpointFor(utils.ServiceTags); ok {
		opts = append(opts, tagSDK.WithGlobalBasePath(clientSDK.MgcUrl(endpoint)))
	}

	return tagSDK.New(dataConfig.CoreFor(utils.ServiceTags), opts...)
}
