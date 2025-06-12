# Create a snapshot for a DBaaS instance
resource "mgc_dbaas_parameter_groups" "example" {
  engine_name    = "mysql"
  engine_version = "8.0"
  name           = "my-custom-parameters"
}
