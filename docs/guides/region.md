---
page_title: "Configure your Terraform Region"
subcategory: "Guides"
description: |-
  How to configure the region in Terraform for Magalu Cloud.
---

# Region Configuration

## Basic Configuration

To configure the provider's region, use the following code in your Terraform configuration:

```hcl
provider "mgc" {
  region="br-se1"
}
```

## Available Regions

| Region Code | Description | Default |
| ----------- | ----------- | ------- |
| br-se1      | Southeast   | Yes     |
| br-ne1      | Northeast   | No      |

## Provider Parameters

| Parameter | Required | Description                                                                         |
| --------- | -------- | ----------------------------------------------------------------------------------- |
| region    | no       | The region code where resources will be created. See Available Regions table above. |

## Multi-Region Example

The following example demonstrates how to configure multiple regions using provider aliases. This is useful when you need to manage resources across different regions in the same Terraform configuration:

```hcl
terraform {
  required_providers {
    mgc = {
      source = "magalucloud/mgc"
    }
  }
}

# Southeast region provider
provider "mgc" {
  alias = "southeast"
  region = "br-se1"
}

# Northeast region provider
provider "mgc" {
  alias  = "northeast"
  region = "br-ne1"
}

# Create a VPC in Northeast region
resource "mgc_network_vpcs" "main_vpc_ne" {
    provider = mgc.northeast
    name = "main-vpc-test-tf-tests"
}

# Create a VPC in Southeast region
resource "mgc_network_vpcs" "main_vpc_se" {
    provider = mgc.southeast
    name = "main-vpc-test-tf-tests"
}
```

~> **Note:** When using multiple providers, make sure to specify the provider using the `provider` argument in your resource blocks.
