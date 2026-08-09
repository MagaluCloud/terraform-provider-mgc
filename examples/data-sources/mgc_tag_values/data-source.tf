data "mgc_tag_values" "ambiente" {
  tag_name = "ambiente"
}

output "ambiente_values" {
  value = [for value in data.mgc_tag_values.ambiente.values : value.name]
}
