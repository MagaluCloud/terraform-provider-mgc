resource "mgc_tag" "ambiente" {
  name  = "ambiente"
  kinds = ["finops"]
}

resource "mgc_tag_value" "producao" {
  tag_name    = mgc_tag.ambiente.name
  name        = "producao"
  description = "Recursos do ambiente produtivo"
}
