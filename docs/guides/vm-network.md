---
page_title: "Creating VMs with Public IPs"
subcategory: "Guides"
description: "A comprehensive guide showing two easy methods to give virtual machines public IP addresses in Magalu Cloud using Terraform - using the default interface or creating a custom interface."
---

# Easy Guide to Creating VMs with Public IPs in Magalu Cloud

There are two simple ways to give your virtual machines public IP addresses in Magalu Cloud. Let's explore both methods with clear, easy-to-follow steps.

# Introduction

When you create a virtual machine in Magalu Cloud, it comes with a default network interface that includes a private IP address. This guide shows you how to assign a public IP address to your VM using Terraform.

## Understanding VM Network Interfaces

Every VM in Magalu Cloud automatically comes with a `network_interfaces` attribute, which is a list of interfaces attached to the VM. Here's what you need to know:

1. **Default Primary Interface**: When a VM is created, it automatically gets a primary interface with a local IPv4 address
2. **Interface Structure**: Each interface in the list contains these attributes:

   - `id`: Unique identifier for the interface
   - `name`: The name of the interface
   - `ipv4`: Public IPv4 address (if a public IP is attached)
   - `local_ipv4`: Private IPv4 address within the VPC
   - `ipv6`: IPv6 address (if configured)
   - `primary`: Boolean flag indicating if this is the primary interface

3. **Accessing Interface Properties**: You need to iterate through the interfaces list to access properties

4. **Multiple Interfaces**: If you attach additional interfaces to your VM, they'll all appear in the `network_interfaces` list, each with their own properties

5. **Tracking Public IPs**: When you attach a public IP to an interface, the `ipv4` field of that interface will be updated with the public IP address

### Example: Getting the Primary Interface's Local IP

```terraform
locals {
  primary_local_ip = [
    for interface in mgc_virtual_machine_instances.my_vm.network_interfaces :
    interface.local_ipv4 if interface.primary
  ][0]
}

output "vm_private_ip" {
  value = local.primary_local_ip
}
```

### Example: Getting the Public IP (if attached)

```terraform
locals {
  primary_public_ip = [
    for interface in mgc_virtual_machine_instances.my_vm.network_interfaces :
    interface.ipv4 if interface.primary
  ][0]
}

output "vm_public_ip" {
  value = local.primary_public_ip
}
```

## Method 1: Using the Default Interface (The Simplest Approach)

Every VM in Magalu Cloud automatically comes with a default network interface. This method uses that existing interface.

### Step 1: Create a Virtual Machine

```terraform
resource "mgc_virtual_machine_instances" "simple_vm" {
  name         = "simple-vm"
  machine_type = "BV1-1-40"
  image        = "cloud-ubuntu-24.04 LTS"
  ssh_key_name = "your-ssh-key-name"
}
```

### Step 2: Create a Public IP

```terraform
resource "mgc_network_public_ips" "vm_public_ip" {
  description = "Public IP for my VM"
  vpc_id      = "your-vpc-id"  # Use your existing VPC ID
}
```

### Step 3: Find the Default Interface and Attach the Public IP

```terraform
# Get the primary interface ID
locals {
  primary_interface_id = [
    for interface in mgc_virtual_machine_instances.simple_vm.network_interfaces :
    interface.id if interface.primary
  ][0]
}

# Attach the public IP to the default interface
resource "mgc_network_public_ips_attach" "attach_to_default" {
  public_ip_id = mgc_network_public_ips.vm_public_ip.id
  interface_id = local.primary_interface_id
}
```

That's it! Your VM now has a public IP attached to its default interface.

---

## Method 2: Creating a Custom Interface

This method involves creating a new network interface and attaching it to your VM.

### Step 1: Set Up Network Resources

First, create a VPC, subnet pool, and subnet:

```terraform
resource "mgc_network_vpcs" "my_vpc" {
  name = "my-vpc"
}

resource "mgc_network_subnetpools" "my_pool" {
  name        = "my-subnet-pool"
  description = "My IP address pool"
  cidr        = "172.29.0.0/16"
}

resource "mgc_network_vpcs_subnets" "my_subnet" {
  cidr_block      = "172.29.1.0/24"
  name            = "my-subnet"
  dns_nameservers = ["8.8.8.8", "8.8.4.4"]
  subnetpool_id   = mgc_network_subnetpools.my_pool.id
  vpc_id          = mgc_network_vpcs.my_vpc.id
}
```

### Step 2: Create a Custom Network Interface

```terraform
resource "mgc_network_vpcs_interfaces" "custom_interface" {
  name   = "my-custom-interface"
  vpc_id = mgc_network_vpcs.my_vpc.id

  # Important: Wait for subnet to be created before creating the interface
  depends_on = [mgc_network_vpcs_subnets.my_subnet]
}
```

### Step 3: Create a Public IP

```terraform
resource "mgc_network_public_ips" "vm_public_ip" {
  description = "Public IP for my VM"
  vpc_id      = mgc_network_vpcs.my_vpc.id
}
```

### Step 4: Attach the Public IP to the Custom Interface

```terraform
resource "mgc_network_public_ips_attach" "interface_ip_attach" {
  public_ip_id = mgc_network_public_ips.vm_public_ip.id
  interface_id = mgc_network_vpcs_interfaces.custom_interface.id
}
```

### Step 5: Create a Virtual Machine

