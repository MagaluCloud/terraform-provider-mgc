# Create a snapshot for a DBaaS instance
resource "mgc_dbaas_parameters" "example" {
  parameter_group_id = "parameter-group-id"
  name               = "max_connections"
  value              = "300"
}
