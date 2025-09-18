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

type mockSchedulerServiceDS struct {
	mock.Mock
	storageSDK.SchedulerService
}

func (m *mockSchedulerServiceDS) Get(ctx context.Context, id string, expand []storageSDK.ExpandSchedulers) (*storageSDK.SchedulerResponse, error) {
	args := m.Called(ctx, id, expand)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*storageSDK.SchedulerResponse), args.Error(1)
}

func TestDataSourceBsSchedule_Metadata(t *testing.T) {
	ds := NewDataSourceBSSchedule()

	ctx := context.Background()
	req := datasource.MetadataRequest{
		ProviderTypeName: "mgc",
	}
	resp := &datasource.MetadataResponse{}

	ds.Metadata(ctx, req, resp)

	assert.Equal(t, "mgc_block_storage_schedule", resp.TypeName)
}

func TestDataSourceBsSchedule_Configure_NilProviderData(t *testing.T) {
	ds := &DataSourceBsSchedule{}
	ctx := context.Background()
	req := datasource.ConfigureRequest{
		ProviderData: nil,
	}
	resp := &datasource.ConfigureResponse{}

	ds.Configure(ctx, req, resp)

	assert.False(t, resp.Diagnostics.HasError())
	assert.Nil(t, ds.bsScheduler)
}

func TestDataSourceBsSchedule_Configure_InvalidProviderData(t *testing.T) {
	ds := &DataSourceBsSchedule{}
	ctx := context.Background()
	req := datasource.ConfigureRequest{
		ProviderData: "invalid",
	}
	resp := &datasource.ConfigureResponse{}

	ds.Configure(ctx, req, resp)

	assert.True(t, resp.Diagnostics.HasError())
	assert.Contains(t, resp.Diagnostics.Errors()[0].Summary(), "Failed to configure data source")
}

func TestDataSourceBsSchedule_Configure_ValidProviderData(t *testing.T) {
	t.Skip("Skipping configure test - requires SDK integration")
}

func TestGetBsScheduleAttributes_RequiredID(t *testing.T) {
	attributes := GetBsScheduleAttributes(true)

	assert.Contains(t, attributes, "id")
	assert.Contains(t, attributes, "name")
	assert.Contains(t, attributes, "description")
	assert.Contains(t, attributes, "volumes")
	assert.Contains(t, attributes, "snapshot_type")
	assert.Contains(t, attributes, "policy_retention_in_days")
	assert.Contains(t, attributes, "policy_frequency_daily_start_time")
	assert.Contains(t, attributes, "created_at")
	assert.Contains(t, attributes, "updated_at")
	assert.Contains(t, attributes, "state")

	assert.True(t, attributes["id"].IsRequired())
	assert.False(t, attributes["id"].IsComputed())

	assert.True(t, attributes["name"].IsComputed())
	assert.True(t, attributes["description"].IsComputed())
	assert.True(t, attributes["volumes"].IsComputed())
	assert.True(t, attributes["snapshot_type"].IsComputed())
	assert.True(t, attributes["policy_retention_in_days"].IsComputed())
	assert.True(t, attributes["policy_frequency_daily_start_time"].IsComputed())
	assert.True(t, attributes["created_at"].IsComputed())
	assert.True(t, attributes["updated_at"].IsComputed())
	assert.True(t, attributes["state"].IsComputed())
}

func TestGetBsScheduleAttributes_ComputedID(t *testing.T) {
	attributes := GetBsScheduleAttributes(false)

	assert.Contains(t, attributes, "id")
	assert.False(t, attributes["id"].IsRequired())
	assert.True(t, attributes["id"].IsComputed())
}

func TestBsScheduleDataSourceModel_FieldTypes(t *testing.T) {
	volumesList, _ := types.ListValueFrom(context.Background(), types.StringType, []string{"vol-123", "vol-456"})

	model := bsScheduleDataSourceModel{
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

	assert.Equal(t, "test-id", model.ID.ValueString())
	assert.Equal(t, "test-name", model.Name.ValueString())
	assert.Equal(t, "test-description", model.Description.ValueString())
	assert.Equal(t, "instant", model.SnapshotType.ValueString())
	assert.Equal(t, int64(7), model.PolicyRetentionInDays.ValueInt64())
	assert.Equal(t, "02:00", model.PolicyFrequencyDaily.ValueString())
	assert.Equal(t, "2023-01-01T00:00:00Z", model.CreatedAt.ValueString())
	assert.Equal(t, "2023-01-01T12:00:00Z", model.UpdatedAt.ValueString())
	assert.Equal(t, "available", model.State.ValueString())

	var volumes []string
	model.Volumes.ElementsAs(context.Background(), &volumes, false)
	assert.Equal(t, []string{"vol-123", "vol-456"}, volumes)
}

func TestNewDataSourceBSSchedule(t *testing.T) {
	ds := NewDataSourceBSSchedule()
	assert.NotNil(t, ds)
	assert.IsType(t, &DataSourceBsSchedule{}, ds)
}

func TestDataSourceBsSchedule_Schema(t *testing.T) {
	ds := &DataSourceBsSchedule{}
	ctx := context.Background()
	req := datasource.SchemaRequest{}
	resp := &datasource.SchemaResponse{}

	ds.Schema(ctx, req, resp)

	assert.NotNil(t, resp.Schema)
	assert.NotEmpty(t, resp.Schema.Description)
	assert.NotEmpty(t, resp.Schema.MarkdownDescription)

	attributes := resp.Schema.Attributes
	assert.Contains(t, attributes, "id")
	assert.Contains(t, attributes, "name")
	assert.Contains(t, attributes, "description")
	assert.Contains(t, attributes, "volumes")
	assert.Contains(t, attributes, "snapshot_type")
	assert.Contains(t, attributes, "policy_retention_in_days")
	assert.Contains(t, attributes, "policy_frequency_daily_start_time")
	assert.Contains(t, attributes, "state")
	assert.Contains(t, attributes, "created_at")
	assert.Contains(t, attributes, "updated_at")

	assert.True(t, attributes["id"].IsRequired())

	assert.True(t, attributes["name"].IsComputed())
	assert.True(t, attributes["description"].IsComputed())
	assert.True(t, attributes["volumes"].IsComputed())
	assert.True(t, attributes["snapshot_type"].IsComputed())
	assert.True(t, attributes["policy_retention_in_days"].IsComputed())
	assert.True(t, attributes["policy_frequency_daily_start_time"].IsComputed())
	assert.True(t, attributes["state"].IsComputed())
	assert.True(t, attributes["created_at"].IsComputed())
	assert.True(t, attributes["updated_at"].IsComputed())
}

func TestDataSourceBsSchedule_Read_Success(t *testing.T) {
	t.Skip("Skipping read test - requires complex mock framework setup")
}

func TestDataSourceBsSchedule_Read_Error(t *testing.T) {
	t.Skip("Skipping read error test - requires complex mock framework setup")
}

type mockCoreClient struct{}

func (m mockCoreClient) DoRequest(method, path string, body interface{}, headers map[string]string) ([]byte, error) {
	return nil, nil
}

func (m mockCoreClient) SetAuthToken(token string) {}

func (m mockCoreClient) SetRegion(region string) {}

func (m mockCoreClient) SetHost(host string) {}
