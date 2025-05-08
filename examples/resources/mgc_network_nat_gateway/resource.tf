terraform {
  required_providers {
    mgc = {
      source = "MagaluCloud/mgc"
    }
  }
}

resource "mgc_network_nat_gateway" "my_ngateway" {
  name        = "example-nat-gateway"
  description = "Example NAT Gateway"
  vpc_id      = "vpc-123456"
  zone        = "zone-1"
} 