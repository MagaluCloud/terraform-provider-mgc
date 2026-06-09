data "mgc_object_storage_objects" "objects" {
  bucket = "my-bucket"
}

output "objects" {
  value = data.mgc_object_storage_objects.objects
}
