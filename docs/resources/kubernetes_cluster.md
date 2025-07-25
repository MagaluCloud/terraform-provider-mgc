---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "mgc_kubernetes_cluster Resource - terraform-provider-mgc"
subcategory: "Kubernetes"
description: |-
  Kubernetes cluster resource in MGC
---

# mgc_kubernetes_cluster (Resource)

Kubernetes cluster resource in MGC

## Example Usage

```terraform
resource "mgc_kubernetes_cluster" "cluster" {
  name                 = "my-cluster"
  version              = mgc_kubernetes_version.versions[0].version
  enabled_server_group = false
  description          = "Cluster Example"
}
```

<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `name` (String) Kubernetes cluster name. Must be unique within a namespace and follow naming rules.

### Optional

> **NOTE**: [Write-only arguments](https://developer.hashicorp.com/terraform/language/resources/ephemeral#write-only-arguments) are supported in Terraform 1.11 and later.

- `allowed_cidrs` (List of String) List of allowed CIDR blocks for API server access.
- `cluster_ipv4_cidr` (String) The IP address range of the Kubernetes cluster.
- `description` (String) A brief description of the Kubernetes cluster.
- `enabled_server_group` (Boolean, [Write-only](https://developer.hashicorp.com/terraform/language/resources/ephemeral#write-only-arguments)) Enables the use of a server group with anti-affinity policy during the creation of the cluster and its node pools. Default is true.
- `services_ipv4_cidr` (String) The IP address range of the Kubernetes cluster service.
- `version` (String) The native Kubernetes version of the cluster. Use the standard "vX.Y.Z" format.

### Read-Only

- `created_at` (String) Creation date of the Kubernetes cluster.
- `id` (String) Cluster's UUID.

## Import

Import is supported using the following syntax:

The [`terraform import` command](https://developer.hashicorp.com/terraform/cli/commands/import) can be used, for example:

```shell
terraform import mgc_kubernetes_cluster.cluster 123
```
