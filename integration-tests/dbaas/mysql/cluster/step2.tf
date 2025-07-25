###############################################################################
# Create Database Cluster: Adjust Backup/Snapshot Period
###############################################################################
resource "mgc_dbaas_clusters" "cluster_instance" {
  name                  = "${var.db_prefix}-cluster-terraform-${var.engine_name}${var.engine_version}"
  user                  = "admin"
  password              = var.db_password
  engine_name           = var.engine_name
  engine_version        = var.engine_version
  instance_type         = "DP2-8-40"
  volume_size           = 20
  backup_retention_days = 2
  backup_start_at       = "02:00:00"
  parameter_group       = resource.mgc_dbaas_parameter_groups.terraform_parameter_group.id
}

output "cluster_instance_id" {
  description = "Created Cluster Instance ID"
  value       = resource.mgc_dbaas_clusters.cluster_instance.id
}

data "mgc_dbaas_cluster" "cluster_instance" {
  id = resource.mgc_dbaas_clusters.cluster_instance.id
}

output "cluster_instance" {
  description = "Created Cluster Instance"
  value       = data.mgc_dbaas_cluster.cluster_instance
}

###############################################################################
### >>>              KEEP CREATED INFRASTRUCTURE                        <<< ###
###############################################################################

###############################################################################
# List All Clusters
###############################################################################

data "mgc_dbaas_clusters" "all_clusters" {
}

output "all_dbaas_clusters" {
  description = "Details of all MGC DBaaS clusters"
  value       = data.mgc_dbaas_clusters.all_clusters.clusters
}

# ###############################################################################
# # Create a Snapshot for Cluster
# ###############################################################################
# resource "mgc_dbaas_clusters_snapshots" "create_manual_snapshot" {
#   cluster_id = resource.mgc_dbaas_cluster.cluster_instance.id
#   name        = "${var.db_prefix}-snapshot-cluster-terraform-${var.engine_name}${var.engine_version}"
#   description = "Snapshot created via Terraform"
# }

###############################################################################
# Create a Parameter Group for Tenant
###############################################################################
resource "mgc_dbaas_parameter_groups" "terraform_parameter_group" {
  engine_name    = var.engine_name
  engine_version = var.engine_version
  name           = "${var.db_prefix}-cluster-parameter-group-terraform-${var.engine_name}${var.engine_version}"
}

###############################################################################
# Update a Parameter Group
###############################################################################
resource "mgc_dbaas_parameters" "terraform_parameter_group" {
  parameter_group_id = resource.mgc_dbaas_parameter_groups.terraform_parameter_group.id
  name               = "MAX_CONNECTIONS"
  value              = 300
}

data "mgc_dbaas_parameter_group" "terraform_parameter_group" {
  id = resource.mgc_dbaas_parameter_groups.terraform_parameter_group.id
}

output "terraform_parameter_group" {
  description = "Details of created parameter group"
  value       = data.mgc_dbaas_parameter_group.terraform_parameter_group
}
