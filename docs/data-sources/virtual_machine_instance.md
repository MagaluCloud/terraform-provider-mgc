---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "mgc_virtual_machine_instance Data Source - terraform-provider-mgc"
subcategory: "Virtual Machine"
description: |-
  Get the available virtual-machine instance details
---

# mgc_virtual_machine_instance (Data Source)

Get the available virtual-machine instance details

## Example Usage

```terraform
data "mgc_virtual_machine_instances" "instances" {
  id = mgc_virtual_machine_instances.my_vm.id
}

output "vm_instances" {
  value = data.mgc_virtual_machine_instances.instances
}
```

<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `id` (String) ID of machine-type.

### Read-Only

- `availability_zone` (String) Availability zone of instance.
- `created_at` (String) Creation timestamp of the instance.
- `error_message` (String) Error message if any.
- `error_slug` (String) Error slug if any.
- `image_id` (String) Image ID of instance.
- `image_name` (String) Image name of instance.
- `image_platform` (String) Image platform type.
- `interfaces` (Attributes List) Network interfaces attached to the instance. (see [below for nested schema](#nestedatt--interfaces))
- `labels` (List of String) Labels associated with the instance.
- `machine_type_disk` (Number) Machine type disk size.
- `machine_type_id` (String) Machine type ID of instance.
- `machine_type_name` (String) Machine type name.
- `machine_type_ram` (Number) Machine type RAM size.
- `machine_type_vcpus` (Number) Machine type vCPUs count.
- `name` (String) Name of instance.
- `ssh_key_name` (String) SSH Key name.
- `state` (String) State of instance.
- `status` (String) Status of instance.
- `updated_at` (String) Last update timestamp of the instance.
- `user_data` (String) User data of instance.
- `vpc_id` (String) VPC ID.
- `vpc_name` (String) VPC name.

<a id="nestedatt--interfaces"></a>
### Nested Schema for `interfaces`

Read-Only:

- `id` (String) Interface ID.
- `local_ipv4` (String) Local IPv4 address.
- `name` (String) Interface name.
- `primary` (Boolean) Whether this is the primary interface.
- `public_ipv4` (String) Public IPv4 address.
- `public_ipv6` (String) Public IPv6 address.
- `security_groups` (List of String) Security groups associated with the interface.
