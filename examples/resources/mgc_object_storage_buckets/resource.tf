resource "mgc_object_storage_buckets" "basic" {
  bucket     = "example-basic-bucket"
  versioning = true
}

resource "mgc_object_storage_buckets" "locked" {
  bucket = "example-locked-bucket"
  lock   = true
}

resource "mgc_object_storage_buckets" "cors" {
  bucket = "example-cors-bucket"

  cors = {
    allowed_methods = ["GET", "HEAD"]
    allowed_origins = ["https://app.example.com", "https://admin.example.com"]
    allowed_headers = ["Authorization", "Content-Type"]
    expose_headers  = ["ETag"]
    max_age_seconds = 600
  }
}
