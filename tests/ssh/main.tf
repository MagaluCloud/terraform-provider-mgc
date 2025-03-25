resource "mgc_ssh_keys" "my_key" {
  name = "my-new-key"
  key  = "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAAAgQDaApQPClXbV1zCp4isUKU5b5+xAuzOX7JS0SsNJ55vlrCQVYnBhgybVm8h1dPwa0NBmnSg82S07Qbw1PWxqq+nGz88xM7KIUsfPpVvJ2TL0QaReYk9b+lAs6zt0CUk4gd1mMAQeK8u1E4OzFyQ/3D8IsrcaX746mkOoS6MLocbLQ=="
}

data "mgc_ssh_keys" "my_keys" {
}

output "keys" {
  value = data.mgc_ssh_keys.my_keys
}
