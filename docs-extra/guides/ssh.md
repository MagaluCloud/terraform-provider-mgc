---
page_title: "SSH Keys Management"
subcategory: "Guides"
description: |-
  How to create, manage, and use SSH keys with virtual machines in Magalu Cloud.
---

# SSH Keys in Magalu Cloud: Working with Virtual Machines

SSH keys are a critical component for securely accessing virtual machines in Magalu Cloud. This guide explains how to create, manage, and use SSH keys with your VMs.

## Understanding SSH Keys in Magalu Cloud

SSH keys in Magalu Cloud are **global resources**, meaning they are not tied to a specific region. Once you create an SSH key, it's available across all regions in your account. This global nature makes them convenient for managing access to VMs deployed in different geographical locations.

## Creating SSH Keys

### 1: Generate SSH Keys on Your Local Machine

Before uploading to Magalu Cloud, you can generate a new SSH key pair:

```bash
# Generate an SSH key pair (RSA)
ssh-keygen -t rsa -b 4096 -C "your_email@example.com"

# Or generate an Ed25519 key (more secure, recommended)
ssh-keygen -t ed25519 -C "your_email@example.com"
```

### 2: Create SSH Keys resource

To add an SSH key to Magalu Cloud using Terraform:

```terraform
resource "mgc_ssh_keys" "my_key" {
  name = "my-production-key"
  key  = "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIP+E3U/DpNagT79ueF+xQn9dNFUKheopjx/kIBC1qQM3 your_email@example.com"
}
```

Important parameters:

- `name`: A unique identifier for your key (lowercase letters, hyphens, underscores, and numbers only)
- `key`: The public key string from your key pair

## Listing Available SSH Keys

To see all SSH keys available in your account:

```terraform
data "mgc_ssh_keys" "available_keys" {
}

output "my_ssh_keys" {
  value = data.mgc_ssh_keys.available_keys.ssh_keys
}
```

This will output a list of all your SSH keys with their IDs, names, and types.

## Using SSH Keys with Virtual Machines

When creating a VM, specify the SSH key name to enable secure access:

```terraform
resource "mgc_virtual_machine_instances" "web_server" {
  name         = "web-server"
  machine_type = "BV1-1-40"
  image        = "cloud-ubuntu-24.04 LTS"
  ssh_key_name = mgc_ssh_keys.my_key.name
}
```

You can also reference an existing SSH key by name:

```terraform
resource "mgc_virtual_machine_instances" "app_server" {
  name         = "app-server"
  machine_type = "BV1-1-40"
  image        = "cloud-ubuntu-24.04 LTS"
  ssh_key_name = "my-existing-key"
}
```

## Multi-Region Deployment with the Same SSH Key

Since SSH keys are global, you can use the same key across regions:

```terraform
# Southeast region provider
provider "mgc" {
  alias  = "southeast"
  region = "br-se1"
}

# Northeast region provider
provider "mgc" {
  alias  = "northeast"
  region = "br-ne1"
}

# Create SSH key (only need to do this once)
resource "mgc_ssh_keys" "global_key" {
  provider = mgc.southeast  # Provider doesn't matter since keys are global
  name     = "global-access-key"
  key      = "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIPmEeVLwGP7A87jxHH+LGShN7h4L3T7TG2FX+S3mNCB7 your_email@example.com"
}

# Create VM in Southeast region using the key
resource "mgc_virtual_machine_instances" "southeast_vm" {
  provider     = mgc.southeast
  name         = "southeast-server"
  machine_type = "BV1-1-40"
  image        = "cloud-ubuntu-24.04 LTS"
  ssh_key_name = mgc_ssh_keys.global_key.name
}

# Create VM in Northeast region using the same key
resource "mgc_virtual_machine_instances" "northeast_vm" {
  provider     = mgc.northeast
  name         = "northeast-server"
  machine_type = "BV1-1-40"
  image        = "cloud-ubuntu-24.04 LTS"
  ssh_key_name = mgc_ssh_keys.global_key.name
}
```

## Managing SSH Access Within VMs

Once your VM is created with an SSH key, the key will be automatically added to the authorized keys for the default user:

- For Ubuntu images: The user is `ubuntu`
- For other Linux distributions: Check the distribution documentation

To connect to your VM:

```bash
# Connect using the private key corresponding to the uploaded public key
ssh -i /path/to/private_key ubuntu@vm_public_ip
```

## SSH Keys and Windows VMs

For Windows VMs, SSH keys are not utilized for direct access. Instead:

- Windows VMs use password authentication
- The password is generated automatically and can be retrieved from the Magalu Cloud CLI
- Consider setting up key-based authentication manually if needed

## Important Notes

1. **SSH Key Naming Restrictions**: SSH key names can only contain lowercase letters, hyphens, underscores, and numbers.

2. **Global Resource**: Remember that SSH keys exist at the account level, not the region level, making them accessible across all regions.

3. **Linux Distributions Only**: SSH key injection at launch only works for Linux VMs; Windows VMs require different authentication methods.

By effectively managing your SSH keys in Magalu Cloud, you can ensure secure access to your virtual machines while maintaining operational flexibility across all regions.
