resource "random_pet" "name" {
  length    = 1
  separator = "-"
}


# DBaaS Instance resource creation
resource "mgc_dbaas_instances" "test_instance" {
  name                  = "test-instance-${random_pet.name.id}"
  user                  = "dbadmin"
  password              = "aaaaaaaaaa"
  engine_name           = "mysql"
  engine_version        = "8.0"
  instance_type         = "cloud-dbaas-bs1.small"
  volume_size           = 60
  backup_retention_days = 10
  backup_start_at       = "16:00:00"
  availability_zone     = "br-ne1-a"
  parameter_group       = mgc_dbaas_parameter_groups.parameter_group.id
}

# Output the created instance details
output "dbaas_instance" {
  value     = mgc_dbaas_instances.test_instance
  sensitive = true
}

# DBaaS Snapshot resource creation
resource "mgc_dbaas_instances_snapshots" "test_snapshot" {
  instance_id = mgc_dbaas_instances.test_instance.id
  name        = "test-snapshot-${random_pet.name.id}"
  description = "Test snapshot for terraform acceptance tests"
}

# Output the created snapshot details
output "test_snapshot" {
  value = mgc_dbaas_instances_snapshots.test_snapshot
}

# Create a snapshot for a DBaaS instance
resource "mgc_dbaas_parameter_groups" "parameter_group" {
  engine_id   = "063f3994-b6c2-4c37-96c9-bab8d82d36f7"
  description = "my-description"
  name        = "test-parameter-group-${random_pet.name.id}"
}

# Output the created parameter group details
resource "mgc_dbaas_parameters" "example" {
  parameter_group_id = mgc_dbaas_parameter_groups.parameter_group.id
  name               = "MAX_CONNECTIONS"
  value              = 300
}

resource "mgc_dbaas_replicas" "dbaas_replica" {
  name      = "test-replica-${random_pet.name.id}"
  source_id = mgc_dbaas_instances.test_instance.id
}


# ------------------------------
# Data Sources
# ------------------------------

# Engine data sources
data "mgc_dbaas_engines" "all_engines" {}

# Instance Types data sources
data "mgc_dbaas_instance_types" "default_instance_types" {}

# DBaaS Instances data sources
data "mgc_dbaas_instances" "all_instances" {}

# Get specific instance data
data "mgc_dbaas_instance" "test_instance" {
  id = mgc_dbaas_instances.test_instance.id
}

# Get snapshots for the instance
data "mgc_dbaas_instances_snapshots" "test_instance_snapshots" {
  instance_id = mgc_dbaas_instances.test_instance.id
}

# List all parameter groups
data "mgc_dbaas_parameter_groups" "parameter_groups" {}

# Get parameter group data
data "mgc_dbaas_parameter_group" "parameter_group_resource" {
  id = mgc_dbaas_parameter_groups.parameter_group.id
}

data "mgc_dbaas_parameters" "parameters" {
  parameter_group_id = mgc_dbaas_parameter_groups.parameter_group.id
}

# Get replica data
data "mgc_dbaas_replica" "dbaas_replica" {
  id = mgc_dbaas_replicas.dbaas_replica.id
}

# Get all replicas data
data "mgc_dbaas_replicas" "all_replicas" {
}

# ------------------------------
# Data Source Outputs
# ------------------------------

# Engines outputs
output "all_engines" {
  value = data.mgc_dbaas_engines.all_engines.engines
}

# Instance types outputs
output "default_instance_types" {
  value = data.mgc_dbaas_instance_types.default_instance_types.instance_types
}

# Instances outputs
output "all_instances" {
  value = data.mgc_dbaas_instances.all_instances.instances
}

# Specific instance output
output "test_instance_data" {
  value = data.mgc_dbaas_instance.test_instance
}

# Snapshots output
output "test_instance_snapshots" {
  value = data.mgc_dbaas_instances_snapshots.test_instance_snapshots
}

# Parameter group output
output "parameter_group_data" {
  value = data.mgc_dbaas_parameter_group.parameter_group_resource
}

# Parameter groups output
output "parameter_groups_data" {
  value = data.mgc_dbaas_parameter_groups.parameter_groups
}

# Parameters output
output "parameters_data" {
  value = data.mgc_dbaas_parameters.parameters
}

# Replica output
output "dbaas_replica_data" {
  value = data.mgc_dbaas_replica.dbaas_replica
}

# All replicas output
output "all_replicas_data" {
  value = data.mgc_dbaas_replicas.all_replicas
}
