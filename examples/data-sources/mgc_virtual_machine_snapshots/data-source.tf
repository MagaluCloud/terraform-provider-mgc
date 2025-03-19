data "mgc_virtual_machine_snapshots" "snaps" {
}

output "vm_snapshots" {
  value = data.mgc_virtual_machine_snapshots.snaps
}