```terraform
resource "mgc_virtual_machine_instances" "custom_vm" {
  name         = "custom-vm"
  machine_type = "BV1-1-40"
  image        = "cloud-ubuntu-24.04 LTS"
  ssh_key_name = "your-ssh-key-name"
}
```

### Step 6: Attach the Custom Interface to the VM

```terraform
resource "mgc_virtual_machine_interface_attach" "attach_interface" {
  instance_id  = mgc_virtual_machine_instances.custom_vm.id
  interface_id = mgc_network_vpcs_interfaces.custom_interface.id
}
```

Now your VM has a custom interface with a public IP attached!

---

## Which Method Should You Choose?

- **Method 1 (Default Interface)**: Quickest and easiest if you already have a VPC. Great for simple setups and testing.

- **Method 2 (Custom Interface)**: Provides more control and flexibility. Ideal for production environments where you need custom network configurations.

## Complete Example: Using the Default Interface

```terraform
# Create a VM with default networking
resource "mgc_virtual_machine_instances" "simple_vm" {
  name         = "simple-vm"
  machine_type = "BV1-1-40"
  image        = "cloud-ubuntu-24.04 LTS"
  ssh_key_name = "your-ssh-key-name"
}

# Create a public IP
resource "mgc_network_public_ips" "vm_public_ip" {
  description = "Public IP for my VM"
  vpc_id      = "your-existing-vpc-id"
}

# Find the default interface and attach the public IP
locals {
  primary_interface_id = [
    for interface in mgc_virtual_machine_instances.simple_vm.network_interfaces :
    interface.id if interface.primary
  ][0]
}

resource "mgc_network_public_ips_attach" "attach_to_default" {
  public_ip_id = mgc_network_public_ips.vm_public_ip.id
  interface_id = local.primary_interface_id
}

# Output the public IP
output "vm_public_ip" {
  value = mgc_network_public_ips.vm_public_ip.public_ip
}
```

## Complete Example: Creating a Custom Interface with Security Groups

```terraform
# Create VPC
resource "mgc_network_vpcs" "main_vpc" {
  name = "main-vpc"
}

# Create subnet pool
resource "mgc_network_subnetpools" "main_subnetpool" {
  name        = "main-subnetpool"
  description = "Main Subnet Pool"
  cidr        = "172.5.0.0/16"
}

# Create subnet
resource "mgc_network_vpcs_subnets" "primary_subnet" {
  cidr_block      = "172.5.1.0/24"
  description     = "Primary Network Subnet"
  dns_nameservers = ["8.8.8.8", "1.1.1.1"]
  ip_version      = "IPv4"
  name            = "primary-subnet"
  subnetpool_id   = mgc_network_subnetpools.main_subnetpool.id
  vpc_id          = mgc_network_vpcs.main_vpc.id
}

# Create security group
resource "mgc_network_security_groups" "vm_sg" {
  name        = "vm-security-group"
  description = "Security group for VM access"
}

# Add SSH rule to security group
resource "mgc_network_security_groups_rules" "ssh_rule" {
  description       = "Allow SSH access"
  direction         = "ingress"
  ethertype         = "IPv4"
  port_range_min    = 22
  port_range_max    = 22
  protocol          = "tcp"
  remote_ip_prefix  = "0.0.0.0/0"
  security_group_id = mgc_network_security_groups.vm_sg.id
}

# Create network interface
resource "mgc_network_vpcs_interfaces" "vm_interface" {
  name   = "vm-interface"
  vpc_id = mgc_network_vpcs.main_vpc.id

  # Important: Wait for subnet to be created
  depends_on = [mgc_network_vpcs_subnets.primary_subnet]
}

# Attach security group to interface
resource "mgc_network_security_groups_attach" "sg_attachment" {
  security_group_id = mgc_network_security_groups.vm_sg.id
  interface_id      = mgc_network_vpcs_interfaces.vm_interface.id
}

# Create public IP
resource "mgc_network_public_ips" "vm_public_ip" {
  description = "VM public IP"
  vpc_id      = mgc_network_vpcs.main_vpc.id
}

# Attach public IP to interface
resource "mgc_network_public_ips_attach" "ip_attachment" {
  public_ip_id = mgc_network_public_ips.vm_public_ip.id
  interface_id = mgc_network_vpcs_interfaces.vm_interface.id
}

# Create VM
resource "mgc_virtual_machine_instances" "custom_vm" {
  name         = "custom-vm"
  machine_type = "BV1-1-40"
  image        = "cloud-ubuntu-24.04 LTS"
  ssh_key_name = "your-ssh-key-name"
}

# Attach interface to VM
resource "mgc_virtual_machine_interface_attach" "interface_attachment" {
  instance_id  = mgc_virtual_machine_instances.custom_vm.id
  interface_id = mgc_network_vpcs_interfaces.vm_interface.id
}

# Output the public IP
output "vm_public_ip" {
  value = mgc_network_public_ips.vm_public_ip.public_ip
}
```

## Important Notes:

1. **Dependencies Matter**: Always add `depends_on` for the interface to depend on the subnet, as shown in the examples.

2. **Security Groups**: Add security groups to control access to your VM - the second complete example shows how to add SSH access.

3. **Subnet Pools**: A subnet pool with the proper CIDR range is required before creating subnets.

4. **VPC Consistency**: Keep all resources (subnets, interfaces, public IPs) in the same VPC.

That's all you need to get started with public IPs for your VMs in Magalu Clou!
