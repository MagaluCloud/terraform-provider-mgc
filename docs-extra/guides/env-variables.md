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
- `TF_VAR_mgc_key_pair_id`
- `TF_VAR_mgc_key_pair_secret`
- `TF_VAR_mgc_region`

These environment variables are used for authentication when working with Terraform. Note that all variables should be prefixed with `TF_VAR_` to be automatically loaded by Terraform.

1. `TF_VAR_mgc_api_key` -
   API key for authentication.

2. `TF_VAR_mgc_key_pair_id` -
   Key Pair ID for Object Storage operations.

3. `TF_VAR_mgc_key_pair_secret` -
   Key Pair Secret for Object Storage operations.

4. `TF_VAR_mgc_region` -
   Specifies the region where resources will be created and managed.

## Setting Environment Variables

You can set these variables in your shell before running Terraform:

```bash
export TF_VAR_mgc_api_key="your-api-key"
export TF_VAR_mgc_key_pair_id="your-key-pair-id"
export TF_VAR_mgc_key_pair_secret="your-key-pair-secret"
export TF_VAR_mgc_region="your-region"
```

## Configuration in Terraform

Example:

```hcl
provider "mgc" {
  alias      = "nordeste"
  region     = var.mgc_region
  api_key    = var.mgc_api_key
  key_pair_id = var.mgc_key_pair_id
  key_pair_secret = var.mgc_key_pair_secret
}
```

variable "mgc_api_key" {
description = "API key for authentication."
}

variable "mgc_key_pair_id" {
description = "Key Pair ID for Object Storage operations."
}

variable "mgc_key_pair_secret" {
description = "Key Pair Secret for Object Storage operations."
}

variable "mgc_region" {
description = "Specifies the region where resources will be created and managed."
}

```

```
