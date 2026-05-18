data "mgc_network_vpcs_routes" "example" {
  vpc_id = mgc_network_vpcs_route.example.vpc_id
}

output "routes_example" {
  value = data.mgc_network_vpcs_routes.example
}
