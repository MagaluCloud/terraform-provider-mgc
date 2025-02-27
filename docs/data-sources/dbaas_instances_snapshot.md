---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "mgc_dbaas_instances_snapshot Data Source - terraform-provider-mgc"
subcategory: "Database"
description: |-
  Get a database snapshot by ID.
---

# mgc_dbaas_instances_snapshot (Data Source)

Get a database snapshot by ID.

## Example Usage

```terraform
data "mgc_dbaas_instances_snapshot" "example" {
  id          = "snapshot-123"
  instance_id = "instance-123"
}
```

<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `id` (String) ID of the snapshot
- `instance_id` (String) ID of the instance

### Read-Only

- `created_at` (String) Creation timestamp
- `description` (String) Description of the snapshot
- `name` (String) Name of the snapshot
- `size` (Number) Size of the snapshot in bytes
- `status` (String) Status of the snapshot
