data "mgc_container_registry_proxy_cache" "proxy_cache" {
  id = "proxy-cache-id"
}

output "proxy_cache" {
  value = {
    id            = data.mgc_container_registry_proxy_cache.proxy_cache.id
    name          = data.mgc_container_registry_proxy_cache.proxy_cache.name
    description   = data.mgc_container_registry_proxy_cache.proxy_cache.description
    provider_name = data.mgc_container_registry_proxy_cache.proxy_cache.provider_name
    url           = data.mgc_container_registry_proxy_cache.proxy_cache.url
    created_at    = data.mgc_container_registry_proxy_cache.proxy_cache.created_at
    updated_at    = data.mgc_container_registry_proxy_cache.proxy_cache.updated_at
  }
}
