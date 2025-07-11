resource "random_pet" "name" {
  length    = 1
  separator = "-"
}

# Security Groups
resource "mgc_network_security_groups" "primary_sg" {
  name                  = "${random_pet.name.id}-primary-security-group-tf2"
  description           = "Primary security group for main services"
  disable_default_rules = true
}

resource "mgc_network_security_groups" "secondary_sg" {
  name                  = "${random_pet.name.id}-secondary-security-group2"
  disable_default_rules = true
}

resource "mgc_network_security_groups" "auxiliary_sg" {
  name = "${random_pet.name.id}-auxiliary-security-group2"
}

data "mgc_network_security_group" "primary_sg_data" {
  id = mgc_network_security_groups.primary_sg.id
}

data "mgc_network_security_groups" "security_groups" {}

# Security Group Rules
resource "mgc_network_security_groups_rules" "ssh_ipv4_rule" {
  description       = "Allow incoming SSH traffic"
  direction         = "ingress"
  ethertype         = "IPv4"
  port_range_max    = 22
  port_range_min    = 22
  protocol          = "tcp"
  remote_ip_prefix  = "192.168.1.0/24"
  security_group_id = mgc_network_security_groups.primary_sg.id
}

resource "mgc_network_security_groups_rules" "ssh_ipv6_rule" {
  description       = "Allow incoming SSH traffic from IPv6"
  direction         = "ingress"
  ethertype         = "IPv6"
  port_range_max    = 22
  port_range_min    = 22
  protocol          = "tcp"
  remote_ip_prefix  = "::/0"
  security_group_id = mgc_network_security_groups.primary_sg.id
}

# VPC Resources
resource "mgc_network_vpcs" "main_vpc" {
  name = "${random_pet.name.id}-main-vpc-test-tf2"
}

data "mgc_network_vpc" "main_vpc_data" {
  id = mgc_network_vpcs.main_vpc.id
}

data "mgc_network_vpcs" "vpcs_data" {}

# VPC Interfaces
resource "mgc_network_vpcs_interfaces" "pip_interface" {
  name       = "${random_pet.name.id}-pip-interface2"
  vpc_id     = data.mgc_network_vpc.main_vpc_data.id
  subnet_ids = [data.mgc_network_vpcs_subnet.primary_subnet_data.id]
  depends_on = [data.mgc_network_vpcs_subnet.primary_subnet_data]
}

data "mgc_network_vpcs_interface" "primary_interface_data" {
  id = mgc_network_vpcs_interfaces.pip_interface.id
}

data "mgc_network_vpcs_interfaces" "vpcs_interfaces_data" {}

# Security Group Attachment
resource "mgc_network_security_groups_attach" "primary_sg_attachment" {
  security_group_id = mgc_network_security_groups.primary_sg.id
  interface_id      = mgc_network_vpcs_interfaces.pip_interface.id
}

#Subnetpools
resource "mgc_network_subnetpools" "main_subnetpool" {
  name        = "${random_pet.name.id}-main-subnetpool2"
  description = "Main Subnet Pool"
  cidr        = "172.5.0.0/16"
}

# Subnet Resources
resource "mgc_network_vpcs_subnets" "primary_subnet" {
  cidr_block      = "172.5.0.0/16"
  description     = "Primary Network Subnet"
  dns_nameservers = ["8.8.8.8", "1.1.1.1"]
  ip_version      = "IPv4"
  name            = "primary-subnet2"
  subnetpool_id   = mgc_network_subnetpools.main_subnetpool.id
  vpc_id          = data.mgc_network_vpc.main_vpc_data.id
}

# resource "mgc_virtual_machine_instances" "tc1_basic_instance" {
#   name              = "${random_pet.name.id}-basic-instance-tf2-with-vpc"
#   availability_zone = "br-ne1-a"
#   machine_type      = "BV1-1-40"
#   image             = "cloud-ubuntu-24.04 LTS"
#   ssh_key_name      = "publio"
#   vpc_id            = data.mgc_network_vpc.main_vpc_data.id
# }

data "mgc_network_vpcs_subnet" "primary_subnet_data" {
  id = mgc_network_vpcs_subnets.primary_subnet.id
}

# Public IP
resource "mgc_network_public_ips" "example" {
  description = "example public ip"
  vpc_id      = data.mgc_network_vpc.main_vpc_data.id
}

data "mgc_network_public_ip" "example" {
  id = mgc_network_public_ips.example.id
}

data "mgc_network_public_ips" "public_ips" {}

data "mgc_network_subnetpool" "subnetpool_data" {
  id = mgc_network_vpcs_subnets.primary_subnet.subnetpool_id
}

data "mgc_network_subnetpools" "subnetpools_data" {}

#Public IP Attachment
resource "mgc_network_public_ips_attach" "example" {
  public_ip_id = mgc_network_public_ips.example.id
  interface_id = mgc_network_vpcs_interfaces.pip_interface.id
}

#NAT Gateway
resource "mgc_network_nat_gateway" "example" {
  name        = "example-nat-gateway2"
  description = "Example NAT Gateway"
  vpc_id      = data.mgc_network_vpc.main_vpc_data.id
  availability_zone = "br-ne1-a"
}

data "mgc_network_nat_gateway" "example" {
  id = mgc_network_nat_gateway.example.id
}

# Outputs
output "primary_security_group_data" {
  value = data.mgc_network_security_group.primary_sg_data
}

output "main_subnetpool_data" {
  value = data.mgc_network_subnetpool.subnetpool_data
}

output "main_vpc_data" {
  value = data.mgc_network_vpc.main_vpc_data
}

output "primary_interface_data" {
  value = data.mgc_network_vpcs_interface.primary_interface_data
}

output "primary_subnet_data" {
  value = data.mgc_network_vpcs_subnet.primary_subnet_data
}

output "datasource_public_ip_id" {
  value = data.mgc_network_public_ip.example
}

output "datasource_sgs" {
  value = data.mgc_network_security_groups.security_groups
}

output "public_ips" {
  value = data.mgc_network_public_ips.public_ips
}

output "subnetpools_data" {
  value = data.mgc_network_subnetpools.subnetpools_data
}

output "vpcs_data" {
  value = data.mgc_network_vpcs.vpcs_data
}

output "vpcs_interfaces_data" {
  value = data.mgc_network_vpcs_interfaces.vpcs_interfaces_data
}

output "nat_gateway_data" {
  value = data.mgc_network_nat_gateway.example
}
