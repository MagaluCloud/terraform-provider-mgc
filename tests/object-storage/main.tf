variable "bucket-name" {
  default = "test-bucket-oooo"
}

resource "mgc_object_storage_buckets" "basic_bucket" {
  bucket           = var.bucket-name
  bucket_is_prefix = false
}

data "mgc_object_storage_buckets" "objects_data" {
}

data "mgc_object_storage_bucket" "object_data" {
  name       = var.bucket-name
  depends_on = [mgc_object_storage_buckets.basic_bucket]
}

output "buckets" {
  value = data.mgc_object_storage_buckets.objects_data
}

output "bucket" {
  value = data.mgc_object_storage_bucket.object_data
}
