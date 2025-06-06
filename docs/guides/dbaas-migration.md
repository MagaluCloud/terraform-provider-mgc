---
page_title: "DBaaS V2 Migration Guide"
subcategory: "Guides"
description: |-
  How to migrate DBaaS resources to provider versions 0.34.0 and beyond. This guide will walk you through the process of removing the old resource from your state and importing it with the new structure.
---

## ⚠️ Important: State Migration for `mgc_dbaas_instances`

With the latest version of our Terraform provider, the `mgc_dbaas_instances` resource has been updated with a new contract. These changes introduce some important modifications to the resource's schema.

Due to the nature of these changes, a direct in-place upgrade is not possible. You'll need to manually update your Terraform state. This guide will walk you through the process of removing the old resource from your state and importing it with the new structure.

### What's Changed?

We've introduced a few key changes to the `mgc_dbaas_instances` resource:

- **New Arguments:** We've added `availability_zone` and `parameter_group` to give you more control over your database instance's placement and configuration.
- **New Read-Only Field:** The `status` of the instance is now available as a read-only attribute.

### Upgrade Steps

Follow these steps to safely migrate your `mgc_dbaas_instances` resources to the new version.

#### 1. Update Your Terraform Configuration

First, modify your `.tf` files to reflect the new schema. The core required arguments remain the same. You can optionally add the new `availability_zone` and `parameter_group` arguments.

**Old Configuration (`.tf`):**

```terraform
resource "mgc_dbaas_instances" "test_instance" {
  name                  = "test-instance"
  user                  = "dbadmin"
  password              = "examplepassword"
  engine_name           = "mysql"
  engine_version        = "8.0"
  instance_type         = "cloud-dbaas-gp1.small"
  volume_size           = 50
  backup_retention_days = 10
  backup_start_at       = "16:00:00"
}
```

**New Configuration (`.tf`):**

```terraform
resource "mgc_dbaas_instances" "test_instance" {
  name                  = "test-instance"
  user                  = "dbadmin"
  password              = "examplepassword"
  engine_name           = "mysql"
  engine_version        = "8.0"
  instance_type         = "cloud-dbaas-gp1.small"
  volume_size           = 50
  backup_retention_days = 10
  backup_start_at       = "16:00:00"

  # Optional new arguments
  # availability_zone = "us-east-1a"
  # parameter_group   = "default.mysql8.0"
}
```

#### 2. Remove the Resource from Terraform State

Next, you need to tell Terraform to "forget" about the existing `mgc_dbaas_instances` resource without deleting the actual database instance.

Run the following command for each of your DBaaS instances managed by Terraform:

```shell
terraform state rm mgc_dbaas_instances.test_instance
```

Replace `mgc_dbaas_instances.test_instance` with the address of your resource.

#### 3. Import the Resource

Now, import the existing database instance back into your Terraform state. This will associate the real-world resource with your updated Terraform configuration.

You'll need the unique ID of your DBaaS instance to do this.

```shell
terraform import mgc_dbaas_instances.test_instance <your-instance-id>
```

Replace `<your-instance-id>` with the actual ID of your `test_instance`.

#### 4. Verify the Changes

Finally, run `terraform plan` to ensure that your configuration matches the imported state. Terraform should report that no changes are needed.

```shell
terraform plan
```

If the plan shows any differences, you may need to adjust your configuration file to perfectly match the state of the imported resource. Once the plan is clean, you can apply it.

```shell
terraform apply
```

And that's it! Your `mgc_dbaas_instances` resource is now managed under the new contract. If you have any questions or run into issues, please reach out to our support team.
