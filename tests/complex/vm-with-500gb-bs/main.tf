# resource "mgc_network_vpcs" "main_vpc" {
#   name = "main-vpc-test-tf-attach-vm13"
# }


# resource "mgc_network_subnetpools" "main_subnetpool" {
#   name        = "main-subnetpool31"
#   description = "Main Subnet Pool"
#   cidr        = "172.35.0.0/16"
#   depends_on = [ mgc_network_vpcs.main_vpc ]

# }

# resource "mgc_network_vpcs_subnets" "primary_subnet" {
#   cidr_block      = "172.35.1.0/24"
#   description     = "Primary Network Subnet"
#   dns_nameservers = ["8.8.8.8", "8.8.4.4"]
#   ip_version      = "IPv4"
#   name            = "primary-subnet31"
#   subnetpool_id   = mgc_network_subnetpools.main_subnetpool.id
#   vpc_id          = mgc_network_vpcs.main_vpc.id

#   depends_on = [mgc_network_subnetpools.main_subnetpool]
# }

# resource "mgc_network_vpcs_interfaces" "primary_interface" {
#   name   = "interface-attach-vm13"
#   vpc_id = mgc_network_vpcs.main_vpc.id

#   depends_on = [mgc_network_vpcs_subnets.primary_subnet]
# }

# resource "mgc_network_public_ips" "public_ip" {
#   vpc_id = mgc_network_vpcs.main_vpc.id
#   description = "Public IP for attach VM"
#   depends_on = [mgc_network_vpcs_interfaces.primary_interface]
# }


# resource "mgc_block_storage_volumes" "example_volume" {
#   name              = "volume-500gb"
#   availability_zone = "br-ne1-a"
#   size              = 500
#   encrypted         = true
#   type              = "cloud_nvme1k"
#   depends_on = [ mgc_network_vpcs_interfaces.primary_interface ]
# }

# resource "mgc_virtual_machine_instances" "basic_instance" {
#   name         = "vm-10gb"
#   machine_type = "BV1-1-10"
#   image        = "cloud-ubuntu-24.04 LTS"
#   ssh_key_name = "geffatual"
#   depends_on = [ mgc_block_storage_volumes.example_volume ]
# }


# resource "mgc_network_public_ips_attach" "public_ip_attach" {
#   interface_id = mgc_network_vpcs_interfaces.primary_interface.id
#   public_ip_id = mgc_network_public_ips.public_ip.id
#   depends_on = [mgc_virtual_machine_instances.basic_instance]
# }


# resource "mgc_block_storage_volume_attachment" "example_attachment" {
#   block_storage_id   = mgc_block_storage_volumes.example_volume.id
#   virtual_machine_id = mgc_virtual_machine_instances.basic_instance.id
#   depends_on = [mgc_virtual_machine_instances.basic_instance]
# }


resource "mgc_block_storage_volumes" "example_loop_volume" {
  for_each = toset([
    "volume-500gb2",
    "volume-500gb3",
    "volume-500gb4",
    "volume-500gb5",
    "volume-500gb6",
    "volume-500gb7",
    "volume-500gb8",
    "volume-500gb9",
    "volume-500gb10",
    "volume-500gb11",
    "volume-500gb12",
    "volume-500gb13",
    "volume-500gb14",
    "volume-500gb15",
    "volume-500gb16",
    "volume-500gb17",
    "volume-500gb18",
    "volume-500gb19",
    "volume-500gb20"
  ])
  name              = each.value
  availability_zone = "br-ne1-a"
  size              = 500
  encrypted         = true
  type              = "cloud_nvme1k"
  # depends_on = [ mgc_network_vpcs_interfaces.primary_interface ]
  timeouts = {
    create = "24h"
    update = "24h"
  }
}
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

# data "mgc_block_storage_volumes" "volumes_data" {
# }

# output "name" {
#   value = data.mgc_block_storage_volumes.volumes_data
# }

# output "volume_details" {
#   value = data.mgc_block_storage_volume.volume_data
# }

# output "snapshot_details" {
#   value = data.mgc_block_storage_snapshot.snapshot_data
# }
