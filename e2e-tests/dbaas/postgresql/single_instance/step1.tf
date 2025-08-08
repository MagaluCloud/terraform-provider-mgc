###############################################################################
# Create Database Instance
###############################################################################
resource "mgc_dbaas_instances" "single_instance" {
  name                  = "${var.db_prefix}-db-terraform-${var.engine_name}${var.engine_version}"
  user                  = "admin"
  password              = var.db_password
  engine_name           = var.engine_name
  engine_version        = var.engine_version
  instance_type         = "BV1-4-10"
  volume_size           = 20
  backup_retention_days = 1
  backup_start_at       = "01:00:00"
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
# Create a Parameter Group for Tenant
###############################################################################
resource "mgc_dbaas_parameter_groups" "terraform_parameter_group" {
  engine_name    = var.engine_name
  engine_version = var.engine_version
  name           = "${var.db_prefix}-parameter-group-terraform-${var.engine_name}${var.engine_version}"
}

###############################################################################
# List All Engines
###############################################################################
data "mgc_dbaas_engines" "all_engines" {
}

output "all_dbaas_engines" {
  description = "Details of all MGC DBaaS engines"
  value       = data.mgc_dbaas_engines.all_engines
}

###############################################################################
# List All Instance-Types
###############################################################################
data "mgc_dbaas_instance_types" "all_instance_types" {
}

output "all_instance_types" {
  description = "Details of all MGC DBaaS instance-types"
  value       = data.mgc_dbaas_instance_types.all_instance_types
}

###############################################################################
# List All Parameter-Groups
###############################################################################
data "mgc_dbaas_parameter_groups" "all_parameter_groups" {
}

output "mgc_dbaas_parameter_groups" {
  description = "Details of all MGC DBaaS parameter groups"
  value       = data.mgc_dbaas_parameter_groups.all_parameter_groups
}

###############################################################################
# List All Instances
###############################################################################
data "mgc_dbaas_instances" "all_instances" {
}

output "all_dbaas_instances" {
  description = "Details of all MGC DBaaS instances"
  value       = data.mgc_dbaas_instances.all_instances.instances
}

###############################################################################
# List All Replicas
###############################################################################
data "mgc_dbaas_replicas" "all_replicas" {
}

output "all_replicas" {
  description = "Details of all MGC DBaaS replicas"
  value       = data.mgc_dbaas_replicas.all_replicas.replicas
}
