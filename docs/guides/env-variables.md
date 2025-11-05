---
page_title: "Environment Variables MGC"
subcategory: "Guides"
description: |-
  How to use Environment Variables in MGC Cloud with Terraform.
---

# Environment Variables MGC

## Introduction

This documentation describes how to configure and use Terraform environment variables for MGC Cloud provider. For more information about Terraform environment variables, please refer to the [official Terraform documentation](https://developer.hashicorp.com/terraform/language/values/variables#environment-variables).

All variables should be prefixed with `TF_VAR_` to be automatically loaded by Terraform:

- `TF_VAR_mgc_api_key`
- `TF_VAR_mgc_access_key`
- `TF_VAR_mgc_secret_key`
- `TF_VAR_mgc_region`

These environment variables are used for authentication when working with Terraform. Note that all variables should be prefixed with `TF_VAR_` to be automatically loaded by Terraform.

1. `TF_VAR_mgc_api_key` -
   API key for authentication.

2. `TF_VAR_mgc_access_key` -
   Access Key (Access ID) for Object Storage operations.

3. `TF_VAR_mgc_secret_key` -
   Secret Key for Object Storage operations.

4. `TF_VAR_mgc_region` -
   Specifies the region where resources will be created and managed.

## Setting Environment Variables

You can set these variables in your shell before running Terraform:

```bash
export TF_VAR_mgc_api_key="your-api-key"
export TF_VAR_mgc_access_key="your-access-key"
export TF_VAR_mgc_secret_key="your-secret-key"
export TF_VAR_mgc_region="your-region"
```

## Configuration in Terraform

Example:

```hcl
provider "mgc" {
  alias      = "nordeste"
  region     = var.mgc_region
  api_key    = var.mgc_api_key
  access_key = var.mgc_access_key
  secret_key = var.mgc_secret_key
}
```

variable "mgc_api_key" {
description = "API key for authentication."
}

variable "mgc_access_key" {
description = "Access Key (Access ID) for Object Storage operations."
}

variable "mgc_secret_key" {
description = "Secret Key for Object Storage operations."
}

variable "mgc_region" {
description = "Specifies the region where resources will be created and managed."
}

```

```
