# Reads the tags of a resource without taking ownership of them, which is what
# the mgc_tag_attachment resource would do.
data "mgc_tag_attachment" "vpc" {
  resource_id = "b45659b5-2da2-4f60-9096-9d8dc7d96450"
}

output "vpc_environment" {
  value = data.mgc_tag_attachment.vpc.tags["environment"]
}

output "vpc_resource_type" {
  value = data.mgc_tag_attachment.vpc.resource_type
}
