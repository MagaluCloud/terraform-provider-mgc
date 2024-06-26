---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "mgc_virtual-machine_snapshots Resource - terraform-provider-mgc"
subcategory: ""
description: |-
  Operations with snapshots for instances.
---

# mgc_virtual-machine_snapshots (Resource)

Operations with snapshots for instances.



<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `name` (String)
- `virtual_machine` (Attributes) (see [below for nested schema](#nestedatt--virtual_machine))

### Read-Only

- `created_at` (String)
- `current_name` (String)
- `id` (String) The ID of this resource.
- `instance` (Attributes) (see [below for nested schema](#nestedatt--instance))
- `size` (Number)
- `state` (String)
- `status` (String)
- `updated_at` (String)

<a id="nestedatt--virtual_machine"></a>
### Nested Schema for `virtual_machine`

Optional:

- `id` (String)
- `name` (String)


<a id="nestedatt--instance"></a>
### Nested Schema for `instance`

Read-Only:

- `id` (String)
- `image` (Attributes) (see [below for nested schema](#nestedatt--instance--image))
- `machine_type` (Attributes) (see [below for nested schema](#nestedatt--instance--machine_type))

<a id="nestedatt--instance--image"></a>
### Nested Schema for `instance.image`

Read-Only:

- `id` (String)
- `name` (String)
- `platform` (String)


<a id="nestedatt--instance--machine_type"></a>
### Nested Schema for `instance.machine_type`

Read-Only:

- `disk` (Number)
- `id` (String)
- `name` (String)
- `ram` (Number)
- `vcpus` (Number)
