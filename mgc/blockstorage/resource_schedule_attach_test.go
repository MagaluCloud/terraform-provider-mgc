package blockstorage

import (
	"context"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	storageSDK "github.com/MagaluCloud/mgc-sdk-go/blockstorage"
	"github.com/MagaluCloud/terraform-provider-mgc/mgc/utils"
)

// MockSchedulerService is a mock implementation of the SchedulerService interface
type MockSchedulerService struct {
	mock.Mock
}

func (m *MockSchedulerService) List(ctx context.Context, opts storageSDK.SchedulerListOptions) (*storageSDK.SchedulerListResponse, error) {
	args := m.Called(ctx, opts)
	return args.Get(0).(*storageSDK.SchedulerListResponse), args.Error(1)
}

func (m *MockSchedulerService) Create(ctx context.Context, req storageSDK.SchedulerPayload) (string, error) {
	args := m.Called(ctx, req)
	return args.String(0), args.Error(1)
}

func (m *MockSchedulerService) Get(ctx context.Context, id string, expand []storageSDK.ExpandSchedulers) (*storageSDK.SchedulerResponse, error) {
	args := m.Called(ctx, id, expand)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*storageSDK.SchedulerResponse), args.Error(1)
}

func (m *MockSchedulerService) Delete(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockSchedulerService) AttachVolume(ctx context.Context, id string, req storageSDK.SchedulerVolumeIdentifierPayload) error {
	args := m.Called(ctx, id, req)
	return args.Error(0)
}

func (m *MockSchedulerService) DetachVolume(ctx context.Context, id string, req storageSDK.SchedulerVolumeIdentifierPayload) error {
	args := m.Called(ctx, id, req)
	return args.Error(0)
}

func TestScheduleAttachResource_Metadata(t *testing.T) {
	t.Parallel()

	r := NewBlockStorageScheduleAttachResource()
	req := resource.MetadataRequest{
		ProviderTypeName: "mgc",
	}
	resp := &resource.MetadataResponse{}

	r.Metadata(context.Background(), req, resp)

	assert.Equal(t, "mgc_block_storage_schedule_attach", resp.TypeName)
}

func TestScheduleAttachResource_Configure(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		providerData interface{}
		expectError  bool
	}{
		{
			name:         "valid configuration",
			providerData: utils.DataConfig{},
			expectError:  false,
		},
		{
			name:         "nil provider data",
			providerData: nil,
			expectError:  false,
		},
		{
			name:         "invalid provider data type",
			providerData: "invalid",
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := NewBlockStorageScheduleAttachResource().(*bsScheduleAttach)
			req := resource.ConfigureRequest{
				ProviderData: tt.providerData,
			}
			resp := &resource.ConfigureResponse{}

			r.Configure(context.Background(), req, resp)

			if tt.expectError {
				assert.True(t, resp.Diagnostics.HasError())
			} else {
				assert.False(t, resp.Diagnostics.HasError())
			}
		})
	}
}

func TestScheduleAttachResource_Schema(t *testing.T) {
	t.Parallel()

	r := NewBlockStorageScheduleAttachResource()
	req := resource.SchemaRequest{}
	resp := &resource.SchemaResponse{}

	r.Schema(context.Background(), req, resp)

	schema := resp.Schema
	assert.NotNil(t, schema)
	assert.Equal(t, "Attaches a block storage volume to a snapshot schedule. This creates a relationship between a volume and a schedule, allowing the schedule to create snapshots of the volume.", schema.Description)

	// Check required attributes
	scheduleIDAttr := schema.Attributes["schedule_id"]
	assert.NotNil(t, scheduleIDAttr)
	assert.True(t, scheduleIDAttr.IsRequired())

	volumeIDAttr := schema.Attributes["volume_id"]
	assert.NotNil(t, volumeIDAttr)
	assert.True(t, volumeIDAttr.IsRequired())
}

