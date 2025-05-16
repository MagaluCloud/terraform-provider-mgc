data "mgc_dbaas_cluster" "specific_test_cluster_pg" {
  id = mgc_dbaas_cluster.test_cluster_with_pg.id
}
