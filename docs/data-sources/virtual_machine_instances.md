---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "mgc_virtual_machine_instances Data Source - terraform-provider-mgc"
subcategory: "Virtual Machine"
description: |-
  Get the available virtual-machine instances.
---

# mgc_virtual_machine_instances (Data Source)

Get the available virtual-machine instances.

## Example Usage

```terraform
data "mgc_virtual_machine_instances" "instances" {
}

output "vm_instances" {
  value = data.mgc_virtual_machine_instances.instances
}
```

<!-- schema generated by tfplugindocs -->
## Schema

### Read-Only

- `instances` (Attributes List) List of available VM instances. (see [below for nested schema](#nestedatt--instances))

<a id="nestedatt--instances"></a>
### Nested Schema for `instances`

Read-Only:

- `availability_zone` (String) Availability zone of instance
- `id` (String) ID of machine-type.
- `image_id` (String) Image ID of instance
- `machine_type_id` (String) Machine type ID of instance
- `name` (String) Name of type.
- `ssh_key_name` (String) SSH Key name
- `state` (String) State of instance
- `status` (String) Status of instance.
