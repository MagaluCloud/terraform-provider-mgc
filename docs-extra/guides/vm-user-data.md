---
page_title: "Creating VMs with User Data"
subcategory: "Guides"
description: "A guide to provisioning virtual machines in Magalu Cloud with custom user data for initial configuration and bootstrapping."
---

# Provisioning VMs with User Data in Magalu Cloud

This guide demonstrates how to create virtual machines in Magalu Cloud with custom initialization scripts using the `user_data` parameter.

## Understanding User Data

User data allows you to pass scripts or cloud-init directives to your virtual machine during its first boot. This feature helps automate the initial configuration of your VM, including:

- Installing software packages
- Configuring services
- Creating users and setting permissions
- Running custom scripts

## Creating a VM with User Data

### Basic Example

Here's a simple example of provisioning a VM with a bash script as user data:

```terraform
resource "mgc_virtual_machine_instances" "web_server" {
  name              = "web-server"
  machine_type      = "cloud-bs1.small"
  image             = "cloud-ubuntu-24.04 LTS"
  ssh_key_name      = "your-ssh-key"
  user_data         = base64encode(<<-EOF
    #!/bin/bash
    apt-get update
    apt-get install -y nginx
    echo '<h1>Hello from Magalu Cloud!</h1>' > /var/www/html/index.html
    systemctl enable nginx
    systemctl start nginx
  EOF
  )
}
```

Key parameters:

- `name`: A descriptive name for your VM
- `machine_type`: The size and resources allocated to your VM
- `image`: The operating system image to use
- `user_data`: Base64-encoded script or cloud-init configuration

### Important Notes About User Data

1. The `user_data` value must be base64-encoded
2. Most cloud images support cloud-init, but script processing behavior may vary by operating system
3. User data execution happens only during the first boot of the instance
4. The maximum size for user data is 65000 characters
