---
page_title: "Creating VMs with Public IPs"
subcategory: "Guides"
description: "A comprehensive guide showing easy methods to give virtual machines public IP addresses in Magalu Cloud using Terraform, with simplified primary interface management."
---

# Easy Guide to Creating VMs with Public IPs in Magalu Cloud

This guide shows you the easiest ways to configure your VM networking and assign public IP addresses in Magalu Cloud.

# Introduction

When you create a virtual machine in Magalu Cloud, it comes with a primary network interface that includes a private IPv4 address and a global routable IPv6 address. This guide shows you how to assign public IP addresses and configure your VM's networking using Terraform.

## Understanding VM Network Interfaces

Every VM in Magalu Cloud automatically comes with a primary network interface. Here's what you need to know:

1. **Primary Interface**: When a VM is created, it automatically gets a primary interface with a local IPv4 address and a global routable IPv6 address
2. **Easy Access**: Read-only attributes make it simple to get primary interface information:
   - `network_interface_id`: The ID of the primary interface
   - `local_ipv4`: The private IPv4 address of the primary interface
   - `ipv6`: The IPv6 address of the primary interface (if configured)
   - `ipv4`: The public IPv4 address of the primary interface (if assigned)

3. **Multiple Interfaces**: You can still attach additional interfaces to your VM if needed

### Example: Accessing Primary Interface Information

```terraform
resource "mgc_virtual_machine_instances" "my_vm" {
  name                 = "my-vm"
  machine_type         = "BV1-1-40"
  image                = "cloud-ubuntu-24.04 LTS"
  ssh_key_name         = "your-ssh-key-name"
  allocate_public_ipv4 = true
}

# Easy access to primary interface information
output "vm_private_ip" {
  value = mgc_virtual_machine_instances.my_vm.local_ipv4
}

output "vm_public_ip" {
  value = mgc_virtual_machine_instances.my_vm.ipv4
}

output "vm_ipv6" {
  value = mgc_virtual_machine_instances.my_vm.ipv6
}

output "network_interface_id" {
  value = mgc_virtual_machine_instances.my_vm.network_interface_id
}
```

## Method 1: Using an Existing Network Interface (Recommended)

This method provides full control and management of your network resources. By creating and managing your own network interface, you have complete visibility and control over all network components.

```terraform
# Create VPC first
resource "mgc_network_vpcs" "custom_vpc" {
  name = "custom-vpc"
}

# Create a custom interface
resource "mgc_network_vpcs_interfaces" "existing_interface" {
  name   = "my-existing-interface"
  vpc_id = mgc_network_vpcs.custom_vpc.id
}

# Attach a public IP to the interface (optional)
resource "mgc_network_public_ips" "interface_public_ip" {
  description = "Public IP for existing interface"
  vpc_id      = mgc_network_vpcs.custom_vpc.id
}

resource "mgc_network_public_ips_attach" "attach_public_ip" {
  public_ip_id = mgc_network_public_ips.interface_public_ip.id
  interface_id = mgc_network_vpcs_interfaces.existing_interface.id
}

# Create VM using the existing interface
# Note: Cannot use vpc_id when network_interface_id is specified
resource "mgc_virtual_machine_instances" "vm_with_existing_interface" {
  name                 = "vm-with-existing-interface"
  machine_type         = "BV1-1-40"
  image                = "cloud-ubuntu-24.04 LTS"
  ssh_key_name         = "your-ssh-key-name"
  network_interface_id = mgc_network_vpcs_interfaces.existing_interface.id
}

# Access the interface information
output "vm_interface_id" {
  value = mgc_virtual_machine_instances.vm_with_existing_interface.network_interface_id
}

output "vm_public_ip" {
  value = mgc_virtual_machine_instances.vm_with_existing_interface.ipv4
}
```

This approach gives you full control over your network resources and allows you to manage them independently of the VM lifecycle.

---

## Method 2: Quick VM with Public IP

The simplest way to create a VM with a public IP address is using the `allocate_public_ipv4` argument.

```terraform
resource "mgc_virtual_machine_instances" "simple_vm" {
  name                 = "simple-vm"
  machine_type         = "BV1-1-40"
  image                = "cloud-ubuntu-24.04 LTS"
  ssh_key_name         = "your-ssh-key-name"
  allocate_public_ipv4 = true
}

# Access the public IP directly
output "vm_public_ip" {
  value = mgc_virtual_machine_instances.simple_vm.ipv4
}

output "vm_private_ip" {
  value = mgc_virtual_machine_instances.simple_vm.local_ipv4
}
```

That's it! Your VM now has a public IP address that you can access directly.

---

## Method 3: VM with Custom Security Groups

Create a VM with custom security groups for the primary interface:

