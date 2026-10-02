resource "mgc_tag" "environment" {
  name  = "environment"
  kinds = ["finops"]
}

resource "mgc_tag_value" "production" {
  tag_name = mgc_tag.environment.name
  name     = "production"
}

resource "mgc_network_vpcs" "main" {
  name = "vpc-example"
}

resource "mgc_tag_attachment" "vpc" {
  resource_id = mgc_network_vpcs.main.id

  # The parentheses turn the key into an expression.
  tags = {
    (mgc_tag.environment.name) = mgc_tag_value.production.name
  }
}
