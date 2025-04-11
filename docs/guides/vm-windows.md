---
page_title: "Creating and Accessing Windows VMs"
subcategory: "Guides"
description: "A guide to creating and connecting to Windows virtual machines in Magalu Cloud, including specific requirements and access methods."
---

# Creating and Accessing Windows VMs in Magalu Cloud

This guide demonstrates how to create Windows virtual machines in Magalu Cloud and how to connect to them using RDP (Remote Desktop Protocol).

## Creating a Windows VM

### Requirements for Windows VMs

Windows virtual machines have specific requirements that differ from Linux VMs:

1. **SSH Key Not Required**: Unlike Linux VMs, Windows VMs don't use SSH keys for authentication
2. **Minimum Resource Requirements**: Windows VMs require at least 2GB RAM and 2 vCPUs
3. **RDP Access**: You'll need to configure security group rules to enable RDP access (port 3389)

### Basic Example

Here's a simple example of creating a Windows VM:

```terraform
resource "mgc_virtual_machine_instances" "windows_server" {
  name         = "windows-server"
  machine_type = "BV4-8-100"  # Ensures minimum 2 vCPUs and 2GB RAM
  image        = "windows-server-2022"
}
```

Key parameters:

- `name`: A descriptive name for your VM
- `machine_type`: Must have at least 2GB RAM and 2 vCPUs (e.g., "BV4-8-100")
- `image`: The Windows image to use (e.g., "windows-server-2022")

### Complete Example with Security Group

```terraform
# Create a security group for Windows RDP access
resource "mgc_network_security_groups" "windows_sg" {
  name        = "windows-rdp-sg"
  description = "Security group for Windows RDP access"
}

# Add RDP rule to security group
resource "mgc_network_security_groups_rules" "rdp_rule" {
  security_group_id = mgc_network_security_groups.windows_sg.id
  ethertype         = "IPv4"
  direction         = "ingress"
  protocol          = "tcp"
  port_range_min    = 3389
  port_range_max    = 3389
  remote_ip_prefix  = "0.0.0.0/0" # Consider restricting to your IP for production
}
resource "mgc_network_security_groups_rules" "rdp_rule_udp" {
  security_group_id = mgc_network_security_groups.windows_sg.id
  ethertype         = "IPv4"
  direction         = "ingress"
  protocol          = "udp"
  port_range_min    = 3389
  port_range_max    = 3389
  remote_ip_prefix  = "0.0.0.0/0" # Consider restricting to your IP for production
}

# Create Windows VM
resource "mgc_virtual_machine_instances" "windows_server" {
  name         = "windows-server"
  machine_type = "BV4-8-100"
  image        = "windows-server-2022"
  vpc_id       = local.vpc_id
}

# Get the primary network interface ID
locals {
  primary_interface_id = [
    for interface in mgc_virtual_machine_instances.windows_server.network_interfaces :
    interface.id if interface.primary
  ][0]
  vpc_id = "9dd2d30e-565d-42ce-a0a3-f2de1c473fed"
}

# Attach security group to the primary network interface
resource "mgc_network_security_groups_attach" "windows_sg_attachment" {
  security_group_id = mgc_network_security_groups.windows_sg.id
  interface_id      = local.primary_interface_id
}

# Create public IP
resource "mgc_network_public_ips" "public_ip" {
  description = "example public ip"
  vpc_id      = local.vpc_id
}

#Public IP Attachment
resource "mgc_network_public_ips_attach" "public_ip_attachment" {
  public_ip_id = mgc_network_public_ips.public_ip.id
  interface_id = local.primary_interface_id
}

# Output the public IP of the instance
output "windows_server_ip" {
  value = mgc_network_public_ips.public_ip.public_ip
}
```

## Accessing Your Windows VM

To access your Windows VM, you'll need to enable RDP access and use an RDP client.

### Enabling RDP Access

Enabling port 3389 is essential for allowing remote access to your Windows Server instance via Remote Desktop Protocol (RDP).

Without this port enabled, the Windows Server would not listen for incoming RDP connections, and remote connection attempts would fail due to the lack of a designated communication channel.

To enable communication through port 3389 for remote access to your Windows Server instance, follow these steps:

1. Access your instance in the cloud service panel
2. Navigate to the instance Details
3. In the "Network" section, find "Security Groups"
4. Click on "Add rule" to create a new rule
5. On the new rule configuration page:
   - Click on "Advanced"
   - Select "TCP" as the protocol
   - Choose "Ingress" as the direction
   - Set the port to "3389"
   - Specify the source IP as "0.0.0.0" (or preferably your specific IP for better security)
   - Click on "Add rule" to apply the configuration

With this rule in effect, you'll be able to access your instance remotely using the Remote Desktop Protocol.

## Retrieving the Windows Password

Unlike Linux VMs where you use your own SSH key, Windows VMs in Magalu Cloud are created with an automatically generated administrator password.

### Using the CLI

You can also retrieve the password using the Magalu Cloud CLI:

```
mgc virtual-machine instances password YOUR_VM_ID
```

This password is required when connecting through RDP using either Remote Desktop Connection or Remmina.

## Troubleshooting Windows VM Connections

If you're having trouble connecting to your Windows VM, check the following:

1. **Security Group Rules**: Ensure port 3389 is open and the security group is properly attached to your VM
2. **VM Status**: Verify that the VM is running and has completed initialization
3. **Network Connectivity**: Confirm that you can reach the VM's IP address (try pinging it)
4. **Credentials**: Double-check that you're using the correct admin password from the portal
