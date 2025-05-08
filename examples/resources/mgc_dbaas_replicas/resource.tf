resource "mgc_dbaas_replicas" "dbaas_replica" {
  name      = "dbaas-read-replica"
  source_id = "source-id"
}
