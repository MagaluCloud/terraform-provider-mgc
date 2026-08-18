resource "mgc_network_vpcs_peering" "example" {
  name             = "peering-prod-to-db"
  description      = "Peering between the production VPC and the database VPC"
  requester_vpc_id = "your-requester-vpc-id"
  accepter_vpc_id  = "your-accepter-vpc-id"
}
