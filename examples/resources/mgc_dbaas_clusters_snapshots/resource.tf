# Create a snapshot for a DBaaS cluster
resource "mgc_dbaas_clusters_snapshots" "example" {
  cluster_id  = mgc_dbaas_clusters.my_cluster.id
  name        = "example-snapshot"
  description = "Snapshot created via Terraform"
}
