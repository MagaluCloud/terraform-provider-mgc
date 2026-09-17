# Import a block storage schedule attach resource
# The import ID format is: [project/{project_id}/]schedule_id,volume_id

# Example: Import an existing schedule-volume attachment
terraform import mgc_block_storage_schedule_attach.example_attach "schedule-123,volume-456"

# Attachment in a project
terraform import mgc_block_storage_schedule_attach.example_attach "project/550e8400-e29b-41d4-a716-446655440000/schedule-123,volume-456"
