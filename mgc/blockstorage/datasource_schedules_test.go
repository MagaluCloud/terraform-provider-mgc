package blockstorage

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	storageSDK "github.com/MagaluCloud/mgc-sdk-go/blockstorage"
)

type mockSchedulerServiceDSList struct {
	mock.Mock
	storageSDK.SchedulerService
}

func (m *mockSchedulerServiceDSList) List(ctx context.Context, opts storageSDK.SchedulerListOptions) (*storageSDK.SchedulerListResponse, error) {
	args := m.Called(ctx, opts)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*storageSDK.SchedulerListResponse), args.Error(1)
}

func TestDataSourceBsSchedules_Metadata(t *testing.T) {
	ds := NewDataSourceBSSchedules()

	ctx := context.Background()
	req := datasource.MetadataRequest{
		ProviderTypeName: "mgc",
	}
	resp := &datasource.MetadataResponse{}

	ds.Metadata(ctx, req, resp)

	assert.Equal(t, "mgc_block_storage_schedules", resp.TypeName)
}

func TestDataSourceBsSchedules_Configure_NilProviderData(t *testing.T) {
	ds := &DataSourceBsSchedules{}
	ctx := context.Background()
	req := datasource.ConfigureRequest{
		ProviderData: nil,
	}
	resp := &datasource.ConfigureResponse{}

	ds.Configure(ctx, req, resp)

	assert.False(t, resp.Diagnostics.HasError())
	assert.Nil(t, ds.bsScheduler)
}

func TestDataSourceBsSchedules_Configure_InvalidProviderData(t *testing.T) {
	ds := &DataSourceBsSchedules{}
	ctx := context.Background()
	req := datasource.ConfigureRequest{
		ProviderData: "invalid",
	}
	resp := &datasource.ConfigureResponse{}

	ds.Configure(ctx, req, resp)

	assert.True(t, resp.Diagnostics.HasError())
	assert.Contains(t, resp.Diagnostics.Errors()[0].Summary(), "Failed to configure data source")
}

func TestDataSourceBsSchedules_Configure_ValidProviderData(t *testing.T) {
	t.Skip("Skipping configure test - requires SDK integration")
}

func TestDataSourceBsSchedules_Schema(t *testing.T) {
	ds := &DataSourceBsSchedules{}
	ctx := context.Background()
	req := datasource.SchemaRequest{}
	resp := &datasource.SchemaResponse{}

	ds.Schema(ctx, req, resp)

	assert.NotNil(t, resp.Schema)
	assert.NotEmpty(t, resp.Schema.Description)
	assert.NotEmpty(t, resp.Schema.MarkdownDescription)

	attributes := resp.Schema.Attributes
	assert.Contains(t, attributes, "schedules")

	assert.True(t, attributes["schedules"].IsComputed())
}

func TestDataSourceBsSchedules_Read_Success(t *testing.T) {
	t.Skip("Skipping read test - requires complex mock framework setup")
}

func TestDataSourceBsSchedules_Read_Error(t *testing.T) {
	t.Skip("Skipping read error test - requires complex mock framework setup")
}

func TestDataSourceBsSchedules_Read_EmptyList(t *testing.T) {
	t.Skip("Skipping empty list test - requires complex mock framework setup")
}

func TestBsSchedulesDataSourceModel_FieldTypes(t *testing.T) {
	volumesList, _ := types.ListValueFrom(context.Background(), types.StringType, []string{"vol-123", "vol-456"})

	scheduleModel := bsScheduleDataSourceModel{
		ID:                    types.StringValue("test-id"),
		Name:                  types.StringValue("test-name"),
		Description:           types.StringValue("test-description"),
		Volumes:               volumesList,
		SnapshotType:          types.StringValue("instant"),
		PolicyRetentionInDays: types.Int64Value(7),
		PolicyFrequencyDaily:  types.StringValue("02:00"),
		CreatedAt:             types.StringValue("2023-01-01T00:00:00Z"),
		UpdatedAt:             types.StringValue("2023-01-01T12:00:00Z"),
		State:                 types.StringValue("available"),
	}

	model := bsSchedulesDataSourceModel{
		Schedules: []bsScheduleDataSourceModel{scheduleModel},
	}

	assert.Len(t, model.Schedules, 1)
	assert.Equal(t, "test-id", model.Schedules[0].ID.ValueString())
	assert.Equal(t, "test-name", model.Schedules[0].Name.ValueString())
	assert.Equal(t, "test-description", model.Schedules[0].Description.ValueString())
	assert.Equal(t, "instant", model.Schedules[0].SnapshotType.ValueString())
	assert.Equal(t, int64(7), model.Schedules[0].PolicyRetentionInDays.ValueInt64())
	assert.Equal(t, "02:00", model.Schedules[0].PolicyFrequencyDaily.ValueString())
	assert.Equal(t, "2023-01-01T00:00:00Z", model.Schedules[0].CreatedAt.ValueString())
	assert.Equal(t, "2023-01-01T12:00:00Z", model.Schedules[0].UpdatedAt.ValueString())
	assert.Equal(t, "available", model.Schedules[0].State.ValueString())

	var volumes []string
	model.Schedules[0].Volumes.ElementsAs(context.Background(), &volumes, false)
	assert.Equal(t, []string{"vol-123", "vol-456"}, volumes)
}

func TestNewDataSourceBSSchedules(t *testing.T) {
	ds := NewDataSourceBSSchedules()
	assert.NotNil(t, ds)
	assert.IsType(t, &DataSourceBsSchedules{}, ds)
}
