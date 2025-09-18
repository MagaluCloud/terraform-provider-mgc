# Import a block storage schedule attach resource
# The import ID format is: schedule_id,volume_id

# Example: Import an existing schedule-volume attachment
terraform import mgc_block_storage_schedule_attach.example_attach "schedule-123,volume-456"
