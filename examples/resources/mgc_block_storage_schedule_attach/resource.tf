# Create a block storage volume
resource "mgc_block_storage_volumes" "example_volume" {
  name              = "example-volume"
  availability_zone = "br-ne1-a"
  size              = 50
  type              = "cloud_nvme1k"
}

# Create a snapshot schedule
resource "mgc_block_storage_schedule" "daily_backup" {
  name                              = "daily-backup-schedule"
  description                       = "Daily backup schedule for production volumes"
  snapshot_type                     = "instant"
  policy_retention_in_days          = 7
  policy_frequency_daily_start_time = "02:00:00"
}

# Attach the volume to the schedule
resource "mgc_block_storage_schedule_attach" "example_attach" {
  schedule_id = mgc_block_storage_schedule.daily_backup.id
  volume_id   = mgc_block_storage_volumes.example_volume.id
}

# Multiple volumes can be attached to the same schedule
resource "mgc_block_storage_volumes" "another_volume" {
  name              = "another-volume"
  availability_zone = "br-ne1-a"
  size              = 100
  type              = "cloud_nvme1k"
}

resource "mgc_block_storage_schedule_attach" "another_attach" {
  schedule_id = mgc_block_storage_schedule.daily_backup.id
  volume_id   = mgc_block_storage_volumes.another_volume.id
}

# A volume can also be attached to multiple schedules
resource "mgc_block_storage_schedule" "weekly_backup" {
  name                              = "weekly-backup-schedule"
  description                       = "Weekly backup schedule for long-term retention"
  snapshot_type                     = "object"
  policy_retention_in_days          = 30
  policy_frequency_daily_start_time = "03:00:00"
}

resource "mgc_block_storage_schedule_attach" "multi_schedule_attach" {
  schedule_id = mgc_block_storage_schedule.weekly_backup.id
  volume_id   = mgc_block_storage_volumes.example_volume.id
}
