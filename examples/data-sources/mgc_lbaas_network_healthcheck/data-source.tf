data "mgc_lbaas_network_healthcheck" "lbs_network_healthcheck" {
  lb_id = mgc_lbaas_network.basic_http_lb.id
  id    = mgc_lbaas_network.basic_http_lb.health_checks[0].id
}

output "lbs_network_healthcheck" {
  value = data.mgc_lbaas_network_healthcheck.lbs_network_healthcheck
}
