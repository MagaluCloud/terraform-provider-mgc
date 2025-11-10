---
page_title: "API Keys"
subcategory: "Guides"
description: |-
  How to configure and use API Keys with the Magalu Cloud Terraform Provider.
---

# API Keys

This page describes how to configure and use API Keys with the Magalu Cloud Terraform Provider.

## Using API Keys with the Provider

API Keys are used for authentication with the Magalu Cloud API when using the Terraform provider. To use an API Key with the provider, configure it in your Terraform configuration:

```terraform
provider "mgc" {
  api_key    = var.api_key
  region     = var.region
  key_pair_id = var.mgc_key_pair_id
  key_pair_secret = var.mgc_key_pair_secret
}
```

It's recommended to use variables for sensitive information like API keys and key pairs:

```terraform
variable "api_key" {
  type        = string
  sensitive   = true
  description = "The Magalu Cloud API Key"
}

variable "mgc_key_pair_id" {
  type        = string
  sensitive   = true
  description = "The Magalu Cloud key pair id"
}

variable "mgc_key_pair_secret" {
  type        = string
  sensitive   = true
  description = "The Magalu Cloud key pair secret"
}
```

## API Key Components

When you create an API Key in Magalu Cloud, you actually receive multiple credential components:

1. **API Key ID**: A unique identifier for your API Key
2. **API Key**: The primary credential used for general API authentication (used in the provider configuration)
3. **Key Pair**: A pair of credentials (ID and Secret) used specifically for Object Storage access

## Creating API Keys

### Via Console

1. Access the [Magalu ID](https://id.magalu.com/api-keys) (ensure you're authenticated with an account that has the necessary access)
2. Click on "Create API Key"
3. Define a name for the API Key
4. Select the expiration period
5. Select the Magalu Cloud applications you want to give permission to
   - For Object Storage access, ensure you select the appropriate Object Storage scopes (Read/Write)
6. Copy the API Key and save the Key Pair (ID/Secret) if needed for Object Storage

> **Warning:** Keep your API Key and Key Pair secret to prevent unauthorized access.

### Via CLI

1. Execute the command:

   ```
   mgc auth api-key create --name="api-key-name"
   ```

2. Select which scopes the API Key will have access to by pressing enter on the desired scopes.
   - For Object Storage operations, select the Object Storage scopes
   - There are separate scopes for read and write operations

3. Press tab to complete the creation. The output will look similar to:

   ```
   Select scopes:
     > Virtual Machine [Read]
     > Object Storage [Read]
     > Object Storage [Write]
   uuid: f4b2345c-fue1-4176-a525-fasdfaaa
   ```

4. Since the output only displays the UUID, query to get the actual API Key and Key Pair:
   ```
   mgc auth api-key get f4b2345c-fue1-4176-a525-fasdfaaa
   ```

The response will include:

- The API Key for general authentication with the provider
- The Key Pair (ID and Secret) if you selected Object Storage scopes

## Using Key Pair for Object Storage

When working with Object Storage in the Magalu Cloud Terraform Provider, you'll need to configure the Key Pair (ID and Secret) that was generated along with your API Key. This is only necessary for object storage operations in the provider; other operations only require the API key.

Provide the credentials using the `key_pair_id` and `key_pair_secret` arguments in the provider configuration:

```terraform
provider "mgc" {
  api_key    = var.api_key
  region     = var.region
  key_pair_id = var.mgc_key_pair_id
  key_pair_secret = var.mgc_key_pair_secret
}
```

You can define your variables like this:

```terraform
variable "api_key" {
  type        = string
  sensitive   = true
  description = "The Magalu Cloud API Key"
}

variable "mgc_key_pair_id" {
  type        = string
  sensitive   = true
  description = "The Magalu Cloud key pair id"
}

variable "mgc_key_pair_secret" {
  type        = string
  sensitive   = true
  description = "The Magalu Cloud key pair secret"
}
```

Remember that:

1. The regular `api_key` is still required for all operations
2. The key pair (id and secret) is only needed for object storage operations
3. You must have selected Object Storage scopes when creating your API key to receive a valid key pair

## Important Notes

- The required field for the provider is the `api_key` value itself, not the API Key ID.
- API Keys should have the necessary permissions for resources you want to manage with Terraform.
- For Object Storage operations, ensure your API Key has the appropriate Object Storage scopes.
- The Key Pair (ID and Secret) is only generated if you select Object Storage scopes when creating the API Key.
- Object Storage scopes are divided into Read and Write - select both if you need full access.
