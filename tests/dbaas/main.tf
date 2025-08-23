resource "random_pet" "name" {
  length    = 1
  separator = "-"
}

# ------------------------------
# Common Variables / Data
# ------------------------------

# Active MySQL 8.0 Engine ID from your provided list
variable "mysql_8_0_engine_id" {
  description = "ID for MySQL 8.0 engine."
  type        = string
  default     = "063f3994-b6c2-4c37-96c9-bab8d82d36f7"
}

# Instance Type label for single instances
variable "instance_type_label_single" {
  description = "Instance type label for single DBaaS instances."
  type        = string
  default     = "BV1-4-10"
}

# Instance Type label for cluster nodes
variable "instance_type_label_cluster" {
  description = "Instance type label for DBaaS cluster nodes."
  type        = string
  default     = "DP2-16-40" # Example, adjust if needed
}

variable "availability_zone" {
  description = "Availability zone for resource deployment."
  type        = string
  default     = "br-se1-a"
}

# ------------------------------
# DBaaS Instance Related Resources
# ------------------------------

resource "mgc_dbaas_parameter_groups" "instance_pg" {
  engine_name    = "mysql"
  engine_version = "8.0"
  description    = "Parameter group for test instances"
  name           = "test-instance-pg-${random_pet.name.id}"
}

resource "mgc_dbaas_instances" "test_instance" {
  name                  = "test-instance-${random_pet.name.id}"
  user                  = "dbadmin"
  password              = "aComplexP@ssw0rd!Insa"
  engine_name           = "mysql"
  engine_version        = "8.0"
  instance_type         = var.instance_type_label_single
  volume_size           = 60
  backup_retention_days = 10
  backup_start_at       = "17:00:00"
  availability_zone     = var.availability_zone
  parameter_group       = mgc_dbaas_parameter_groups.instance_pg.id
}

output "dbaas_instance_details" {
  value     = mgc_dbaas_instances.test_instance
  sensitive = true
}

resource "mgc_dbaas_instances_snapshots" "test_snapshot" {
  instance_id = mgc_dbaas_instances.test_instance.id
  name        = "test-snapshot-${random_pet.name.id}"
  description = "Test snapshot for terraform acceptance tests"
}

output "dbaas_snapshot_details" {
  value = mgc_dbaas_instances_snapshots.test_snapshot
}

resource "mgc_dbaas_parameters" "instance_param_max_connections" {
  parameter_group_id = mgc_dbaas_parameter_groups.instance_pg.id
  name               = "MAX_CONNECTIONS" # Common parameter for MySQL
  value              = 20
}

resource "mgc_dbaas_replicas" "dbaas_replica" {
  name      = "test-replica-${random_pet.name.id}"
  source_id = mgc_dbaas_instances.test_instance.id
}

output "dbaas_replica_details" {
  value = mgc_dbaas_replicas.dbaas_replica
}

# ------------------------------
# DBaaS Cluster Related Resources
# ------------------------------

resource "mgc_dbaas_parameter_groups" "cluster_pg" {
  engine_name    = "mysql"
  engine_version = "8.0"
  description    = "Parameter group for test clusters"
  name           = "test-cluster-pg-${random_pet.name.id}"
}

# resource "mgc_dbaas_clusters" "test_cluster_with_pg" {
#   name                  = "test-cluster-pg-${random_pet.name.id}"
#   user                  = "clusteradmin"
#   password              = "aVerySecureClu$terP@ssw0rd"
#   engine_name           = "mysql"
#   engine_version        = "8.0"
#   instance_type         = var.instance_type_label_cluster
#   volume_size           = 100
#   volume_type           = "CLOUD_NVME15K"
#   backup_retention_days = 7
#   backup_start_at       = "03:00:00"
#   parameter_group_id    = mgc_dbaas_parameter_groups.cluster_pg.id
# }

# resource "mgc_dbaas_clusters" "test_cluster_no_pg" {
#   name                  = "test-cluster-nopg-${random_pet.name.id}"
#   user                  = "clusteradmin2"
#   password              = "anotherS&cureP@sswordClu1"
#   engine_name           = "mysql"
#   engine_version        = "8.0"
#   instance_type         = var.instance_type_label_cluster
#   volume_size           = 50
#   backup_retention_days = 5
#   backup_start_at       = "02:00:00"
# }

