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

resource "mgc_container_registry_proxy_cache" "proxy_cache" {
  name          = "terraform-proxy-cache"
  description   = "terraform test"
  provider_name = "docker-hub"
  url           = "https://hub.docker.com"
}

data "mgc_container_registry_proxy_cache" "proxy_cache_info" {
  id = mgc_container_registry_proxy_cache.proxy_cache.id
}

data "mgc_container_registry_proxy_caches" "proxy_list" {}
