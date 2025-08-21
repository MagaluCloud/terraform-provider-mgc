data "mgc_lbaas_network_certificate" "lbs_network_certificate" {
  lb_id = mgc_lbaas_network.basic_http_lb.id
  id    = mgc_lbaas_network.basic_http_lb.tls_certificates[0].id
}

output "lbs_network_certificate" {
  value = data.mgc_lbaas_network_certificate.lbs_network_certificate
}
