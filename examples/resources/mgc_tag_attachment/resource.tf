resource "mgc_tag" "ambiente" {
  name  = "ambiente"
  kinds = ["finops"]
}

resource "mgc_tag_value" "producao" {
  tag_name = mgc_tag.ambiente.name
  name     = "producao"
}

resource "mgc_network_vpcs" "main" {
  name = "vpc-exemplo"
}

# This resource owns every tag of the target: a tag attached outside Terraform is
# removed on the next apply.
resource "mgc_tag_attachment" "vpc" {
  resource_id = mgc_network_vpcs.main.id

  # The parentheses turn the key into an expression. Referencing the tag and the
  # value (instead of writing them as literal strings) is also what makes
  # Terraform create both before attaching them.
  tags = {
    (mgc_tag.ambiente.name) = mgc_tag_value.producao.name
  }
}
