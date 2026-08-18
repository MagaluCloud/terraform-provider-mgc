# All peerings of the current tenant.
data "mgc_network_vpcs_peerings" "all" {}

# Only the peerings a given VPC takes part in.
data "mgc_network_vpcs_peerings" "by_vpc" {
  vpc_id = "your-vpc-id"
}
