# Data source to get the latest Kubernetes version
variable "cluster-version" {
  default = "v1.32.3"
}

variable "cluster-flavor" {
  default = "cloud-k8s.gp1.small"
}

resource "random_pet" "name" {
  length    = 1
  separator = "-"
}

# Basic Kubernetes cluster
resource "mgc_kubernetes_cluster" "basic_cluster" {
  name        = "${random_pet.name.id}-basic-cluster"
  version     = var.cluster-version
  description = "Basic Kubernetes cluster for smoke test"
}

data "mgc_kubernetes_cluster" "basic_cluster_data" {
  id = mgc_kubernetes_cluster.basic_cluster.id
}

# # Full-featured Kubernetes cluster
# resource "mgc_kubernetes_cluster" "full_cluster" {
#   name                 = "${random_pet.name.id}-full-cluster"
#   version              = var.cluster-version
#   description          = "Full-featured Kubernetes cluster for smoke test"
#   enabled_server_group = true
#   allowed_cidrs        = ["10.0.0.0/24", "192.168.1.0/24"]
# }

# # Outputs for verification
# output "basic_cluster" {
#   value = mgc_kubernetes_cluster.basic_cluster
# }

# output "full_cluster" {
#   value = mgc_kubernetes_cluster.full_cluster
# }

# data "mgc_kubernetes_flavor" "available_flavors" {
# }

# output "flavor_list" {
#   value = data.mgc_kubernetes_flavor.available_flavors
# }

# data "mgc_kubernetes_version" "latest" {
# }

# output "versions_list" {
#   value = data.mgc_kubernetes_version.latest.versions
# }

# Basic nodepool for the basic cluster
resource "mgc_kubernetes_nodepool" "basic_nodepool" {
  name        = "basic-nodepool"
  cluster_id  = mgc_kubernetes_cluster.basic_cluster.id
  flavor_name = var.cluster-flavor
  replicas    = 1
}

data "mgc_kubernetes_nodepool" "basic_nodepool_data" {
  id         = mgc_kubernetes_nodepool.basic_nodepool.id
  cluster_id = mgc_kubernetes_cluster.basic_cluster.id
}

# # Full-featured nodepool for the full cluster
# resource "mgc_kubernetes_nodepool" "full_nodepool" {
#   name         = "full-nodepool"
#   cluster_id   = mgc_kubernetes_cluster.full_cluster.id
#   flavor_name  = var.cluster-flavor
#   replicas     = 1
#   min_replicas = 1
#   max_replicas = 5

#   taints = [
#     {
#       key    = "dedicated"
#       value  = "smoke-test"
#       effect = "NoSchedule"
#     }
#   ]
# }

# output "basic_nodepool" {
#   value = mgc_kubernetes_nodepool.basic_nodepool
# }

# output "full_nodepool" {
#   value = mgc_kubernetes_nodepool.full_nodepool
# }
