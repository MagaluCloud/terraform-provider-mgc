data "mgc_container_registry_scans" "scans" {
  registry_id = mgc_container_registries.registry.id
  repository_id = data.mgc_container_repository.repository.id
  digest_or_tag = "sha256:6b3de2e6b4ccfc5fae404042cb1a025b1de13c73458e50455e3143bf12e98eae"
}

output "scans" {
  value = data.mgc_container_registry_scans.scans
}