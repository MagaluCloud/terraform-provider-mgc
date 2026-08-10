# A route points to exactly one target: either a port or a VPC peering.

# Route through a port.
resource "mgc_network_vpcs_route" "through_port" {
  vpc_id           = "your-vpc-id"
  port_id          = "your-port-id"
  cidr_destination = "xxx.xxx.xxx.xxx/xx"
  description      = "Route example"
}

# Route through a VPC peering.
resource "mgc_network_vpcs_peering" "example" {
  name             = "peering-example"
  requester_vpc_id = "your-requester-vpc-id"
  accepter_vpc_id  = "your-accepter-vpc-id"
}

resource "mgc_network_vpcs_route" "through_peering" {
  vpc_id           = mgc_network_vpcs_peering.example.requester_vpc_id
  vpc_peering_id   = mgc_network_vpcs_peering.example.id
  cidr_destination = "xxx.xxx.xxx.xxx/xx"
  description      = "Route to the peered VPC"
}
