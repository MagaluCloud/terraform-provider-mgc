---
page_title: "Creating VMs with Volumes"
subcategory: "Guides"
description: "A comprehensive guide to managing block storage in Magalu Cloud, including creating volumes, attaching them to VMs, and working with snapshots."
---

# Managing Block Storage in Magalu Cloud: Volumes, Attachments, and Snapshots

This guide shows you how to work with block storage in Magalu Cloud, including creating volumes, attaching them to virtual machines, and managing snapshots.

## Creating and Attaching Storage Volumes

Block storage volumes are persistent storage devices you can attach to virtual machines. They function like external hard drives for your cloud VMs.

### 1. Creating a Volume

First, let's create a block storage volume:

```terraform
resource "mgc_block_storage_volumes" "data_volume" {
  name              = "data-volume"
  availability_zone = "br-ne1-a"  # Make sure this matches your VM's availability zone
  size              = 100         # Size in GB
  type              = "cloud_nvme1k"
  encrypted         = true        # Optional: Enable encryption
}
```

Key parameters:

- `name`: A descriptive name for your volume
- `size`: Storage capacity in GB
- `type`: Performance characteristics (e.g., "cloud_nvme1k" for NVMe SSD)
- `availability_zone`: Must match the zone where your VM is deployed
- `encrypted`: Whether to encrypt the volume data (optional)

### 2. Creating a Virtual Machine

Create a VM to attach the volume to:

```terraform
resource "mgc_virtual_machine_instances" "app_server" {
  name              = "app-server"
  machine_type      = "cloud-bs1.small"
  image             = "cloud-ubuntu-24.04 LTS"
  ssh_key_name      = "your-ssh-key"
  availability_zone = "br-ne1-a"  # Must match the volume's zone
}
```

### 3. Attaching the Volume to the VM

Now, attach the volume to your virtual machine:

```terraform
resource "mgc_block_storage_volume_attachment" "app_data_attachment" {
  block_storage_id   = mgc_block_storage_volumes.data_volume.id
  virtual_machine_id = mgc_virtual_machine_instances.app_server.id
}
```

That's it! The volume is now attached to your VM. After the infrastructure is provisioned, you'll need to format and mount the volume from within the VM.

## Creating Volume Snapshots

Snapshots let you create point-in-time backups of your volumes. They're useful for data backup, cloning environments, or disaster recovery.

### 1. Creating a Snapshot of an Attached Volume

```terraform
resource "mgc_block_storage_snapshots" "data_volume_backup" {
  name        = "data-volume-backup"
  description = "Daily backup of application data"
  type        = "instant"
  volume_id   = mgc_block_storage_volumes.data_volume.id

  # Wait for the volume to be attached before creating the snapshot
  depends_on  = [mgc_block_storage_volume_attachment.app_data_attachment]
}
```

Key parameters:

- `name`: A descriptive name for the snapshot
- `description`: Details about the snapshot's purpose
- `type`: The snapshot type ("instant" is commonly used)
- `volume_id`: The ID of the volume to snapshot
- `depends_on`: Ensures the volume is properly attached before taking the snapshot

### 2. Creating a Volume from a Snapshot

You can restore data by creating a new volume from a snapshot:

```terraform
resource "mgc_block_storage_volumes" "restored_volume" {
  name              = "restored-volume"
  availability_zone = "br-ne1-a"
  size              = 100
  type              = "cloud_nvme1k"
  encrypted         = true //Encryption request must match with snapshot's encryption
  snapshot_id       = mgc_block_storage_snapshots.data_volume_backup.id
}
```

The `snapshot_id` parameter tells Magalu Cloud to populate the new volume with data from the specified snapshot.

## Complete Example: Volume Management Workflow

Here's a complete example showing a typical block storage workflow:

```terraform
# Create a volume
resource "mgc_block_storage_volumes" "data_volume" {
  name              = "data-volume"
  availability_zone = "br-ne1-a"
  size              = 200
  encrypted         = true
  type              = "cloud_nvme1k"
}

# Create a VM
resource "mgc_virtual_machine_instances" "web_server" {
  name         = "web-server"
  machine_type = "cloud-bs1.small"
  image        = "cloud-ubuntu-24.04 LTS"
  ssh_key_name = "your-ssh-key"

  # Keep VM and volume in the same availability zone
  availability_zone = "br-ne1-a"
}

# Attach volume to VM
resource "mgc_block_storage_volume_attachment" "web_data_attachment" {
  block_storage_id   = mgc_block_storage_volumes.data_volume.id
  virtual_machine_id = mgc_virtual_machine_instances.web_server.id
}

# Create a snapshot of the volume
resource "mgc_block_storage_snapshots" "daily_backup" {
  name        = "web-data-backup"
  description = "Daily backup of web server data"
  type        = "instant"
  volume_id   = mgc_block_storage_volumes.data_volume.id

  # Make sure the volume is attached before creating the snapshot
  depends_on  = [mgc_block_storage_volume_attachment.web_data_attachment]
}

# Create a new volume from the snapshot (for recovery purposes)
resource "mgc_block_storage_volumes" "recovery_volume" {
  name              = "recovery-volume"
  availability_zone = "br-ne1-a"
  size              = 200
  type              = "cloud_nvme1k"
  encrypted         = true //Encryption request must match with snapshot's encryption
  snapshot_id       = mgc_block_storage_snapshots.daily_backup.id
}

# Output the volume and snapshot details
output "volume_id" {
  value = mgc_block_storage_volumes.data_volume.id
}

output "snapshot_id" {
  value = mgc_block_storage_snapshots.daily_backup.id
}
```
