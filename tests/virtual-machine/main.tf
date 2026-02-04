# Variables for testing updates
variable "instance_name" {
  description = "Name for the test instance"
  type        = string
  default     = "tc1-basic-instance"
}

variable "machine_type" {
  description = "Machine type for the test instance"
  type        = string
  default     = "BV1-2-40"
}

resource "mgc_ssh_keys" "ssh_key" {
  name = "testkey"
  key  = "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIAKWxY1opxbh0LNTNeAHORTZEHdPRFvrfcXyhZQlsR7t"
}

# # # Test Case 1: Basic VM Instance
# resource "mgc_virtual_machine_instances" "tc1_basic_instance" {
#   name              = var.instance_name
#   availability_zone = "br-se1-a"
#   machine_type      = var.machine_type
#   image             = "cloud-ubuntu-24.04 LTS"
#   ssh_key_name      = mgc_ssh_keys.ssh_key.name
#   user_data         = base64encode("#!/bin/bash\necho 'Test Case 1: Basic Instance'")

#   lifecycle {
#     create_before_destroy = true
#   }
# }

# resource "mgc_virtual_machine_instances" "tc1_basic_instance_from_restore" {
#   name              = "vm-from-snapshot"
#   availability_zone = "br-se1-a"
#   machine_type      = var.machine_type
#   # image             = "cloud-ubuntu-24.04 LTS"
#   ssh_key_name = mgc_ssh_keys.ssh_key.name
#   user_data    = base64encode("#!/bin/bash\necho 'Test Case 1: Basic Instance'")
#   snapshot_id  = "70b82c67-3e0a-4652-8c09-aa7710a6cdc6"
# }

# # # Test Case 2: VM Instance with Availability Zone
# resource "mgc_virtual_machine_instances" "tc2_instance_with_az" {
#   name              = "tc2-instance-with-az"
#   availability_zone = "br-se1-a"
#   machine_type      = "BV4-8-100"
#   image             = "cloud-ubuntu-24.04 LTS"
#   ssh_key_name      = mgc_ssh_keys.ssh_key.name
#   user_data         = base64encode("#!/bin/bash\necho 'Test Case 2: AZ Instance'")

#   depends_on = [mgc_virtual_machine_instances.tc1_basic_instance]
# }

# Test Case 3: VM Instance with custom interface
resource "mgc_network_vpcs_interfaces" "custom_interface" {
  name   = "custom-pip-interface"
  vpc_id = "144e5176-5a75-4afc-ae75-38160f9fd21d"
}

resource "mgc_virtual_machine_instances" "instance_with_custom_interface" {
  name                 = "tc4-instance-with-custom-interface-hc"
  machine_type         = var.machine_type
  image                = "cloud-ubuntu-24.04 LTS"
  ssh_key_name         = mgc_ssh_keys.ssh_key.name
  network_interface_id = mgc_network_vpcs_interfaces.custom_interface.id
}

resource "mgc_virtual_machine_instances" "instance_with_pip" {
  name                 = "tc4-instance-with-public-ip"
  machine_type         = var.machine_type
  image                = "cloud-ubuntu-24.04 LTS"
  ssh_key_name         = mgc_ssh_keys.ssh_key.name
  allocate_public_ipv4 = true
}

# resource "mgc_network_security_groups" "security_group" {
#   name = "auxiliary-security-group-vm-a"
# }

# resource "mgc_virtual_machine_instances" "instance_with_sg" {
#   name                     = "tc4-instance-with-sg"
#   machine_type             = var.machine_type
#   image                    = "cloud-ubuntu-24.04 LTS"
#   ssh_key_name             = mgc_ssh_keys.ssh_key.name
#   creation_security_groups = [mgc_network_security_groups.security_group.id]
# }

# # # Data Sources for Validation
# data "mgc_virtual_machine_instance" "tc1_validation" {
#   id = mgc_virtual_machine_instances.tc1_basic_instance.id
# }

# data "mgc_virtual_machine_instance" "tc2_validation" {
#   id = mgc_virtual_machine_instances.tc2_instance_with_az.id
# }

# # data "mgc_virtual_machine_instance" "tc4_validation" {
# #   id = mgc_virtual_machine_instances.instance_with_custom_interface.id
# # }

# # # List Resources for Testing
# data "mgc_virtual_machine_instances" "all_instances" {}
# data "mgc_virtual_machine_images" "available_images" {}
# data "mgc_virtual_machine_types" "available_types" {}
