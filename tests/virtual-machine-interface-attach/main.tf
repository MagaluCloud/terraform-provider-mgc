resource "mgc_network_vpcs" "main_vpc" {
  name = "main-vpc-test-tf-attach-vm"

}

resource "mgc_network_subnetpools" "main_subnetpool" {
  name        = "main-subnetpool"
  description = "Main Subnet Pool"
  cidr        = "172.29.0.0/16"

}

resource "mgc_network_vpcs_subnets" "primary_subnet" {
  cidr_block      = "172.29.1.0/24"
  description     = "Primary Network Subnet"
  dns_nameservers = ["8.8.8.8", "8.8.4.4"]
  ip_version      = "IPv4"
  name            = "primary-subnet"
  subnetpool_id   = mgc_network_subnetpools.main_subnetpool.id
  vpc_id          = mgc_network_vpcs.main_vpc.id

  depends_on = [mgc_network_subnetpools.main_subnetpool]
}

resource "mgc_network_vpcs_interfaces" "primary_interface" {
  name   = "interface-attach-vm"
  vpc_id = mgc_network_vpcs.main_vpc.id

  depends_on = [mgc_network_vpcs_subnets.primary_subnet]
}

resource "mgc_network_public_ips" "public_ip" {
  vpc_id = mgc_network_vpcs.main_vpc.id
  description = "Public IP for attach VM"
  depends_on = [mgc_network_vpcs.main_vpc]
}

resource "mgc_network_public_ips_attach" "public_ip_attach" {
  interface_id = mgc_network_vpcs_interfaces.primary_interface.id
  public_ip_id = mgc_network_public_ips.public_ip.id
  depends_on = [mgc_network_public_ips.public_ip]
}

resource "mgc_virtual_machine_instances" "tc1_basic_instance" {
  name         = "tc1-basic-instance-attach-vm"
  machine_type = "BV1-1-40"
  image        = "cloud-ubuntu-24.04 LTS"
  ssh_key_name = "publio"
}

resource "mgc_virtual_machine_instances" "tc1_basic_instance_with_vpcid" {
  name         = "tc1-basic-instance-with-vpcid"
  machine_type = "BV1-1-40"
  vpc_id       = mgc_network_vpcs.main_vpc.id
  image        = "cloud-ubuntu-24.04 LTS"
  ssh_key_name = "publio"
}

# 8141911d-62e8-4337-b735-3dce3c9fd3c7
resource "mgc_virtual_machine_interface_attach" "attach_vm" {
  instance_id  = mgc_virtual_machine_instances.tc1_basic_instance.id
  interface_id = mgc_network_vpcs_interfaces.primary_interface.id
}
