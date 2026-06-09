data "mgc_container_registry_scan_vulnerabilities" "vulnerabilities" {
  scan_id = mgc_container_registry_scan.scan.id
}

output "vulnerabilities" {
  value = data.mgc_container_registry_scan_vulnerabilities.vulnerabilities
}