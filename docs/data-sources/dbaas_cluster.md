---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "mgc_dbaas_cluster Data Source - terraform-provider-mgc"
subcategory: "Database"
description: |-
  Retrieves information about a specific DBaaS cluster.
---

# mgc_dbaas_cluster (Data Source)

Retrieves information about a specific DBaaS cluster.

## Example Usage

```terraform
data "mgc_dbaas_cluster" "specific_test_cluster_pg" {
  id = mgc_dbaas_cluster.test_cluster_with_pg.id
}
```

<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `id` (String) The ID of the DBaaS cluster to retrieve.

### Read-Only

- `addresses` (Attributes List) Network addresses for connecting to the cluster. (see [below for nested schema](#nestedatt--addresses))
- `apply_parameters_pending` (Boolean) Indicates if parameter changes are pending application.
- `backup_retention_days` (Number) Number of days to retain automated backups.
- `backup_start_at` (String) Time to initiate the daily backup in UTC (format: 'HH:MM:SS').
- `created_at` (String) Timestamp of when the cluster was created.
- `engine_id` (String) ID of the database engine used by the cluster.
- `finished_at` (String) Timestamp of when the cluster last finished an operation.
- `instance_type_id` (String) ID of the instance type for the cluster nodes.
- `name` (String) Name of the DBaaS cluster.
- `parameter_group_id` (String) ID of the parameter group associated with the cluster.
- `started_at` (String) Timestamp of when the cluster was last started.
- `status` (String) Current status of the DBaaS cluster.
- `updated_at` (String) Timestamp of when the cluster was last updated.
- `volume_size` (Number) Size of the storage volume in GB.
- `volume_type` (String) Type of the storage volume.

<a id="nestedatt--addresses"></a>
### Nested Schema for `addresses`

Read-Only:

- `access` (String) Access type (e.g., 'public', 'private').
- `address` (String) The IP address or hostname.
- `port` (String) The port number.
- `type` (String) Address type (e.g., 'read-write', 'read-only').
