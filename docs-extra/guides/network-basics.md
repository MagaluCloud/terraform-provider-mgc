---
page_title: "Networking in Magalu Cloud"
subcategory: "Guides"
description: "A comprehensive guide to understanding and implementing networking in Magalu Cloud, including VPCs, subnets, interfaces, and security groups."
---

# Comprehensive Guide to Magalu Cloud Networking

This guide will help you understand and implement networking in Magalu Cloud, explaining how the various network resources work together to create secure, flexible network architectures.

## Core Networking Concepts in Magalu Cloud

Magalu Cloud's networking follows a hierarchical model that allows you to create isolated networks with fine-grained security controls:

1. **VPCs (Virtual Private Clouds)**: Isolated virtual networks
2. **Subnet Pools**: Collections of IP addresses that can be allocated to subnets
3. **Subnets**: Segmented network address spaces within a VPC
4. **Interfaces**: Connection points for Virtual Machines to a network
   - Also known as NICs (Network Interface Cards) and Ports
5. **Security Groups**: Virtual firewalls for controlling inbound and outbound traffic
6. **Public IPs**: Addresses for connecting resources to the internet

Let's explore how these components work together.

## Creating a Basic Network Infrastructure

### Step 1: Create a VPC

First, create a Virtual Private Cloud (VPC) - an isolated network environment:

```terraform
resource "mgc_network_vpcs" "main_vpc" {
  name        = "main-vpc"
  description = "Main production network"
}
```

### Step 2: Create a Subnet Pool

Subnet pools define the IP addresses that can be used within your VPC:

```terraform
resource "mgc_network_subnetpools" "main_subnetpool" {
  name        = "main-subnetpool"
  description = "Main subnet pool for production"
  type        = "pip"  # "pip" for Public IP allocation
  cidr        = "172.16.0.0/16"  # The address space for your network
}
```

The `type = "pip"` parameter indicates this subnet pool can be used for allocating public IPs.

### Step 3: Create a Subnet

Now create a subnet within your VPC using the subnet pool:

```terraform
resource "mgc_network_vpcs_subnets" "web_subnet" {
  name            = "web-subnet"
  description     = "Subnet for web servers"
  vpc_id          = mgc_network_vpcs.main_vpc.id
  subnetpool_id   = mgc_network_subnetpools.main_subnetpool.id
  cidr_block      = "172.16.1.0/24"  # A subset of the subnet pool CIDR
  ip_version      = "IPv4"
  dns_nameservers = ["8.8.8.8", "8.8.4.4"]  # Google DNS servers
}
```

This creates a subnet with the specified CIDR range and DNS servers.

## Working with Network Interfaces

### Creating a Network Interface

Network interfaces connect resources like virtual machines to your network:

```terraform
resource "mgc_network_vpcs_interfaces" "web_interface" {
  name   = "web-server-interface"
  vpc_id = mgc_network_vpcs.main_vpc.id

  # Important: Wait for the subnet to be created first
  depends_on = [mgc_network_vpcs_subnets.web_subnet]
}
```

Note the `depends_on` attribute ensures that the subnet exists before creating the interface.

## Implementing Network Security

### Creating a Security Group

Security groups act as virtual firewalls:

```terraform
resource "mgc_network_security_groups" "web_sg" {
  name                  = "web-security-group"
  description           = "Security group for web servers"
  disable_default_rules = false  # Use default rules
}
```

### Adding Security Rules

Define traffic rules for your security group:

```terraform
# Allow HTTP traffic
resource "mgc_network_security_groups_rules" "allow_http" {
  description       = "Allow incoming HTTP traffic"
  direction         = "ingress"
  ethertype         = "IPv4"
  port_range_min    = 80
  port_range_max    = 80
  protocol          = "tcp"
  remote_ip_prefix  = "0.0.0.0/0"  # Allow from anywhere
  security_group_id = mgc_network_security_groups.web_sg.id
}

# Allow HTTPS traffic
resource "mgc_network_security_groups_rules" "allow_https" {
  description       = "Allow incoming HTTPS traffic"
  direction         = "ingress"
  ethertype         = "IPv4"
  port_range_min    = 443
  port_range_max    = 443
  protocol          = "tcp"
  remote_ip_prefix  = "0.0.0.0/0"  # Allow from anywhere
  security_group_id = mgc_network_security_groups.web_sg.id
}

# Allow SSH from specific admin network
resource "mgc_network_security_groups_rules" "allow_ssh" {
  description       = "Allow SSH from admin network"
  direction         = "ingress"
  ethertype         = "IPv4"
  port_range_min    = 22
  port_range_max    = 22
  protocol          = "tcp"
  remote_ip_prefix  = "10.0.0.0/8"  # Admin network
  security_group_id = mgc_network_security_groups.web_sg.id
}
```

