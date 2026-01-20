resource "mgc_dbaas_clusters" "my_cluster" {
  name                  = "test-cluster"
  user                  = "clusteradmin"
  password              = "aVerySecureClu$terP@ssw0rd"
  engine_name           = "mysql"
  engine_version        = "8.0"
  instance_type         = var.instance_type_label_cluster
  volume_size           = 100
  volume_type           = "CLOUD_NVME15K"
  backup_retention_days = 7
  backup_start_at       = "03:00:00"
  parameter_group_id    = mgc_dbaas_parameter_groups.cluster_pg.id
  deletion_protected    = true
}

resource "mgc_dbaas_clusters" "my_cluster_no_parameter_group" {
  name                  = "test-cluster-nopg-${random_pet.name.id}"
  user                  = "clusteradmin2"
  password              = "anotherS&cureP@sswordClu1"
  engine_name           = "mysql"
  engine_version        = "8.0"
  instance_type         = var.instance_type_label_cluster
  volume_size           = 50
  backup_retention_days = 5
  backup_start_at       = "02:00:00"
}
