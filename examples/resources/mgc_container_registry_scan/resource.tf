resource "mgc_container_registry_scan" "scan" {
  registry_id   = mgc_container_registries.registry.id
  repository_id = "afd1f6d8-1313-42da-9248-b8959400ab14"
  digest_or_tag = "sha256:6b3de2e6b4ccfc5fae404042cb1a025b1de13c73458e50455e3143bf12e98eae"
}
