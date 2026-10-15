---
page_title: "Organize resources with Projects"
subcategory: "Guides"
description: |-
  How to create and import Magalu Cloud resources inside a project with Terraform.
---

# Projects

A project groups resources inside your tenant. Terraform does not create or read projects: create them in the Magalu Cloud console or with the CLI (`mgc project create`), then reference the project ID in your configuration.

## Supported products

Only these products are scoped by project:

| Product          | Resources and data sources                                                                    |
| ---------------- | --------------------------------------------------------------------------------------------- |
| Block Storage    | `mgc_block_storage_*` (except `mgc_block_storage_volume_types`)                               |
| Virtual Machines | `mgc_virtual_machine_*` (except `mgc_virtual_machine_images` and `mgc_virtual_machine_types`) |
| Kubernetes       | `mgc_kubernetes_*` (except `mgc_kubernetes_flavor` and `mgc_kubernetes_version`)              |

Other products, such as networks, load balancers, SSH keys, databases and container registries, belong to the tenant and have no `project_id`. A virtual machine in a project can use a VPC, subnet or SSH key normally.

## Creating resources in a project

Set `project_id` on each resource. When it is omitted, the resource goes to the tenant's default project.

```terraform
variable "project_id" {
  type = string
}

resource "mgc_block_storage_volumes" "data" {
  project_id = var.project_id
  name       = "data"
  size       = 20
  type       = "cloud_nvme20k"
}

resource "mgc_virtual_machine_instances" "app" {
  project_id   = var.project_id
  name         = "app"
  machine_type = "BV1-1-10"
  image        = "cloud-ubuntu-24.04 LTS"
  ssh_key_name = "my-key"
}
```

~> **Important:** `project_id` cannot be changed in place. Changing it destroys the resource and creates it again in the new project. Removing it from the configuration keeps the resource in its current project.

-> Terraform does not check that the project exists before applying. An unknown project, or one you are not allowed to use, is reported by the API when the resource is created.

## Reading resources in a project

Data sources accept the same attribute. Use the project the resource belongs to; a resource in a project is not found when read from the default project.

```terraform
data "mgc_kubernetes_cluster" "main" {
  project_id = var.project_id
  id         = "ac9709ae-b298-41c6-bc58-92a6b8b92d8b"
}
```

## Importing resources from a project

### With an import ID

Prefix the ID with `project/{project_id}/`. An ID without the prefix is looked up in the default project.

```shell
terraform import mgc_kubernetes_cluster.main project/550e8400-e29b-41d4-a716-446655440000/ac9709ae-b298-41c6-bc58-92a6b8b92d8b
```

Resources imported with a composite ID keep their format after the prefix:

| Resource                            | Import ID                                          |
| ----------------------------------- | -------------------------------------------------- |
| `mgc_kubernetes_nodepool`           | `project/{project_id}/{cluster_id},{node_pool_id}` |
| `mgc_block_storage_schedule_attach` | `project/{project_id}/{schedule_id},{volume_id}`   |

### With an import block and identity

Terraform 1.12 and later can import using the resource identity:

```terraform
import {
  to = mgc_kubernetes_cluster.main
  identity = {
    id         = "ac9709ae-b298-41c6-bc58-92a6b8b92d8b"
    project_id = "550e8400-e29b-41d4-a716-446655440000"
  }
}
```

-> After importing, `project_id` can be left out of the configuration: the resource keeps the project it was imported from. Declaring a different `project_id` replaces the resource.

## Troubleshooting

When a resource is not found, the error names the project that was used. If a resource exists but is reported as missing, check that `project_id` matches the project it belongs to, or that the import ID includes the `project/{project_id}/` prefix.
