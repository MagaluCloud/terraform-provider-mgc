data "mgc_lbaas_network" "lbs" {
}

output "basic_http_lb_output" {
  value = data.mgc_lbaas_networks.lbs
}