### Applying Security Groups to Interfaces

Attach the security group to your network interface:

```terraform
resource "mgc_network_security_groups_attach" "web_sg_attachment" {
  security_group_id = mgc_network_security_groups.web_sg.id
  interface_id      = mgc_network_vpcs_interfaces.web_interface.id
}
```

## Managing Public IP Addresses

### Creating a Public IP

To connect a resource to the internet, create a public IP:

```terraform
resource "mgc_network_public_ips" "web_public_ip" {
  description = "Public IP for web server"
  vpc_id      = mgc_network_vpcs.main_vpc.id
}
```

### Attaching a Public IP to an Interface

Connect the public IP to your network interface:

```terraform
resource "mgc_network_public_ips_attach" "web_ip_attachment" {
  public_ip_id = mgc_network_public_ips.web_public_ip.id
  interface_id = mgc_network_vpcs_interfaces.web_interface.id
}
```

## Complete Network Architecture Example

Here's a comprehensive example showing all networking components working together:

```terraform
# 1. Create VPC
resource "mgc_network_vpcs" "app_vpc" {
  name        = "application-vpc"
  description = "VPC for application servers"
}

# 2. Create subnet pool
resource "mgc_network_subnetpools" "app_subnet_pool" {
  name        = "app-subnet-pool"
  description = "Subnet pool for application networks"
  type        = "pip"
  cidr        = "10.0.0.0/16"
}

# 3. Create subnets
resource "mgc_network_vpcs_subnets" "web_subnet" {
  name            = "web-subnet"
  description     = "Subnet for web tier"
  vpc_id          = mgc_network_vpcs.app_vpc.id
  subnetpool_id   = mgc_network_subnetpools.app_subnet_pool.id
  cidr_block      = "10.0.1.0/24"
  ip_version      = "IPv4"
  dns_nameservers = ["8.8.8.8", "1.1.1.1"]
}

resource "mgc_network_vpcs_subnets" "app_subnet" {
  name            = "app-subnet"
  description     = "Subnet for application tier"
  vpc_id          = mgc_network_vpcs.app_vpc.id
  subnetpool_id   = mgc_network_subnetpools.app_subnet_pool.id
  cidr_block      = "10.0.2.0/24"
  ip_version      = "IPv4"
  dns_nameservers = ["8.8.8.8", "1.1.1.1"]
}

resource "mgc_network_vpcs_subnets" "db_subnet" {
  name            = "db-subnet"
  description     = "Subnet for database tier"
  vpc_id          = mgc_network_vpcs.app_vpc.id
  subnetpool_id   = mgc_network_subnetpools.app_subnet_pool.id
  cidr_block      = "10.0.3.0/24"
  ip_version      = "IPv4"
  dns_nameservers = ["8.8.8.8", "1.1.1.1"]
}

# 4. Create security groups
resource "mgc_network_security_groups" "web_sg" {
  name        = "web-security-group"
  description = "Security group for web servers"
}

resource "mgc_network_security_groups" "app_sg" {
  name        = "app-security-group"
  description = "Security group for application servers"
}

resource "mgc_network_security_groups" "db_sg" {
  name        = "db-security-group"
  description = "Security group for database servers"
}

# 5. Create security group rules
# Web tier rules
resource "mgc_network_security_groups_rules" "web_http" {
  description       = "Allow HTTP traffic"
  direction         = "ingress"
  ethertype         = "IPv4"
  port_range_min    = 80
  port_range_max    = 80
  protocol          = "tcp"
  remote_ip_prefix  = "0.0.0.0/0"
  security_group_id = mgc_network_security_groups.web_sg.id
}

resource "mgc_network_security_groups_rules" "web_https" {
  description       = "Allow HTTPS traffic"
  direction         = "ingress"
  ethertype         = "IPv4"
  port_range_min    = 443
  port_range_max    = 443
  protocol          = "tcp"
  remote_ip_prefix  = "0.0.0.0/0"
  security_group_id = mgc_network_security_groups.web_sg.id
}

# App tier rules - only allow traffic from web tier
resource "mgc_network_security_groups_rules" "app_from_web" {
  description       = "Allow traffic from web tier"
  direction         = "ingress"
  ethertype         = "IPv4"
  port_range_min    = 8080
  port_range_max    = 8080
  protocol          = "tcp"
  remote_ip_prefix  = "10.0.1.0/24"  # Web subnet CIDR
  security_group_id = mgc_network_security_groups.app_sg.id
}

# DB tier rules - only allow traffic from app tier
resource "mgc_network_security_groups_rules" "db_from_app" {
  description       = "Allow traffic from app tier"
  direction         = "ingress"
  ethertype         = "IPv4"
  port_range_min    = 5432  # PostgreSQL
  port_range_max    = 5432
  protocol          = "tcp"
  remote_ip_prefix  = "10.0.2.0/24"  # App subnet CIDR
  security_group_id = mgc_network_security_groups.db_sg.id
}

# 6. Create network interfaces
resource "mgc_network_vpcs_interfaces" "web_interface" {
  name   = "web-interface"
  vpc_id = mgc_network_vpcs.app_vpc.id

  depends_on = [mgc_network_vpcs_subnets.web_subnet]
}

resource "mgc_network_vpcs_interfaces" "app_interface" {
  name   = "app-interface"
  vpc_id = mgc_network_vpcs.app_vpc.id

  depends_on = [mgc_network_vpcs_subnets.app_subnet]
}

resource "mgc_network_vpcs_interfaces" "db_interface" {
  name   = "db-interface"
  vpc_id = mgc_network_vpcs.app_vpc.id

  depends_on = [mgc_network_vpcs_subnets.db_subnet]
}

# 7. Attach security groups to interfaces
resource "mgc_network_security_groups_attach" "web_sg_attach" {
  security_group_id = mgc_network_security_groups.web_sg.id
  interface_id      = mgc_network_vpcs_interfaces.web_interface.id
}

resource "mgc_network_security_groups_attach" "app_sg_attach" {
  security_group_id = mgc_network_security_groups.app_sg.id
  interface_id      = mgc_network_vpcs_interfaces.app_interface.id
}

resource "mgc_network_security_groups_attach" "db_sg_attach" {
  security_group_id = mgc_network_security_groups.db_sg.id
  interface_id      = mgc_network_vpcs_interfaces.db_interface.id
}

# 8. Create and attach public IP for web tier only
resource "mgc_network_public_ips" "web_public_ip" {
  description = "Public IP for web server"
  vpc_id      = mgc_network_vpcs.app_vpc.id
}

resource "mgc_network_public_ips_attach" "web_public_ip_attach" {
  public_ip_id = mgc_network_public_ips.web_public_ip.id
  interface_id = mgc_network_vpcs_interfaces.web_interface.id
}

# Output the public IP
output "web_public_ip" {
  value = mgc_network_public_ips.web_public_ip.public_ip
}
```

