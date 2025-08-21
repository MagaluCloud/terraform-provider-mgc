package network

import (
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

func GetDataSources() []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewDataSourceNetworkNatGateway,
		NewDataSourceNetworkPublicIP,
		NewDataSourceNetworkPublicIPs,
		NewDataSourceNetworkSecurityGroup,
		NewDataSourceNetworkSecurityGroups,
		NewDataSourceNetworkSubnetpool,
		NewDataSourceNetworkSubnetpools,
		NewDataSourceNetworkVPC,
		NewDataSourceNetworkVPCs,
		NewDataSourceNetworkVPCInterface,
		NewDataSourceNetworkVPCInterfaces,
		NewDataSourceNetworkVpcsSubnet,
	}
}

func GetResources() []func() resource.Resource {
	return []func() resource.Resource{
		NewNetworkNatGatewayResource,
		NewNetworkPublicIPResource,
		NewNetworkPublicIPAttachResource,
		NewNetworkSecurityGroupsResource,
		NewNetworkSecurityGroupsAttachResource,
		NewNetworkSecurityGroupsRulesResource,
		NewNetworkSubnetpoolsResource,
		// NewNetworkSubnetPoolsBookCIDRResource, waiting for API to implement Get/List
		NewNetworkVPCResource,
		NewNetworkVPCInterfaceResource,
		NewNetworkVpcsSubnetsResource,
	}
}
