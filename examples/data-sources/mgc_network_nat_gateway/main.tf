terraform {
  required_providers {
    mgc = {
      source = "MagaluCloud/mgc"
    }
  }
}

data "mgc_network_nat_gateway" "example" {
  id = "nat-gateway-123456"
}

output "nat_gateway_name" {
  value = data.mgc_network_nat_gateway.example.name
}

output "nat_gateway_description" {
  value = data.mgc_network_nat_gateway.example.description
}

output "nat_gateway_vpc_id" {
  value = data.mgc_network_nat_gateway.example.vpc_id
}

output "nat_gateway_zone" {
  value = data.mgc_network_nat_gateway.example.zone
} 