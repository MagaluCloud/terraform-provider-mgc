resource "mgc_object_storage_buckets" "basic_bucket" {
  bucket           = "smoke-test-basic-bucket"
  bucket_is_prefix = false
}

data "mgc_object_storage_buckets" "objects_data" {
}

data "mgc_object_storage_bucket" "object_data" {
  name = "felipe-felipe-teste"
}

output "buckets" {
  value = data.mgc_object_storage_buckets.objects_data
}

output "bucket" {
  value = data.mgc_object_storage_bucket.object_data
}
