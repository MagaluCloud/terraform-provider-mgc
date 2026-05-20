resource "mgc_container_registry_member" "member" {
  registry_id = mgc_container_registries.registry.id
  user_id     = mgc_container_registry_user.user.id
  role        = "developer"
}
