---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "mgc_network_nat_gateway Resource - terraform-provider-mgc"
subcategory: "Network"
description: |-
  Manages a NAT Gateway resource.
---

# mgc_network_nat_gateway (Resource)

Manages a NAT Gateway resource.

## Example Usage

```terraform
terraform {
  required_providers {
    mgc = {
      source = "MagaluCloud/mgc"
    }
  }
}

resource "mgc_network_nat_gateway" "my_ngateway" {
  name        = "example-nat-gateway"
  description = "Example NAT Gateway"
  vpc_id      = "vpc-123456"
  zone        = "zone-1"
}
```

<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `name` (String) The name of the NAT Gateway.
- `vpc_id` (String) The ID of the VPC where the NAT Gateway will be created.

### Optional

- `availability_zone` (String) The availability zone of the NAT Gateway.
- `description` (String) The description of the NAT Gateway.

### Read-Only

- `id` (String) The ID of the NAT Gateway.

## Import

Import is supported using the following syntax:

The [`terraform import` command](https://developer.hashicorp.com/terraform/cli/commands/import) can be used, for example:

```shell
#!/bin/bash

terraform import mgc_network_nat_gateway.my_ngateway 123456
```
