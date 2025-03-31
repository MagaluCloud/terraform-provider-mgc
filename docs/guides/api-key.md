# API Keys

This page describes how to configure and use API Keys with the Magalu Cloud Terraform Provider.

## Using API Keys with the Provider

API Keys are used for authentication with the Magalu Cloud API when using the Terraform provider. To use an API Key with the provider, configure it in your Terraform configuration:

```terraform
provider "mgc" {
  api_key = var.api_key
  region  = var.region
}
```

It's recommended to use variables for sensitive information like API keys:

```terraform
variable "api_key" {
  type        = string
  sensitive   = true
  description = "The Magalu Cloud API Key"
}
```

## Creating API Keys

### Via Console

1. Access the Magalu ID (ensure you're authenticated with an account that has the necessary access)
2. Click on "Create API Key"
3. Define a name for the API Key
4. Select the expiration period
5. Select the Magalu Cloud applications you want to give permission to
6. Copy the API Key

> **Warning:** Keep your API Key secret to prevent unauthorized access.

### Via CLI

1. Execute the command:

   ```
   mgc auth api-key create --name="api-key-name"
   ```

2. Select which scopes the API Key will have access to by pressing enter on the desired scopes.

3. Press tab to complete the creation. The output will look similar to:

   ```
   Select scopes:
     > Virtual Machine [Read]
   uuid: f4b2345c-fue1-4176-a525-fasdfaaa
   ```

4. Since the output only displays the UUID, query to get the actual API Key:
   ```
   mgc auth api-key get f4b2345c-fue1-4176-a525-fasdfaaa
   ```

Your API Key will be returned and can now be used with the Terraform provider.

## Important Notes

- The required field for the provider is the `api_key` value itself, not the API Key ID or key pair.
- API Keys should have the necessary permissions for resources you want to manage with Terraform.
