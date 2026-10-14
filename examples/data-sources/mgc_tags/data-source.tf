# Every tag of the tenant.
data "mgc_tags" "all" {
}

# Only the tags used for cost reporting.
data "mgc_tags" "finops" {
  kinds = ["finops"]
}

output "finops_tag_names" {
  value = [for tag in data.mgc_tags.finops.tags : tag.name]
}
