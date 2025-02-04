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
- `TF_VAR_mgc_obj_key_id`
- `TF_VAR_mgc_obj_key_secret`
- `TF_VAR_mgc_region`
- `TF_VAR_mgc_env`

These environment variables are used for authentication and environment configuration when working with Terraform. Note that all variables should be prefixed with `TF_VAR_` to be automatically loaded by Terraform.

1. `TF_VAR_mgc_api_key` - 
API key for authentication.

2. `TF_VAR_mgc_obj_key_id` - 
Key ID to access the Object Storage product.

3. `TF_VAR_mgc_obj_key_secret` - 
*Secret* of the key to access the Object Storage product. 

4. `TF_VAR_mgc_region` - 
Specifies the region where resources will be created and managed.

5. `TF_VAR_mgc_env` - 
Defines the operating environment to differentiate between different phases of development.

## Setting Environment Variables

You can set these variables in your shell before running Terraform:

```bash
export TF_VAR_mgc_api_key="your-api-key"
export TF_VAR_mgc_obj_key_id="your-key-id"
export TF_VAR_mgc_obj_key_secret="your-key-secret"
export TF_VAR_mgc_region="your-region"
export TF_VAR_mgc_env="your-environment"
```

## Configuration in Terraform

Example:

```hcl
provider "mgc" {
  alias = "nordeste"
  region = var.mgc_region
  api_key = var.mgc_api_key
  object_storage = {
    key_pair = {
      key_id = var.mgc_obj_key_id
      key_secret = var.mgc_obj_key_secret
    }
  }
}

variable "mgc_api_key" {
  description = "API key for authentication."
}

variable "mgc_obj_key_id" {
  description = "Key ID to access the Object Storage product."
}

variable "mgc_obj_key_secret" {
  description = "Secret of the key to access the Object Storage product."
}

variable "mgc_region" {
  description = "Specifies the region where resources will be created and managed."
}

variable "mgc_env" {
  description = "Defines the operating environment"
}
```
