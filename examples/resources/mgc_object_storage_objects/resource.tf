resource "mgc_object_storage_objects" "from_file" {
  bucket = mgc_object_storage_buckets.my_bucket.bucket
  key    = "path/to/file.txt"
  source = "./file.txt"
}

resource "mgc_object_storage_objects" "inline" {
  bucket  = mgc_object_storage_buckets.my_bucket.bucket
  key     = "hello.txt"
  content = "Hello, World!"
}
