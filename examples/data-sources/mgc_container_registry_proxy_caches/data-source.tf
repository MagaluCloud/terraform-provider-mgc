data "mgc_container_registry_proxy_caches" "proxy_list" {}

output "proxy_list" {
  value = data.mgc_container_registry_proxy_caches.proxy_list.proxy_caches
}