## Understanding the Network Resource Hierarchy

The networking components in Magalu Cloud follow a hierarchical relationship:

1. **VPC** is the parent container for all other network resources
2. **Subnet Pool** defines the IP address range available for allocation
3. **Subnet** segments the network within a VPC and uses addresses from the subnet pool
4. **Network Interface** connects to a subnet and provides network connectivity
5. **Security Group** attaches to interfaces to control traffic
6. **Public IP** attaches to interfaces to provide internet connectivity

## Best Practices for Magalu Cloud Networking

1. **Plan Your Address Space**: Carefully plan your IP address allocation to avoid overlapping or running out of addresses

2. **Security in Depth**:

   - Use security groups at each tier
   - Only allow traffic from necessary sources
   - Define specific port ranges instead of allowing all ports

3. **Network Segmentation**:

   - Create separate subnets for different application tiers
   - Isolate sensitive workloads in dedicated subnets

4. **Minimize Public IPs**:

   - Only expose services that need to be public
   - Consider using a load balancer as the single public entry point

5. **Dependencies Matter**:

   - Always use `depends_on` for interfaces to depend on the subnet
   - Create resources in the correct order to avoid dependency issues

6. **Documentation**:
   - Use descriptive names and descriptions for all resources
   - Document CIDR ranges and their purposes

## Troubleshooting Network Connectivity

If you encounter connectivity issues:

1. **Check Security Groups**: Ensure the security group rules allow traffic on the required ports
2. **Verify Interface Attachments**: Confirm interfaces are properly attached to resources
3. **Public IP Verification**: Ensure public IPs are correctly attached to interfaces
4. **CIDR Overlap**: Check for overlapping CIDR blocks in your subnet configurations
5. **Resource Dependencies**: Verify resources are created in the correct order
