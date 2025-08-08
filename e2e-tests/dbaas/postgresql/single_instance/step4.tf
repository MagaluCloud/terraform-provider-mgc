###############################################################################
# Database Instance: Resize Volume and Instance-Type
###############################################################################

resource "mgc_dbaas_instances" "single_instance" {
  name                  = "${var.db_prefix}-db-terraform-${var.engine_name}${var.engine_version}"
  user                  = "admin"
  password              = var.db_password
  engine_name           = var.engine_name
  engine_version        = var.engine_version
  instance_type         = "BV1-4-10"
  volume_size           = 40
  backup_retention_days = 2
  backup_start_at       = "02:00:00"
  availability_zone     = "${var.mgc_region}-${var.mgc_zone}"
  parameter_group       = resource.mgc_dbaas_parameter_groups.terraform_parameter_group.id
}

output "single_instance_id" {
  description = "Created Single Instance ID"
  value       = resource.mgc_dbaas_instances.single_instance.id
}

data "mgc_dbaas_instance" "single_instance" {
  id = resource.mgc_dbaas_instances.single_instance.id
}

output "single_instance" {
  description = "Created Single Instance"
  value       = data.mgc_dbaas_instance.single_instance
}

###############################################################################
### >>>              KEEP CREATED INFRASTRUCTURE                        <<< ###
###############################################################################

###############################################################################
# Create Database Replica for Instance
###############################################################################
resource "mgc_dbaas_replicas" "single_instance_replica" {
  name                  = "${var.db_prefix}-replica-terraform-${var.engine_name}${var.engine_version}"
  source_id             = resource.mgc_dbaas_instances.single_instance.id
}

output "single_instance_replica_id" {
  description = "Created Single Instance Replica ID"
  value       = resource.mgc_dbaas_replicas.single_instance_replica.id
}

data "mgc_dbaas_replica" "single_instance_replica" {
  id = resource.mgc_dbaas_replicas.single_instance_replica.id
}

output "single_instance_replica" {
  description = "Created Single Instance Replica"
  value       = data.mgc_dbaas_replica.single_instance_replica
}

###############################################################################
# Create a Snapshot for Instance
###############################################################################
resource "mgc_dbaas_instances_snapshots" "create_manual_snapshot" {
  instance_id = resource.mgc_dbaas_instances.single_instance.id
  name        = "${var.db_prefix}-snapshot-terraform-${var.engine_name}${var.engine_version}"
  description = "Snapshot created via Terraform"
}

###############################################################################
# Create a Parameter Group for Tenant
###############################################################################
resource "mgc_dbaas_parameter_groups" "terraform_parameter_group" {
  engine_name    = var.engine_name
  engine_version = var.engine_version
  name           = "${var.db_prefix}-parameter-group-terraform-${var.engine_name}${var.engine_version}"
}
