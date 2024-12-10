---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "mgc_network_vpcs_subnets Resource - terraform-provider-mgc"
subcategory: "Network"
description: |-
  Network VPC Subnet
---

# mgc_network_vpcs_subnets (Resource)

Network VPC Subnet

## Example Usage

```terraform
resource "mgc_network_vpcs_subnets" "example" {
  cidr_block      = "10.0.0.0/16"  
  description     = "Example Subnet"
  dns_nameservers = ["8.8.8.8", "8.8.4.4"] 
  ip_version      = "IPv4"  
  name            = "example-subnet"  
  subnetpool_id   = "subnetpool-12345" 
  vpc_id          = mgc_network_vpcs.example.id  
}
```

<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `cidr_block` (String) The CIDR block of the VPC subnet. Example: '192.168.1.0/24', '0.0.0.0/0', '::/0' or '2001:db8::/32'
- `ip_version` (String) Network protocol version. Allowed values: 'IPv4' or 'IPv6'. Example: 'IPv4'
- `name` (String) The name of the VPC subnet
- `subnetpool_id` (String) The ID of the subnet pool
- `vpc_id` (String) The ID of the VPC

### Optional

- `description` (String) The description of the VPC subnet
- `dns_nameservers` (List of String) The DNS nameservers of the VPC subnet

### Read-Only

- `id` (String) The ID of the VPC subnet

## Import

Import is supported using the following syntax:

```shell
terraform import mgc_network_vpcs_subnets.example 123
```