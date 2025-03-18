# DBaaS Instance resource creation
resource "mgc_dbaas_instances" "test_instance" {
  name                  = "test-instance-tf-acceptance"
  user                  = "dbadmin"
  password              = "aaaaaaaaaa"
  engine_name           = "mysql"
  engine_version        = "8.0"
  instance_type         = "cloud-dbaas-bs1.small"
  volume_size           = 60
  backup_retention_days = 10
  backup_start_at       = "16:00:00"
}

# Output the created instance details
output "dbaas_instance" {
  value     = mgc_dbaas_instances.test_instance
  sensitive = true
}

# DBaaS Snapshot resource creation
resource "mgc_dbaas_instances_snapshots" "test_snapshot" {
  instance_id = mgc_dbaas_instances.test_instance.id
  name        = "test-snapshot-tf-acceptance"
  description = "Test snapshot for terraform acceptance tests"
}

# Output the created snapshot details
output "test_snapshot" {
  value = mgc_dbaas_instances_snapshots.test_snapshot
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
