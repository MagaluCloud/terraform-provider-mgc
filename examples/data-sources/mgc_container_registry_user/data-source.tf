data "mgc_container_registry_user" "user" {}

output "user" {
  value = data.mgc_container_registry_user.user
}