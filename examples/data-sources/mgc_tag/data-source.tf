data "mgc_tag" "environment" {
  name = "environment"
}

output "environment_color" {
  value = data.mgc_tag.environment.color
}

# The answer already embeds the values defined for the tag.
output "environment_values" {
  value = [for value in data.mgc_tag.environment.values : value.name]
}
