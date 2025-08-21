data "mgc_lbaas_network" "basic_http_lb" {
  id = mgc_lbaas_network.basic_http_lb.id
}

output "basic_http_lb_output" {
  value = data.mgc_lbaas_network.basic_http_lb
}
