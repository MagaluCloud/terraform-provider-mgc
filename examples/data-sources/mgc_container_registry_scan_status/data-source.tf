data "mgc_container_registry_scan_status" "scan_status" {
  scan_id = mgc_container_registry_scan.scan.id
}

output "scans_status" {
  value = data.mgc_container_registry_scan_status.scan_status
}