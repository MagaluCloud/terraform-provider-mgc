# A route points to exactly one target: either a port or a VPC peering.

# Route through a port.
resource "mgc_network_vpcs_route" "through_port" {
  vpc_id           = "your-vpc-id"
  port_id          = "your-port-id"
  cidr_destination = "xxx.xxx.xxx.xxx/xx"
  description      = "Route example"
}

# Route through a VPC peering.
resource "mgc_network_vpcs_peering" "peering" {
  name             = "peering_name"
  description      = "peering_description"
  requester_vpc_id = mgc_network_vpcs.requester.id
  accepter_vpc_id  = mgc_network_vpcs.accepter.id
}

resource "mgc_network_vpcs_route" "route_r" {
  vpc_id           = mgc_network_vpcs.requester.id
  peering_id       = mgc_network_vpcs_peering.peering.id
  cidr_destination = mgc_network_vpcs_subnets.subnet_accepter.cidr_block
  description      = "Route vpcs peering requester"
}

resource "mgc_network_vpcs_route" "route_a" {
  vpc_id           = mgc_network_vpcs.accepter.id
  peering_id       = mgc_network_vpcs_peering.peering.id
  cidr_destination = mgc_network_vpcs_subnets.subnet_requester.cidr_block
  description      = "Route vpcs peering accepter"
}
