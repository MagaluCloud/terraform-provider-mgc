data "mgc_kubernetes_clusters" "clusterlist"{
}

output "myclusters" {
  value = data.mgc_kubernetes_clusters.clusterlist
}