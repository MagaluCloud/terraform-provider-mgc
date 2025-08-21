data "mgc_lbaas_network_listeners" "lbs_network_listeners" {
  lb_id = mgc_lbaas_network.basic_http_lb.id
}

output "lbs_network_listeners" {
  value = data.mgc_lbaas_network_listeners.lbs_network_listeners
}
