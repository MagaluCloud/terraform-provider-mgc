# Example: Basic daily schedule with instant snapshots
# Can be imported using: terraform import mgc_block_storage_schedule.daily_backup "schedule-id"
resource "mgc_block_storage_schedule" "daily_backup" {
  name                              = "daily-backup-schedule"
  description                       = "Daily backup schedule for production volumes"
  snapshot_type                     = "instant"
  policy_retention_in_days          = 7
  policy_frequency_daily_start_time = "02:00:00"
}

# Example: Weekly schedule with object snapshots and longer retention
# Can be imported using: terraform import mgc_block_storage_schedule.weekly_backup "schedule-id"
resource "mgc_block_storage_schedule" "weekly_backup" {
  name                              = "weekly-backup-schedule"
  description                       = "Weekly backup schedule for development volumes"
  snapshot_type                     = "object"
  policy_retention_in_days          = 30
  policy_frequency_daily_start_time = "23:00:00"
}

# Example: Minimal schedule configuration (no description)
# Can be imported using: terraform import mgc_block_storage_schedule.minimal_schedule "schedule-id"
resource "mgc_block_storage_schedule" "minimal_schedule" {
  name                              = "minimal-schedule"
  snapshot_type                     = "instant"
  policy_retention_in_days          = 1
  policy_frequency_daily_start_time = "23:30:00"
}
