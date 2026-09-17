# Example: Import an existing schedule
terraform import mgc_block_storage_schedule.daily_backup "schedule-123"

# Schedule in a project
terraform import mgc_block_storage_schedule.daily_backup "project/550e8400-e29b-41d4-a716-446655440000/schedule-123"
