resource "mgc_tag" "environment" {
  name  = "environment"
  kinds = ["finops"]
}

resource "mgc_tag_value" "production" {
  tag_name    = mgc_tag.environment.name
  name        = "production"
  description = "Resources of the production environment"
}
