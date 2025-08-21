data "mgc_lbaas_network_healthchecks" "lbs_network_healthchecks" {
  lb_id = mgc_lbaas_network.basic_http_lb.id
}

output "lbs_network_healthchecks" {
  value = data.mgc_lbaas_network_healthchecks.lbs_network_healthchecks
}
