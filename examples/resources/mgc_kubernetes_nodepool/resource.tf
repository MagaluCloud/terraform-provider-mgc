resource "mgc_kubernetes_nodepool" "nodepool" {
  name         = "gandalf"
  cluster_id   = mgc_kubernetes_cluster.cluster_with_nodepool.id
  flavor_name  = kubernetes_flavor.flavors[0].name
  replicas     = 1
  min_replicas = 1
  max_replicas = 5
  version      = mgc_kubernetes_cluster.cluster_with_nodepool.version

  # Optional labels, set only at creation. Changing them recreates the node pool.
  # To remove every label, set `labels = {}`. Deleting the block keeps the labels
  # already in state.
  labels = {
    environment = "staging"
    team        = "devX"
    tier        = "backend"
  }
}
