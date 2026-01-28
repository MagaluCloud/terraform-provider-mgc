variable "bucket_name" {
  type        = string
  description = "Name of the test bucket"
  default     = "test-mgc-tf-basic-9987775998"
}

variable "bucket_policy_tenant_id" {
  type        = string
  description = "Tenant ID allowed to read objects from the policy example bucket"
  default     = "123e4567-e89b-12d3-a456-426614174000"
}

variable "bucket_policy_source_ip" {
  type        = string
  description = "CIDR block allowed to access the policy example bucket"
  default     = "203.0.113.0/24"
}

# Test 1: Basic bucket creation
resource "mgc_object_storage_buckets" "basic" {
  bucket     = var.bucket_name
  versioning = false
  lock       = false
}

# Test 2: Bucket with simple CORS configuration
resource "mgc_object_storage_buckets" "cors_example" {
  bucket     = format("%s-cors", var.bucket_name)
  versioning = true
  lock       = false

  cors = {
    allowed_methods = ["GET", "HEAD"]
    allowed_origins = ["https://example.com"]
    allowed_headers = ["*"]
    expose_headers  = ["ETag"]
    max_age_seconds = 3600
  }
}

# Test 3: Bucket with a read-only policy for a specific tenant
resource "mgc_object_storage_buckets" "policy_example" {
  bucket     = format("%s-policy-1", var.bucket_name)
  versioning = false
  lock       = false

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Sid       = "AllowTenantReadOnly"
        Effect    = "Allow"
        Principal = var.bucket_policy_tenant_id
        Action = [
          "s3:ListBucket",
          "s3:GetObject"
        ]
        Resource = [
          format("%s-policy", var.bucket_name),
          format("%s-policy/*", var.bucket_name)
        ]
        Condition = {
          IpAddress = {
            "aws:SourceIp" = [
              var.bucket_policy_source_ip
            ]
          }
        }
      }
    ]
  })
}

# Data source: list all buckets
data "mgc_object_storage_buckets" "all" {}

# Data source: get details for a specific bucket
data "mgc_object_storage_bucket" "cors_details" {
  bucket = mgc_object_storage_buckets.cors_example.bucket
}

data "mgc_object_storage_bucket" "policy_details" {
  bucket = mgc_object_storage_buckets.policy_example.bucket
}
