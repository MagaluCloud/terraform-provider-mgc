data "mgc_tag_values" "environment" {
  tag_name = "environment"
}

output "environment_values" {
  value = [for value in data.mgc_tag_values.environment.values : value.name]
}
