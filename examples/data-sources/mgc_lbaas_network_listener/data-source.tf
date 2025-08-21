data "mgc_lbaas_network_listener" "lbs_network_listener" {
  lb_id = mgc_lbaas_network.basic_http_lb.id
  id    = mgc_lbaas_network.basic_http_lb.listeners[0].id
}

output "lbs_network_listener" {
  value = data.mgc_lbaas_network_listener.lbs_network_listener
}
