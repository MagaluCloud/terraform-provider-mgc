# Every resource of the tenant that carries at least one tag.
data "mgc_tag_attachments" "all" {
}

# Only the tagged VPCs of one region.
data "mgc_tag_attachments" "vpcs" {
  resource_type = "net.vpc"
  region        = "br-se1"
}

output "tagged_vpc_ids" {
  value = [for attachment in data.mgc_tag_attachments.vpcs.attachments : attachment.resource_id]
}

data "mgc_tag_attachments" "environment" {
  tag_name = "environment"
}

output "production_resources" {
  value = [
    for attachment in data.mgc_tag_attachments.environment.attachments :
    attachment.resource_id if attachment.tags["environment"] == "production"
  ]
}
