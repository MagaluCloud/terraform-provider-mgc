data "mgc_tag" "ambiente" {
  name = "ambiente"
}

output "ambiente_color" {
  value = data.mgc_tag.ambiente.color
}

# The answer already embeds the values defined for the tag.
output "ambiente_values" {
  value = [for value in data.mgc_tag.ambiente.values : value.name]
}
