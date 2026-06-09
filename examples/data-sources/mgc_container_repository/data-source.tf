data "mgc_container_repository" "repository" {
  registry_id = mgc_container_registries.registry.id
  id          = "794d667e-95c3-4939-b509-72e51195184f"
}

output "repository" {
  value = data.mgc_container_repository.repository
}