# output "dbaas_cluster_with_pg_details" {
#   value     = mgc_dbaas_clusters.test_cluster_with_pg
#   sensitive = true
# }

# output "dbaas_cluster_no_pg_details" {
#   value     = mgc_dbaas_clusters.test_cluster_no_pg
#   sensitive = true
# }


# ------------------------------
# Data Sources
# ------------------------------

data "mgc_dbaas_engines" "all_engines" {}

data "mgc_dbaas_instance_types" "all_instance_types" {}

data "mgc_dbaas_instances" "all_db_instances" {
  # status_filter = "ACTIVE"
}

data "mgc_dbaas_instance" "specific_test_instance" {
  id = mgc_dbaas_instances.test_instance.id
}

data "mgc_dbaas_instances_snapshots" "specific_test_instance_snapshots" {
  instance_id = mgc_dbaas_instances.test_instance.id
}

data "mgc_dbaas_parameter_groups" "all_parameter_groups" {
  # engine_id = var.mysql_8_0_engine_id # Example: Filter PGs by engine
}

data "mgc_dbaas_parameter_group" "specific_instance_pg_data" {
  id = mgc_dbaas_parameter_groups.instance_pg.id
}

data "mgc_dbaas_parameter_group" "specific_cluster_pg_data" {
  id = mgc_dbaas_parameter_groups.cluster_pg.id
}

data "mgc_dbaas_parameters" "instance_pg_parameters_data" {
  parameter_group_id = mgc_dbaas_parameter_groups.instance_pg.id
}

data "mgc_dbaas_replica" "specific_dbaas_replica_data" {
  id = mgc_dbaas_replicas.dbaas_replica.id
}

data "mgc_dbaas_replicas" "all_db_replicas" {}

# --- DBaaS Cluster Data Sources ---
# data "mgc_dbaas_clusters" "all_clusters" {
#   # status_filter = "ACTIVE"
#   # engine_id_filter = var.mysql_8_0_engine_id
# }

# data "mgc_dbaas_cluster" "specific_test_cluster_pg" {
#   id = mgc_dbaas_clusters.test_cluster_with_pg.id
# }

# data "mgc_dbaas_cluster" "specific_test_cluster_no_pg" {
#   id = mgc_dbaas_clusters.test_cluster_no_pg.id
# }

# ------------------------------
# Data Source Outputs
# ------------------------------

output "all_engines_data" {
  value = data.mgc_dbaas_engines.all_engines.engines
}

output "all_instance_types_data" {
  value = data.mgc_dbaas_instance_types.all_instance_types.instance_types
}

output "all_db_instances_data" {
  value = data.mgc_dbaas_instances.all_db_instances.instances
}

output "specific_test_instance_data" {
  value = data.mgc_dbaas_instance.specific_test_instance
}

output "specific_test_instance_snapshots_data" {
  value = data.mgc_dbaas_instances_snapshots.specific_test_instance_snapshots
}

output "all_parameter_groups_data" {
  value = data.mgc_dbaas_parameter_groups.all_parameter_groups
}

output "specific_instance_pg_details_data" {
  value = data.mgc_dbaas_parameter_group.specific_instance_pg_data
}

output "specific_cluster_pg_details_data" {
  value = data.mgc_dbaas_parameter_group.specific_cluster_pg_data
}

output "instance_pg_parameters_details_data" {
  value = data.mgc_dbaas_parameters.instance_pg_parameters_data
}

output "specific_dbaas_replica_details_data" {
  value = data.mgc_dbaas_replica.specific_dbaas_replica_data
}

output "all_db_replicas_data" {
  value = data.mgc_dbaas_replicas.all_db_replicas
}

# --- Cluster Data Source Outputs ---
# output "all_dbaas_clusters_data" {
#   value = data.mgc_dbaas_clusters.all_clusters.clusters
# }

# output "specific_test_cluster_pg_data" {
#   value = data.mgc_dbaas_cluster.specific_test_cluster_pg
# }

# output "specific_test_cluster_no_pg_data" {
#   value = data.mgc_dbaas_cluster.specific_test_cluster_no_pg
# }
