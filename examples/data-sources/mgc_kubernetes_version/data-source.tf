data "mgc_kubernetes_version" "cluster_version" {
  include_deprecated = false
}

output "cluster_version_output" {
  value = data.mgc_kubernetes_version.cluster_version
}
