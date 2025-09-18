# Create volumes that will be used with schedules and attachments
resource "mgc_block_storage_volumes" "test_volume_1" {
  name = "schedule-test-volume-1"

  availability_zone = "br-ne1-a"
  size              = 50
  type              = "cloud_nvme1k"
}

resource "mgc_block_storage_volumes" "test_volume_2" {
  name              = "schedule-test-volume-2"
  availability_zone = "br-ne1-a"
  size              = 100
  type              = "cloud_nvme1k"
}

resource "mgc_block_storage_volumes" "test_volume_3" {
  name              = "schedule-test-volume-3"
  availability_zone = "br-ne1-a"
  size              = 75
  type              = "cloud_nvme1k"
}

resource "mgc_block_storage_volumes" "test_volume_4" {
  name              = "schedule-test-volume-4"
  availability_zone = "br-ne1-a"
  size              = 80
  type              = "cloud_nvme1k"
}

resource "mgc_block_storage_volumes" "test_volume_5" {
  name              = "schedule-test-volume-5"
  availability_zone = "br-ne1-a"
  size              = 60
  type              = "cloud_nvme1k"
}

# SCHEDULE TESTING - Test different schedule configurations
# Test 1: Basic daily schedule with instant snapshots
resource "mgc_block_storage_schedule" "daily_instant" {
  name                              = "test-daily-instant"
  description                       = "Test daily schedule with instant snapshots"
  snapshot_type                     = "instant"
  policy_retention_in_days          = 7
  policy_frequency_daily_start_time = "02:00:00"
}

# Test 2: Weekly schedule with object snapshots
resource "mgc_block_storage_schedule" "weekly_object" {
  name                              = "test-weekly-object"
  description                       = "Test weekly schedule with object snapshots"
  snapshot_type                     = "object"
  policy_retention_in_days          = 30
  policy_frequency_daily_start_time = "03:00:00"
}

# Test 3: Minimal schedule without description
resource "mgc_block_storage_schedule" "minimal" {
  name                              = "test-minimal-schedule"
  snapshot_type                     = "instant"
  policy_retention_in_days          = 1
  policy_frequency_daily_start_time = "23:00:00"
}

# Test 4: Long retention schedule
resource "mgc_block_storage_schedule" "long_retention" {
  name                              = "test-long-retention"
  description                       = "Test schedule with maximum retention period"
  snapshot_type                     = "object"
  policy_retention_in_days          = 365
  policy_frequency_daily_start_time = "01:00:00"
}

# Test 5: Schedule with special characters in name
resource "mgc_block_storage_schedule" "special_chars" {
  name                              = "test-special_chars-123"
  description                       = "Test schedule with underscores and numbers in name"
  snapshot_type                     = "instant"
  policy_retention_in_days          = 14
  policy_frequency_daily_start_time = "12:00:00"
}

# SCHEDULE ATTACHMENT TESTING - Test volume-schedule relationships
# Each volume has exactly one schedule, one schedule can have multiple volumes

# Test 6: Multiple volumes attached to daily_instant schedule
resource "mgc_block_storage_schedule_attach" "daily_volume_1" {
  schedule_id = mgc_block_storage_schedule.daily_instant.id
  volume_id   = mgc_block_storage_volumes.test_volume_1.id
}

resource "mgc_block_storage_schedule_attach" "daily_volume_2" {
  schedule_id = mgc_block_storage_schedule.daily_instant.id
  volume_id   = mgc_block_storage_volumes.test_volume_2.id
}

resource "mgc_block_storage_schedule_attach" "daily_volume_3" {
  schedule_id = mgc_block_storage_schedule.daily_instant.id
  volume_id   = mgc_block_storage_volumes.test_volume_3.id
}

# Test 7: Single volume attached to weekly_object schedule
resource "mgc_block_storage_schedule_attach" "weekly_volume_4" {
  schedule_id = mgc_block_storage_schedule.weekly_object.id
  volume_id   = mgc_block_storage_volumes.test_volume_4.id
}

# Test 8: Single volume attached to minimal schedule
resource "mgc_block_storage_schedule_attach" "minimal_volume_5" {
  schedule_id = mgc_block_storage_schedule.minimal.id
  volume_id   = mgc_block_storage_volumes.test_volume_5.id
}

# Test 9: long_retention and special_chars schedules have no volumes attached
# This tests schedules without any volume attachments

# DATA SOURCE TESTING - Verify schedules and their attached volumes
data "mgc_block_storage_schedule" "daily_instant_data" {
  id = mgc_block_storage_schedule.daily_instant.id
  depends_on = [
    mgc_block_storage_schedule_attach.daily_volume_1,
    mgc_block_storage_schedule_attach.daily_volume_2,
    mgc_block_storage_schedule_attach.daily_volume_3
  ]
}

data "mgc_block_storage_schedule" "weekly_object_data" {
  id = mgc_block_storage_schedule.weekly_object.id
  depends_on = [
    mgc_block_storage_schedule_attach.weekly_volume_4
  ]
}

data "mgc_block_storage_schedule" "minimal_data" {
  id = mgc_block_storage_schedule.minimal.id
  depends_on = [
    mgc_block_storage_schedule_attach.minimal_volume_5
  ]
}

data "mgc_block_storage_schedule" "long_retention_data" {
  id = mgc_block_storage_schedule.long_retention.id
}

data "mgc_block_storage_schedule" "special_chars_data" {
  id = mgc_block_storage_schedule.special_chars.id
}

data "mgc_block_storage_schedules" "all_schedules" {
  depends_on = [
    mgc_block_storage_schedule.daily_instant,
    mgc_block_storage_schedule.weekly_object,
    mgc_block_storage_schedule.minimal,
    mgc_block_storage_schedule.long_retention,
    mgc_block_storage_schedule.special_chars
  ]
}
