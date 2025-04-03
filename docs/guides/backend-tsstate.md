---
page_title: "Persisting Terraform State in MGC"
subcategory: "Guides"
description: |-
  How to persist Terraform state in MGC Cloud using object storage.
---

# Using S3 Backend for Terraform State in Magalu Cloud

This guide explains how to configure Terraform to use the S3 backend for storing the Terraform state file (`terraform.tfstate`) in Magalu Cloud using its S3-compatible object storage. This approach enhances your Terraform projects with better state management, collaboration, and security.

## Prerequisites

- Terraform installed on your local machine.
- Access to Magalu Cloud with permissions to create and manage object buckets.
- An object bucket created in Magalu Cloud for storing the Terraform state file.
- Access and secret keys with appropriate permissions to read from and write to the bucket.

## Configuration Steps

### 1. Create an S3 Bucket in Magalu Cloud

Before configuring Terraform, ensure you have an object bucket in Magalu Cloud

### 2. Obtain Your Access Key and Secret Key

For Terraform to access your bucket, you need to create or use existing credentials.
Ensure the key has permissions for the following actions:

- `s3:GetObject`
- `s3:PutObject`
- `s3:DeleteObject`
- `s3:ListBucket`

### 3. Configure the Terraform Backend

In your Terraform configuration file (`main.tf`), specify the S3 backend settings under the `backend "s3"` block. Here's a detailed configuration tailored for Magalu Cloud's S3-compatible implementation:

```terraform
terraform {
  required_providers {
    mgc = {
      source = "magalucloud/mgc"
    }
  }

  backend "s3" {
    bucket                      = "terraform-state-bucket"
    key                         = "project/environment/terraform.tfstate"
    secret_key                  = "your-secret-key"
    access_key                  = "your-access-key"
    region                      = "your-region"         # e.g., "br-ne1"

    # Required for S3-compatible storage
    skip_region_validation      = true
    skip_credentials_validation = true
    skip_requesting_account_id  = true
    skip_s3_checksum            = true

    # Endpoint specific to Magalu Cloud
    endpoints = {
      s3 = "https://your-region.magaluobjects.com/"  # Replace with your region
    }
  }
}
```
