data "mgc_lbaas_network_certificates" "lbs_network_certificates" {
  lb_id = mgc_lbaas_network.basic_http_lb.id
}

output "lbs_network_certificates" {
  value = data.mgc_lbaas_network_certificates.lbs_network_certificates
}
