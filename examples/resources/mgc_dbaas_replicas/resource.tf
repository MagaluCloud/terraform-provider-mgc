resource "mgc_dbaas_replicas" "dbaas_replica" {
  name              = "dbaas-read-replica"
  source_id         = "source-id"
  availability_zone = "br-se1-a" # optional; defaults to the source instance's availability zone
}
