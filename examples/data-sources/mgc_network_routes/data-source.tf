data "mgc_network_routes" "example" {
  vpc_id = mgc_network_route.example.vpc_id
}

output "routes_example" {
  value = data.mgc_network_routes.example
}
