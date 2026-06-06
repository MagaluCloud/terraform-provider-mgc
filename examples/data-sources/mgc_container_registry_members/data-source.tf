data "mgc_container_registry_members" "members" {
  registry_id = mgc_container_registries.registry.id
}

output "members" {
  value = data.mgc_container_registry_members.members
}