data "mgc_tag_value" "production" {
  tag_name = "environment"
  name     = "production"
}

output "production_description" {
  value = data.mgc_tag_value.production.description
}
