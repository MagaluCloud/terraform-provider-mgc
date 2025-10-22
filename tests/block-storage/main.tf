resource "mgc_block_storage_volumes" "example_volume" {
  name              = "example-volume"
  availability_zone = "br-ne1-a"
  size              = 200
  encrypted         = true
  type              = "cloud_nvme1k"
}

# resource "mgc_virtual_machine_instances" "basic_instance" {
#   name         = "basic-instance-test-smoke"
#   machine_type = "cloud-bs1.xsmall"
#   image        = "cloud-ubuntu-24.04 LTS"
#   ssh_key_name = "publio"
# }

# resource "mgc_block_storage_volume_attachment" "example_attachment" {
#   block_storage_id   = mgc_block_storage_volumes.example_volume.id
#   virtual_machine_id = mgc_virtual_machine_instances.basic_instance.id
# }

# resource "mgc_block_storage_snapshots" "snapshot_example" {
#   name        = "snapshot-example"
#   description = "Example snapshot description"
#   type        = "instant"
#   volume_id   = mgc_block_storage_volumes.example_volume.id
#   depends_on  = [mgc_block_storage_volume_attachment.example_attachment]
# }

# data "mgc_block_storage_volume" "volume_data" {
#   id = mgc_block_storage_volumes.example_volume.id

#   depends_on = [mgc_block_storage_volumes.example_volume]
# }

# data "mgc_block_storage_snapshot" "snapshot_data" {
#   id = mgc_block_storage_snapshots.snapshot_example.id

#   depends_on = [mgc_block_storage_snapshots.snapshot_example]
# }

data "mgc_block_storage_volumes" "volumes_data" {
}

data "mgc_block_storage_volume_types" "volume_types_data" {
}

data "mgc_block_storage_schedules" "schedules_data" {
}

output "name" {
  value = data.mgc_block_storage_volumes.volumes_data
}

output "volume_types" {
  value = data.mgc_block_storage_volume_types.volume_types_data
}

output "schedules" {
  value = data.mgc_block_storage_schedules.schedules_data
}

# output "volume_details" {
#   value = data.mgc_block_storage_volume.volume_data
# }

# output "snapshot_details" {
#   value = data.mgc_block_storage_snapshot.snapshot_data
# }
