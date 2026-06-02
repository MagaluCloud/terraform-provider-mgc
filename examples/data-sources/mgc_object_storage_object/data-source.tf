data "mgc_object_storage_object" "object" {
  bucket = mgc_object_storage_buckets.my_bucket.bucket
  key    = "path/to/file.txt"
}

output "object" {
  value = data.mgc_object_storage_object.object
}
