resource "mgc_tag" "environment" {
  name        = "environment"
  description = "Group resources by environment"
  color       = "0086ff"
  kinds       = ["finops"]
}
