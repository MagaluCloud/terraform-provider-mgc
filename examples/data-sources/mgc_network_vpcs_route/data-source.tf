data "mgc_network_vpcs_route" "example" {
  id     = mgc_network_vpcs_route.example.id
  vpc_id = mgc_network_vpcs_route.example.vpc_id
}

output "route_example" {
  value = data.mgc_network_vpcs_route.example
}
