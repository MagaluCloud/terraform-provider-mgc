data "mgc_lbaas_network_backends" "lbs_network_backends" {
  lb_id = mgc_lbaas_network.basic_http_lb.id
}

output "lbs_network_backends" {
  value = data.mgc_lbaas_network_backends.lbs_network_backends
}
