---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "mgc_block_storage_volume_types Data Source - terraform-provider-mgc"
subcategory: "Block Storage"
description: |-
  Block-storage Volume Types
---

# mgc_block_storage_volume_types (Data Source)

Block-storage Volume Types

## Example Usage

```terraform
data "mgc_block_storage_volume_types" "types" {
}

output "types" {
value = {
    for id, volume_types in data.mgc_block_storage_volume_types.types.volume_types :
    id => {
      id = volume_types.id
      name = volume_types.name
    }
  }
}
```

<!-- schema generated by tfplugindocs -->
## Schema

### Read-Only

- `volume_types` (Attributes List) List of available Block-storage Volume Types. (see [below for nested schema](#nestedatt--volume_types))

<a id="nestedatt--volume_types"></a>
### Nested Schema for `volume_types`

Read-Only:

- `availability_zones` (List of String) The volume type availability zones.
- `disk_type` (String) The disk type.
- `id` (String) ID of image.
- `iops` (Number) The volume type IOPS.
- `name` (String) The volume type name.
- `status` (String) The volume type status.