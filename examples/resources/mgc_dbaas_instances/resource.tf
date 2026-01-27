resource "mgc_dbaas_instances" "test_instance" {
  name                  = "test-instance"
  user                  = "dbadmin"
  password              = "examplepassword"
  engine_name           = "mysql"
  engine_version        = "8.0"
  instance_type         = "DP2-8-40"
  volume_size           = 50
  volume_type           = "CLOUD_NVME15K"
  backup_retention_days = 10
  backup_start_at       = "16:00:00"
  deletion_protected    = true
}
