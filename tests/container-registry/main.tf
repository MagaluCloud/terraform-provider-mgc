resource "mgc_container_registries" "registry" {
  name = "my_registry"
}

data "mgc_container_registries" "registry" {
}

data "mgc_container_repositories" "repository" {
  registry_id = mgc_container_registries.registry.id
}

data "mgc_container_images" "image" {
  registry_id     = mgc_container_registries.registry.id
  repository_name = "alohomora"
}