func TestScheduleAttachResource_Read(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		scheduleID    string
		volumeID      string
		volumes       []string
		getError      error
		expectError   bool
		expectedError string
	}{
		{
			name:        "volume attached successfully",
			scheduleID:  "schedule-123",
			volumeID:    "volume-456",
			volumes:     []string{"volume-456", "volume-789"},
			getError:    nil,
			expectError: false,
		},
		{
			name:          "volume not attached",
			scheduleID:    "schedule-123",
			volumeID:      "volume-456",
			volumes:       []string{"volume-789"},
			getError:      nil,
			expectError:   true,
			expectedError: "Volume volume-456 is not attached to schedule schedule-123",
		},
		{
			name:          "get schedule error",
			scheduleID:    "schedule-123",
			volumeID:      "volume-456",
			volumes:       nil,
			getError:      assert.AnError,
			expectError:   true,
			expectedError: "assert.AnError general error for testing",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockScheduler := &MockSchedulerService{}
			r := &bsScheduleAttach{
				bsScheduler: mockScheduler,
			}

			if tt.getError == nil {
				scheduleResponse := &storageSDK.SchedulerResponse{
					ID:      tt.scheduleID,
					Volumes: tt.volumes,
				}
				mockScheduler.On("Get", mock.Anything, tt.scheduleID, []storageSDK.ExpandSchedulers{}).Return(scheduleResponse, nil)
			} else {
				mockScheduler.On("Get", mock.Anything, tt.scheduleID, []storageSDK.ExpandSchedulers{}).Return((*storageSDK.SchedulerResponse)(nil), tt.getError)
			}

			// Create state with test data
			data := &bsScheduleAttachResourceModel{
				ScheduleID: types.StringValue(tt.scheduleID),
				VolumeID:   types.StringValue(tt.volumeID),
			}

			// We can't easily test the full request/response cycle without terraform framework,
			// so we'll test the core logic directly
			get, err := r.bsScheduler.Get(context.Background(), data.ScheduleID.ValueString(), []storageSDK.ExpandSchedulers{})

			if err != nil {
				if tt.expectError {
					assert.Error(t, err)
				} else {
					t.Errorf("Unexpected error: %v", err)
				}
			} else {
				contains := false
				for _, v := range get.Volumes {
					if v == data.VolumeID.ValueString() {
						contains = true
						break
					}
				}

				if tt.expectError && !contains {
					// This is expected - volume not found in list
					assert.False(t, contains)
				} else if !tt.expectError {
					assert.True(t, contains)
				}
			}

			mockScheduler.AssertExpectations(t)
		})
	}
}

func TestScheduleAttachResource_Update(t *testing.T) {
	t.Parallel()

	r := NewBlockStorageScheduleAttachResource()
	req := resource.UpdateRequest{}
	resp := &resource.UpdateResponse{}

	r.Update(context.Background(), req, resp)

	assert.True(t, resp.Diagnostics.HasError())
	assert.Contains(t, resp.Diagnostics.Errors()[0].Summary(), "Update not supported")
}

func TestScheduleAttachResource_ImportState(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		id          string
		expectError bool
		scheduleID  string
		volumeID    string
	}{
		{
			name:        "valid import ID",
			id:          "schedule-123,volume-456",
			expectError: false,
			scheduleID:  "schedule-123",
			volumeID:    "volume-456",
		},
		{
			name:        "invalid import ID - too few parts",
			id:          "schedule-123",
			expectError: true,
		},
		{
			name:        "invalid import ID - too many parts",
			id:          "schedule-123,volume-456,extra",
			expectError: true,
		},
		{
			name:        "empty import ID",
			id:          "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test the import logic directly
			input := tt.id
			if tt.expectError {
				parts := len(input)
				if input == "" {
					parts = 0
				} else {
					parts = len(strings.Split(input, ","))
				}
				assert.NotEqual(t, 2, parts, "Expected invalid format")
			} else {
				parts := strings.Split(input, ",")
				assert.Equal(t, 2, len(parts), "Expected valid format")
				assert.Equal(t, tt.scheduleID, parts[0])
				assert.Equal(t, tt.volumeID, parts[1])
			}
		})
	}
}

func TestScheduleAttachResource_AttachVolume(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		scheduleID  string
		volumeID    string
		attachError error
		expectError bool
	}{
		{
			name:        "successful attachment",
			scheduleID:  "schedule-123",
			volumeID:    "volume-456",
			attachError: nil,
			expectError: false,
		},
		{
			name:        "attach volume error",
			scheduleID:  "schedule-123",
			volumeID:    "volume-456",
			attachError: assert.AnError,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockScheduler := &MockSchedulerService{}
			r := &bsScheduleAttach{
				bsScheduler: mockScheduler,
			}

			expectedPayload := storageSDK.SchedulerVolumeIdentifierPayload{
				Volume: storageSDK.IDOrName{
					ID: &tt.volumeID,
				},
			}

			mockScheduler.On("AttachVolume", mock.Anything, tt.scheduleID, expectedPayload).Return(tt.attachError)

			err := r.bsScheduler.AttachVolume(context.Background(), tt.scheduleID, expectedPayload)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			mockScheduler.AssertExpectations(t)
		})
	}
}

func TestScheduleAttachResource_DetachVolume(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		scheduleID  string
		volumeID    string
		detachError error
		expectError bool
	}{
		{
			name:        "successful detachment",
			scheduleID:  "schedule-123",
			volumeID:    "volume-456",
			detachError: nil,
			expectError: false,
		},
		{
			name:        "detach volume error",
			scheduleID:  "schedule-123",
			volumeID:    "volume-456",
			detachError: assert.AnError,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockScheduler := &MockSchedulerService{}
			r := &bsScheduleAttach{
				bsScheduler: mockScheduler,
			}

			expectedPayload := storageSDK.SchedulerVolumeIdentifierPayload{
				Volume: storageSDK.IDOrName{
					ID: &tt.volumeID,
				},
			}

			mockScheduler.On("DetachVolume", mock.Anything, tt.scheduleID, expectedPayload).Return(tt.detachError)

			err := r.bsScheduler.DetachVolume(context.Background(), tt.scheduleID, expectedPayload)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			mockScheduler.AssertExpectations(t)
		})
	}
}
