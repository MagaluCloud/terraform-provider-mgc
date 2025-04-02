---
page_title: "Creating Kubernetes Clusters"
subcategory: "Guides"
description: "A comprehensive guide to creating and managing Kubernetes clusters in Magalu Cloud, including how to add and manage nodepools."
---

# Kubernetes

This guide will walk you through creating and managing Kubernetes clusters in Magalu Cloud, including how to add and manage nodepools.

# API Key scopes required

Before creating clusters in Magalu Cloud via Terraform, make sure the used API Key has the following scopes:

- MKE: Resources creation
- ID Magalu: Manage service accounts

The "Manage service accounts" scope is required by Kubernetes so it can create needed resources in behalf of the user.

For more information on how to create an API Key with such scopes, please check the [API Key tutorial](https://docs.magalu.cloud/docs/devops-tools/api-keys/how-to/other-products/create-api-key).

## Creating a Kubernetes Cluster

Kubernetes clusters in Magalu Cloud provide a managed environment to run your containerized applications.

### Basic Cluster Creation

Here's a simple example to create a basic Kubernetes cluster:

```terraform
resource "mgc_kubernetes_cluster" "basic_cluster" {
  name        = "my-basic-cluster"
  version     = "v1.30.2"  # Specify the Kubernetes version you want
  description = "My first Kubernetes cluster"
}
```

Key parameters:

- `name`: A unique identifier for your cluster
- `version`: The Kubernetes version to deploy
- `description`: A brief description of the cluster's purpose

### Advanced Cluster Creation

For a more feature-rich cluster with additional security options:

```terraform
resource "mgc_kubernetes_cluster" "advanced_cluster" {
  name          = "my-advanced-cluster"
  version       = "v1.30.2"
  description   = "Production-grade Kubernetes cluster"
  allowed_cidrs = ["10.0.0.0/16", "192.168.1.0/24"] # Restrict API server access
}
```

Additional parameters:

- `allowed_cidrs`: Restricts Kubernetes API access to specific IP ranges

### Checking Available Kubernetes Versions

Before creating a cluster, you might want to check which Kubernetes versions are available:

```terraform
# Get available Kubernetes versions
data "mgc_kubernetes_version" "available_versions" {
}

output "k8s_versions" {
  value = data.mgc_kubernetes_version.available_versions.versions
}
```

## Managing Nodepools

Nodepools represent groups of worker nodes with similar configurations in your Kubernetes cluster.

### Creating a Basic Nodepool

After creating your cluster, add a basic nodepool:

```terraform
resource "mgc_kubernetes_nodepool" "basic_nodepool" {
  name        = "standard-nodes"
  cluster_id  = mgc_kubernetes_cluster.basic_cluster.id
  flavor_name = "cloud-k8s.gp1.small"  # Machine type/size
  replicas    = 3                       # Number of nodes
}
```

Key parameters:

- `name`: A descriptive name for the nodepool
- `cluster_id`: The ID of the cluster this nodepool belongs to
- `flavor_name`: The machine type/size for the nodepool nodes
- `replicas`: The number of nodes to create in this pool

### Creating an Autoscaling Nodepool

For workloads with variable demand, create a nodepool with autoscaling capabilities:

```terraform
resource "mgc_kubernetes_nodepool" "autoscaling_nodepool" {
  name         = "autoscale-nodes"
  cluster_id   = mgc_kubernetes_cluster.advanced_cluster.id
  flavor_name  = "cloud-k8s.gp1.medium"
  replicas     = 2           # Initial number of nodes
  min_replicas = 1           # Minimum nodes during low demand
  max_replicas = 5           # Maximum nodes during high demand
}
```

Additional parameters:

- `min_replicas`: Minimum number of nodes the pool can scale down to
- `max_replicas`: Maximum number of nodes the pool can scale up to

### Checking Available Node Flavors

Before creating nodepools, you can check which machine types (flavors) are available:

```terraform
# Get available node flavors
data "mgc_kubernetes_flavor" "available_flavors" {
}

output "node_flavors" {
  value = data.mgc_kubernetes_flavor.available_flavors
}
```

## Complete Example: Cluster with Multiple Nodepools

Here's a comprehensive example showing a production-ready cluster with multiple nodepools for different workloads:

```terraform
# Create the cluster
resource "mgc_kubernetes_cluster" "production_cluster" {
  name          = "production"
  version       = "v1.30.2"  # Check for use latest version
  description   = "Production Kubernetes cluster"
  allowed_cidrs = ["10.0.0.0/16", "192.168.0.0/16"]
}

# System nodepool for infrastructure components
resource "mgc_kubernetes_nodepool" "system_nodepool" {
  name        = "system-nodes"
  cluster_id  = mgc_kubernetes_cluster.production_cluster.id
  flavor_name = "cloud-k8s.gp1.small"
  replicas    = 3  # For high availability
}

# General purpose nodepool with autoscaling
resource "mgc_kubernetes_nodepool" "general_nodepool" {
  name         = "general-nodes"
  cluster_id   = mgc_kubernetes_cluster.production_cluster.id
  flavor_name  = "cloud-k8s.gp1.medium"
  replicas     = 3
  min_replicas = 2
  max_replicas = 10
}

# High-performance nodepool for data processing
resource "mgc_kubernetes_nodepool" "compute_nodepool" {
  name         = "compute-nodes"
  cluster_id   = mgc_kubernetes_cluster.production_cluster.id
  flavor_name  = "cloud-k8s.gp1.large"
  replicas     = 2
  min_replicas = 0  # Scale to zero when not needed
  max_replicas = 8
}

# Output cluster information
output "cluster_id" {
  value = mgc_kubernetes_cluster.production_cluster.id
}

data "mgc_kubernetes_cluster_kubeconfig" "kubeconfig" {
  cluster_id = mgc_kubernetes_cluster.production_cluster.id
}

output "kubeconfig_yaml" {
  value = data.mgc_kubernetes_cluster_kubeconfig.kubeconfig
}
```

## Managing Existing Nodepools

### Scaling a Nodepool

To scale an existing nodepool, simply update the `replicas` value:
The replicas value can be updated to scale the nodepool up or down.
Be aware that replicas values is dynamic when autoscaling is enabled, this means that when a nodepool is created and you perform an action on terraform, it may return a different value than the one you set originally.

```terraform
# Scale up the general nodepool
resource "mgc_kubernetes_nodepool" "general_nodepool" {
  # ... other parameters unchanged
  replicas = 5  # Increased from previous value
}
```

### Updating Autoscaling Parameters

You can modify the autoscaling behavior of a nodepool:

```terraform
# Update autoscaling settings
resource "mgc_kubernetes_nodepool" "general_nodepool" {
  # ... other parameters unchanged
  min_replicas = 3
  max_replicas = 15
}
```

## Importing Existing Resources

If you've created clusters or nodepools outside of Terraform, you can import them:

```bash
# Import a cluster
terraform import mgc_kubernetes_cluster.production my-cluster-id

# Import a nodepool
terraform import mgc_kubernetes_nodepool.general_nodepool my-cluster-id,my-nodepool-id
```
