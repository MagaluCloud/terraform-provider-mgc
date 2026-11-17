# Every type of resource that can be tagged.
data "mgc_tag_resource_types" "all" {
}

# Only the types owned by the network product.
data "mgc_tag_resource_types" "network" {
  product = "network"
}

output "taggable_types" {
  value = [for type in data.mgc_tag_resource_types.all.resource_types : type.name]
}
