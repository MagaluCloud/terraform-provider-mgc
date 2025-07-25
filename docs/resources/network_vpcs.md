---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "mgc_network_vpcs Resource - terraform-provider-mgc"
subcategory: "Network"
description: |-
  Network VPC
---

# mgc_network_vpcs (Resource)

Network VPC

## Example Usage

```terraform
resource "mgc_network_vpcs" "example" {
  name        = "example-vpc"
  description = "An example VPC"
}
```

<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `name` (String) The name of the VPC

### Optional

- `description` (String) The description of the VPC

### Read-Only

- `id` (String) The ID of the VPC

## Import

Import is supported using the following syntax:

The [`terraform import` command](https://developer.hashicorp.com/terraform/cli/commands/import) can be used, for example:

```shell
terraform import mgc_network_vpcs.example 123
```
