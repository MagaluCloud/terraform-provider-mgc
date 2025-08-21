data "mgc_lbaas_network_backend" "lbs_network_backend" {
  lb_id = mgc_lbaas_network.basic_http_lb.id
  id    = mgc_lbaas_network.basic_http_lb.backends[0].id
}

output "lbs_network_backend" {
  value = data.mgc_lbaas_network_backend.lbs_network_backend
}
