data "mgc_tag_value" "producao" {
  tag_name = "ambiente"
  name     = "producao"
}

output "producao_description" {
  value = data.mgc_tag_value.producao.description
}
