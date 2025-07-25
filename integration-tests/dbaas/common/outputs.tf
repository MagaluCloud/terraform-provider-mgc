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
