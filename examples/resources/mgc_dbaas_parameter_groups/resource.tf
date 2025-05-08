# Create a snapshot for a DBaaS instance
resource "mgc_dbaas_parameter_groups" "example" {
  engine_id = "db-engine-id"
  name      = "my-custom-parameters"
}
