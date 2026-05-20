data "mgc_container_registry_member" "member" {
  registry_id = mgc_container_registries.registry.id
  member_id   = mgc_container_registry_member.member.id
}

output "member" {
  value = data.mgc_container_registry_member.member
}