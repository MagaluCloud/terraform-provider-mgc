---
page_title: "Configure your Terraform Region"
subcategory: "Guides"
description: |-
  How to configure the region in Terraform for Magalu Cloud.
---

# Understanding Regions and Availability Zones in Magalu Cloud

This guide explains how to effectively use regions and availability zones in Magalu Cloud to build resilient, high-performance applications.

## Regions in Magalu Cloud

Regions are geographic areas where Magalu Cloud operates its data centers. Each region is composed of multiple physically separated data centers, providing redundancy and resilience.

### Available Regions

| Region Code | Description | Default |
| ----------- | ----------- | ------- |
| br-se1      | Southeast   | Yes     |
| br-ne1      | Northeast   | No      |

### Configuring Regions in Terraform

To specify which region you want to deploy resources to:

```terraform
provider "mgc" {
  region = "br-se1"  # Southeast region
}
```

### Working with Multiple Regions

You can manage resources across different regions using provider aliases:

```terraform
# Southeast region provider (default)
provider "mgc" {
  alias  = "southeast"
  region = "br-se1"
}

# Northeast region provider
provider "mgc" {
  alias  = "northeast"
  region = "br-ne1"
}

# Create resources in specific regions
resource "mgc_network_vpcs" "vpc_southeast" {
  provider = mgc.southeast
  name     = "vpc-southeast"
}

resource "mgc_network_vpcs" "vpc_northeast" {
  provider = mgc.northeast
  name     = "vpc-northeast"
}
```

## Availability Zones (AZs)

Availability Zones are distinct locations within a region that are engineered to be isolated from failures in other zones. By deploying resources across multiple AZs, you can build highly available applications.

### AZ Naming Convention

Availability Zones follow the pattern: `country-region-zone`

For example:

- `br-se1-a` represents Brazil, Southeast region 1, zone a
- `br-se1-b` represents Brazil, Southeast region 1, zone b
- `br-ne1-a` represents Brazil, Northeast region 1, zone a

### Discovering Available Zones

You can use a data source to discover available zones:

```terraform
data "mgc_availability_zones" "available" {
}

output "available_zones" {
  value = data.mgc_availability_zones.available.regions
}
```

### Specifying Availability Zones for Resources

Many resources in Magalu Cloud allow you to specify which availability zone they should be deployed to:

```terraform
# Create a VM in a specific availability zone
resource "mgc_virtual_machine_instances" "web_server" {
  name              = "web-server"
  availability_zone = "br-se1-a"
  machine_type      = "BV1-1-40"
  image             = "cloud-ubuntu-24.04 LTS"
  ssh_key_name      = "your-ssh-key"
}

# Create a block storage volume in the same AZ
resource "mgc_block_storage_volumes" "web_data" {
  name              = "web-data"
  availability_zone = "br-se1-a"  # Must match the VM's zone
  size              = 100
  type              = "cloud_nvme1k"
}
```

## Benefits of Using Multiple Availability Zones

### High Availability

By distributing your application across multiple AZs, you can ensure it remains available even if an entire zone experiences an outage.

### Disaster Recovery

AZs provide physical isolation, protecting your resources from localized disasters like power outages, network issues, or natural events.

### Load Distribution

Distributing workloads across multiple AZs helps balance traffic and prevents any single zone from becoming overloaded.
