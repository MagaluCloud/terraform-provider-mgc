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
  default     = "cloud-dbaas-bs1.medium" # Example, adjust if needed
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
  engine_id   = var.mysql_8_0_engine_id
  description = "Parameter group for test instances"
  name        = "test-instance-pg-${random_pet.name.id}"
}

resource "mgc_dbaas_instances" "test_instance" {
  name                  = "test-instance-${random_pet.name.id}"
  user                  = "dbadmin"
  password              = "aComplexP@ssw0rd!Inst"
  engine_name           = "mysql"
  engine_version        = "8.0"
  instance_type         = var.instance_type_label_single
  volume_size           = 60
  backup_retention_days = 10
  backup_start_at       = "16:00:00"
  availability_zone     = var.availability_zone
  parameter_group       = mgc_dbaas_parameter_groups.instance_pg.id
}