```terraform
# Create a security group
resource "mgc_network_security_groups" "web_sg" {
  name        = "web-security-group"
  description = "Security group for web access"
}

# Add SSH rule
resource "mgc_network_security_groups_rules" "ssh_rule" {
  description       = "Allow SSH access"
  direction         = "ingress"
  ethertype         = "IPv4"
  port_range_min    = 22
  port_range_max    = 22
  protocol          = "tcp"
  remote_ip_prefix  = "0.0.0.0/0"
  security_group_id = mgc_network_security_groups.web_sg.id
}

# Add HTTP rule
resource "mgc_network_security_groups_rules" "http_rule" {
  description       = "Allow HTTP access"
  direction         = "ingress"
  ethertype         = "IPv4"
  port_range_min    = 80
  port_range_max    = 80
  protocol          = "tcp"
  remote_ip_prefix  = "0.0.0.0/0"
  security_group_id = mgc_network_security_groups.web_sg.id
}

# Create VM with custom security groups and public IP
resource "mgc_virtual_machine_instances" "web_vm" {
  name                     = "web-vm"
  machine_type             = "BV1-1-40"
  image                    = "cloud-ubuntu-24.04 LTS"
  ssh_key_name             = "your-ssh-key-name"
  creation_security_groups = [mgc_network_security_groups.web_sg.id]
  allocate_public_ipv4     = true
}

# Easy access to the public IP
output "web_vm_public_ip" {
  value = mgc_virtual_machine_instances.web_vm.ipv4
}
```

---

## Method 4: VM in Custom VPC

Create a VM in a specific VPC with custom networking:

```terraform
# Create VPC
resource "mgc_network_vpcs" "custom_vpc" {
  name = "custom-vpc"
}

# Create subnet pool
resource "mgc_network_subnetpools" "custom_subnetpool" {
  name        = "custom-subnetpool"
  description = "Custom Subnet Pool"
  cidr        = "172.16.0.0/16"
}

# Create subnet
resource "mgc_network_vpcs_subnets" "custom_subnet" {
  cidr_block      = "172.16.1.0/24"
  description     = "Custom Network Subnet"
  dns_nameservers = ["8.8.8.8", "1.1.1.1"]
  name            = "custom-subnet"
  subnetpool_id   = mgc_network_subnetpools.custom_subnetpool.id
  vpc_id          = mgc_network_vpcs.custom_vpc.id
}

# Create VM in the custom VPC
resource "mgc_virtual_machine_instances" "custom_vpc_vm" {
  name                 = "custom-vpc-vm"
  machine_type         = "BV1-1-40"
  image                = "cloud-ubuntu-24.04 LTS"
  ssh_key_name         = "your-ssh-key-name"
  vpc_id               = mgc_network_vpcs.custom_vpc.id
  allocate_public_ipv4 = true
}

# Access the network information
output "custom_vm_public_ip" {
  value = mgc_virtual_machine_instances.custom_vpc_vm.ipv4
}

output "custom_vm_private_ip" {
  value = mgc_virtual_machine_instances.custom_vpc_vm.local_ipv4
}
```

---

## Method 5: Adding Secondary Interfaces

You can still add secondary interfaces to your VM for advanced networking:

```terraform
# Create VM with primary interface
resource "mgc_virtual_machine_instances" "multi_interface_vm" {
  name                 = "multi-interface-vm"
  machine_type         = "BV1-1-40"
  image               = "cloud-ubuntu-24.04 LTS"
  ssh_key_name        = "your-ssh-key-name"
  allocate_public_ipv4 = true
}

# Create secondary interface
resource "mgc_network_vpcs_interfaces" "secondary_interface" {
  name   = "secondary-interface"
  vpc_id = mgc_virtual_machine_instances.multi_interface_vm.vpc_id
}

# Attach secondary interface to VM
resource "mgc_virtual_machine_interface_attach" "attach_secondary" {
  instance_id  = mgc_virtual_machine_instances.multi_interface_vm.id
  interface_id = mgc_network_vpcs_interfaces.secondary_interface.id
}

# Output primary interface information
output "primary_public_ip" {
  value = mgc_virtual_machine_instances.multi_interface_vm.ipv4
}

output "primary_private_ip" {
  value = mgc_virtual_machine_instances.multi_interface_vm.local_ipv4
}

output "network_interface_id" {
  value = mgc_virtual_machine_instances.multi_interface_vm.network_interface_id
}
```

---

## Important Notes:

1. **Simplified Access**: Use the read-only attributes (`ipv4`, `local_ipv4`, `ipv6`) and the readable `network_interface_id` for easy access to primary interface information.

2. **Write-Only Arguments**: The `creation_security_groups` and `allocate_public_ipv4` arguments are write-only. The `network_interface_id` is readable after creation.

3. **Mutually Exclusive Options**:
   - When `network_interface_id` is specified, you cannot use `vpc_id`, `creation_security_groups`, or `allocate_public_ipv4`
   - When using `creation_security_groups` or `allocate_public_ipv4`, you cannot specify `network_interface_id`

4. **Public IPv4 Billing**: When using `allocate_public_ipv4 = true`, remember that the public IP will persist after VM deletion and may incur charges.

5. **Default VPC**: If you don't specify a `vpc_id` or `network_interface_id`, the VM will be created in the default VPC.

6. **Security Groups**: If you don't specify `creation_security_groups`, the default security group of the VPC will be used. For managing security groups after instance creation, use the network resources.

This guide shows you the easiest ways to create VMs with public IPs in Magalu Cloud. The primary interface attributes make network configuration simple and intuitive!
