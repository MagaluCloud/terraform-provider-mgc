---
page_title: "DBaaS Creation"
subcategory: "Guides"
description: |-
  How to create a database instance or cluster with terraform.
---

# Getting Started with DBaaS on Magalu Cloud

This guide provides simple examples of how to create and manage database resources using Magalu Cloud's Database-as-a-Service (DBaaS) offerings.

## Basic DBaaS Instance

Here's how to create a standalone database instance:

```terraform
resource "mgc_dbaas_instances" "web_db" {
  name                  = "web-database"
  user                  = "admin"
  password              = "SecurePass123!" # Use a stronger password in production
  engine_name           = "mysql"
  engine_version        = "8.0"
  instance_type         = "cloud-dbaas-bs1.small"
  volume_size           = 50
  volume_type           = "CLOUD_NVME15K"
  backup_retention_days = 7
  backup_start_at       = "02:00:00"
  availability_zone     = "br-se1-a"
}
```

Key parameters:

- `name`: Unique name for your database instance
- `user`: Admin username for the database
- `password`: Strong password meeting complexity requirements
- `engine_name/version`: Database type and version
- `instance_type`: Determines compute resources allocated
- `volume_size`: Storage capacity in GB
- `volume_type`: Type of the storage volume (e.g., 'CLOUD_NVME15K' or 'CLOUD_NVME20K')
- `backup_*`: Configures automatic backups

## Creating a Database Cluster

For higher availability, create a database cluster:

```terraform
resource "mgc_dbaas_clusters" "production_db" {
  name                  = "production-cluster"
  user                  = "clusteradmin"
  password              = "ClusterPass123!"
  engine_name           = "mysql"
  engine_version        = "8.0"
  instance_type         = "cloud-dbaas-bs1.medium"
  volume_size           = 100
  volume_type           = "CLOUD_NVME15K"
  backup_retention_days = 14
  backup_start_at       = "01:00:00"
}
```

Clusters provide:

- Automatic failover
- Read replicas
- Better performance for production workloads
- More storage options (`volume_type`)

## Creating a Read Replica

Scale read operations by adding a replica:

```terraform
resource "mgc_dbaas_replicas" "read_replica" {
  name       = "read-replica-1"
  source_id  = mgc_dbaas_instances.web_db.id
}
```

Replicas help by:

- Offloading read queries from primary
- Providing geographic distribution
- Serving as failover targets

## Managing Parameter Groups

Customize database behavior with parameter groups:

```terraform
resource "mgc_dbaas_parameter_groups" "custom_params" {
  engine_id   = "063f3994-b6c2-4c37-96c9-bab8d82d36f7" # MySQL 8.0 engine ID
  name        = "custom-mysql-params"
  description = "Custom MySQL parameters for web apps"
}

resource "mgc_dbaas_parameters" "timeout_settings" {
  parameter_group_id = mgc_dbaas_parameter_groups.custom_params.id
  name               = "MAX_CONNECTIONS"
  value              = 300
}
```

Parameter groups allow:

- Custom database configurations
- Engine-specific tuning
- Grouping related parameters

## Creating Database Snapshots

Backup your data with snapshots:

```terraform
resource "mgc_dbaas_instances_snapshots" "weekly_backup" {
  instance_id = mgc_dbaas_instances.web_db.id
  name        = "weekly-backup"
  description = "Every Sunday backup"
}
```

Snapshots provide:

- Point-in-time recovery
- Data migration between instances
- Testing environments from production data

## Querying Existing Resources

Get information about your existing databases:

```terraform
data "mgc_dbaas_instances" "all_instances" {}

output "instance_list" {
  value = data.mgc_dbaas_instances.all_instances.instances
}

data "mgc_dbaas_clusters" "active_clusters" {
  status_filter = "ACTIVE"
}
```

Data sources help with:

- Inventory management
- Automation workflows
- Monitoring and reporting

### Documentation

Find more in the [DbaaS documentation page](https://docs.magalu.cloud/docs/dbaas/overview/).
