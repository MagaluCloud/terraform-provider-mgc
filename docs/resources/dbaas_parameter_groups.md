---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "mgc_dbaas_parameter_groups Resource - terraform-provider-mgc"
subcategory: "Database"
description: |-
  Manages a DBaaS parameters groups
---

# mgc_dbaas_parameter_groups (Resource)

Manages a DBaaS parameters groups

## Example Usage

```terraform
# Create a snapshot for a DBaaS instance
resource "mgc_dbaas_parameter_groups" "example" {
  engine_name    = "mysql"
  engine_version = "8.0"
  name           = "my-custom-parameters"
}
```

<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `engine_name` (String) Type of database engine to use (e.g., 'mysql', 'postgresql'). Cannot be changed after creation.
- `engine_version` (String) Version of the database engine (e.g., '8.0', '13.3'). Must be compatible with the selected engine_name.
- `name` (String) Name of the parameters group

### Optional

- `description` (String) Description of the parameters group

### Read-Only

- `id` (String) Unique identifier for the parameters group

## Import

Import is supported using the following syntax:

The [`terraform import` command](https://developer.hashicorp.com/terraform/cli/commands/import) can be used, for example:

```shell
terraform import mgc_dbaas_parameter_groups.example "parameter-group-id"
```
