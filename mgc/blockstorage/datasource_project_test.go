package blockstorage

import (
	"context"
	"testing"

	"github.com/MagaluCloud/terraform-provider-mgc/mgc/internal/testutils"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/stretchr/testify/assert"
)

func TestBlockStorageDataSourcesDeclareProject(t *testing.T) {
	ctx := context.Background()
	const projectID = "550e8400-e29b-41d4-a716-446655440000"

	testCases := []struct {
		name  string
		ds    datasource.DataSource
		model any
	}{
		{"volume", &DataSourceBsVolume{}, &bsVolumeResourceModel{}},
		{"volumes", &DataSourceBsVolumes{}, &bsVolumesDataSourceModel{}},
		{"snapshot", &DataSourceBsSnapshotDatasource{}, &bsSnapshotDataSourceModel{}},
		{"snapshots", &DataSourceBsSnapshots{}, &bsSnapshotsListDataSourceModel{}},
		{"schedule", &DataSourceBsSchedule{}, &bsScheduleDataSourceReadModel{}},
		{"schedules", &DataSourceBsSchedules{}, &bsSchedulesDataSourceModel{}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			s := testutils.GetDataSourceTestSchema(t, tc.ds)

			config := tfsdk.Config{Schema: s.Schema, Raw: testutils.ObjectWith(ctx, s.Schema.Type(), map[string]tftypes.Value{
				"project_id": tftypes.NewValue(tftypes.String, projectID),
			})}

			assert.False(t, config.Get(ctx, tc.model).HasError(), "model must match the schema")

			var got types.String
			assert.False(t, config.GetAttribute(ctx, path.Root("project_id"), &got).HasError())
			assert.Equal(t, types.StringValue(projectID), got)
		})
	}
}

func TestBlockStorageListItemsStayUnscoped(t *testing.T) {
	_, onSchedule := GetBsScheduleAttributes(false)["project_id"]
	_, onSnapshot := GetBsSnapshotAttributes(false)["project_id"]

	assert.False(t, onSchedule, "project_id belongs to the data source, not to each schedule item")
	assert.False(t, onSnapshot, "project_id belongs to the data source, not to each snapshot item")
}
