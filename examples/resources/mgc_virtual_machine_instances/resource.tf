resource "mgc_virtual_machine_instances" "basic_instance" {
  name         = "basic-instance-name"
  machine_type = "BV1-1-40"
  image        = "cloud-ubuntu-24.04 LTS"
  ssh_key_name = "your-ssh-key-name"
}

resource "mgc_virtual_machine_instances" "instance_with_az" {
  name              = "instance-with-az"
  availability_zone = "br-ne1-a"
  machine_type      = "BV4-8-100"
  image             = "cloud-ubuntu-24.04 LTS"
  ssh_key_name      = "your-ssh-key-name"
}

resource "mgc_virtual_machine_instances" "instance_with_userdata" {
  name         = "instance-with-userdata"
  machine_type = "BV4-8-100"
  image        = "cloud-ubuntu-24.04 LTS"
  ssh_key_name = "your-ssh-key-name"
  user_data    = base64encode("#!/bin/bash\necho 'Hello, World!'")
}

resource "mgc_virtual_machine_instances" "instance_with_windows" {
  name         = "instance-with-windows"
  machine_type = "BV4-8-100"
  image        = "windows-server-2022"
}

resource "mgc_virtual_machine_instances" "instance_with_custom_interface" {
  name                 = "instance-with-custom-interface"
  machine_type         = "BV2-4-10"
  image                = "cloud-ubuntu-24.04 LTS"
  ssh_key_name         = "your-ssh-key-name"
  network_interface_id = mgc_network_vpcs_interfaces.custom_interface.id
}

resource "mgc_virtual_machine_instances" "instance_with_public_ipv4" {
  name                 = "instance-with-public-ipv4"
  machine_type         = "BV2-4-10"
  image                = "cloud-ubuntu-24.04 LTS"
  ssh_key_name         = "your-ssh-key-name"
  allocate_public_ipv4 = true
}

resource "mgc_virtual_machine_instances" "instance_with_security_groups" {
  name                     = "instance-with-security-groups"
  machine_type             = "BV2-4-10"
  image                    = "cloud-ubuntu-24.04 LTS"
  ssh_key_name             = "your-ssh-key-name"
  creation_security_groups = [mgc_network_security_groups.security_group.id]
}

resource "mgc_virtual_machine_instances" "instance_with_security_groups_and_public_ipv4" {
  name                     = "instance_with_security_groups_and_public_ipv4"
  machine_type             = "BV2-4-10"
  image                    = "cloud-ubuntu-24.04 LTS"
  ssh_key_name             = "your-ssh-key-name"
  allocate_public_ipv4     = true
  creation_security_groups = [mgc_network_security_groups.security_group.id]
}
