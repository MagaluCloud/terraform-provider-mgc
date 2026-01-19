resource "mgc_container_registry_proxy_cache" "example" {
  name          = "terraform-proxy-cache"
  description   = "description"
  provider_name = "docker-hub"
  url           = "https://hub.docker.com"
  access_key    = "access_key"
  access_secret = "access_secret"
}
