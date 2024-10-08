---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "mgc_network_vpc Resource - terraform-provider-mgc"
subcategory: "Network"
description: |-
  vpc
---

# mgc_network_vpc (Resource)

vpc

## Example Usage

```terraform
resource "mgc_network_vpcs" "my_vpc" {
  name        = "my-vpc"
  description = "My VPC"
}
```

<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `name` (String)

### Optional

- `description` (String)

### Read-Only

- `created_at` (String)
- `current_id` (String)
- `current_name` (String)
- `external_network` (String)
- `id` (String) ID of the VPC to delete
- `is_default` (Boolean)
- `network_id` (String)
- `router_id` (String)
- `security_groups` (List of String)
- `subnets` (List of String)
- `tenant_id` (String)
- `updated` (String)
