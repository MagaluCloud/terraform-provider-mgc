data "mgc_virtual_machine_instance" "instance" {
  id = mgc_virtual_machine_instances.my_vm.id
}

output "vm_instance" {
  value = data.mgc_virtual_machine_instance.instance
}