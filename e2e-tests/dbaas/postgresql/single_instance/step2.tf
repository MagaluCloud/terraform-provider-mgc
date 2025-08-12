###############################################################################
# Database Instance: Resize Volume
###############################################################################
resource "mgc_dbaas_instances" "single_instance" {
  name                  = "${var.db_prefix}-db-terraform-${var.engine_name}${var.engine_version}"
  user                  = "admin"
  password              = var.db_password
  engine_name           = var.engine_name
  engine_version        = var.engine_version
  instance_type         = "BV1-4-10"
  volume_size           = 30
  backup_retention_days = 1
  backup_start_at       = "01:00:00"
  availability_zone     = "${var.mgc_region}-${var.mgc_zone}"
  parameter_group       = resource.mgc_dbaas_parameter_groups.terraform_parameter_group.id
}

###############################################################################
### >>>              KEEP CREATED INFRASTRUCTURE                        <<< ###
###############################################################################

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

###############################################################################
# Update a Parameter Group
###############################################################################
# resource "mgc_dbaas_parameters" "terraform_parameter_group" {
#   parameter_group_id = resource.mgc_dbaas_parameter_groups.terraform_parameter_group.id
#   name               = "MAX_CONNECTIONS"
#   value              = 300
# }
# resource "mgc_dbaas_parameters" "terraform_parameter_general_log" {
#   parameter_group_id = resource.mgc_dbaas_parameter_groups.terraform_parameter_group.id
#   name               = "GENERAL_LOG"
#   value              = "OFF"
# }

# data "mgc_dbaas_parameter_group" "terraform_parameter_group" {
#   id = resource.mgc_dbaas_parameter_groups.terraform_parameter_group.id
# }

# output "terraform_parameter_group" {
#   description = "Details of created parameter group"
#   value       = data.mgc_dbaas_parameter_group.terraform_parameter_group
# }
