data "mgc_network_vpcs" "my_vpcs" {
}

output "my_vpcs" {
  description = "Details of all VPCs"
  value       = data.mgc_network_vpcs.my_vpcs.items
}

# Create a VM with default networking
resource "mgc_virtual_machine_instances" "vm_client" {
  name         = "${var.db_prefix}-vm-client"
  machine_type = "BV2-4-20"
  image        = "cloud-ubuntu-24.04 LTS"
  ssh_key_name = "your-ssh-key-name"
  db_prefix_data    = base64encode(<<-EOF
    #!/bin/bash
    apt-get update
    apt-get install -y mysql-client
    apt-get install curl
  EOF
  )
}

# Create a public IP
resource "mgc_network_public_ips" "vm_public_ip" {
  description = "Public IP for my VM"
  vpc_id      = data.mgc_network_vpcs.my_vpcs.items[0].id
}

# Find the default interface and attach the public IP
locals {
  primary_interface_id = [
    for interface in mgc_virtual_machine_instances.vm_client.network_interfaces :
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