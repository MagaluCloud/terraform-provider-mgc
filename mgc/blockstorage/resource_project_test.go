package blockstorage

import (
	"context"
	"testing"

	storageSDK "github.com/MagaluCloud/mgc-sdk-go/blockstorage"
	clientSDK "github.com/MagaluCloud/mgc-sdk-go/client"
	"github.com/MagaluCloud/terraform-provider-mgc/mgc/internal/project"
	"github.com/MagaluCloud/terraform-provider-mgc/mgc/internal/testutils"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/stretchr/testify/assert"
)

const bsTestProjectID = "550e8400-e29b-41d4-a716-446655440000"

type recordingVolumeService struct {
	storageSDK.VolumeService
	gotCallOpts clientSDK.CallOptions
}

func (m *recordingVolumeService) Get(_ context.Context, callOpts clientSDK.CallOptions, id string, _ []storageSDK.SnapshotExpand) (*storageSDK.Volume, error) {
	m.gotCallOpts = callOpts
	return &storageSDK.Volume{ID: id, Name: "v", Size: 10}, nil
}

type recordingSchedulerService struct {
	storageSDK.SchedulerService
	gotCallOpts clientSDK.CallOptions
}

func (m *recordingSchedulerService) Get(_ context.Context, callOpts clientSDK.CallOptions, id string, _ []storageSDK.ExpandSchedulers) (*storageSDK.SchedulerResponse, error) {
	m.gotCallOpts = callOpts
	return &storageSDK.SchedulerResponse{ID: id, Name: "s", Snapshot: &storageSDK.SnapshotConfig{Type: "instant"}}, nil
}

func TestBlockStorageResourcesDeclareProject(t *testing.T) {
	ctx := context.Background()

	testCases := []struct {
		name  string
		r     resource.Resource
		model any
	}{
		{"volumes", &bsVolumes{}, &bsVolumesResourceModel{}},
		{"snapshots", &bsSnapshots{}, &bsSnapshotsResourceModel{}},
		{"schedule", &bsSchedule{}, &bsScheduleResourceModel{}},
		{"schedule_attach", &bsScheduleAttach{}, &bsScheduleAttachResourceModel{}},
		{"volume_attachment", &VolumeAttach{}, &VolumeAttachResourceModel{}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			s := testutils.GetResourceTestSchema(t, tc.r).Schema

			state := tfsdk.State{Schema: s, Raw: testutils.ObjectWith(ctx, s.Type(), map[string]tftypes.Value{
				"project_id": tftypes.NewValue(tftypes.String, bsTestProjectID),
			})}
			assert.False(t, state.Get(ctx, tc.model).HasError(), "model must match the schema")
		})
	}
}

func TestBlockStorageReadKeepsProject(t *testing.T) {
	ctx := context.Background()

	for _, projectID := range []string{bsTestProjectID, ""} {
		expected := types.StringNull()
		if projectID != "" {
			expected = types.StringValue(projectID)
		}

		t.Run("volumes/"+projectID, func(t *testing.T) {
			mock := &recordingVolumeService{}
			r := &bsVolumes{bsVolumes: mock}
			s := testutils.GetResourceTestSchema(t, r).Schema
			current := tfsdk.State{Schema: s, Raw: testutils.ObjectWith(ctx, s.Type(), map[string]tftypes.Value{
				"id":         tftypes.NewValue(tftypes.String, "vol-1"),
				"project_id": testutils.OptionalString(projectID),
			})}

			resp := &resource.ReadResponse{State: current}
			r.Read(ctx, resource.ReadRequest{State: current}, resp)
			assert.False(t, resp.Diagnostics.HasError(), "%v", resp.Diagnostics)

			assert.Equal(t, clientSDK.CallOptions{ProjectID: projectID}, mock.gotCallOpts)
			var got types.String
			assert.False(t, resp.State.GetAttribute(ctx, path.Root("project_id"), &got).HasError())
			assert.Equal(t, expected, got)
		})

		t.Run("schedule/"+projectID, func(t *testing.T) {
			mock := &recordingSchedulerService{}
			r := &bsSchedule{bsScheduler: mock}
			s := testutils.GetResourceTestSchema(t, r).Schema
			current := tfsdk.State{Schema: s, Raw: testutils.ObjectWith(ctx, s.Type(), map[string]tftypes.Value{
				"id":         tftypes.NewValue(tftypes.String, "sch-1"),
				"project_id": testutils.OptionalString(projectID),
			})}

			resp := &resource.ReadResponse{State: current}
			r.Read(ctx, resource.ReadRequest{State: current}, resp)
			assert.False(t, resp.Diagnostics.HasError(), "%v", resp.Diagnostics)

			assert.Equal(t, clientSDK.CallOptions{ProjectID: projectID}, mock.gotCallOpts)
			var got types.String
			assert.False(t, resp.State.GetAttribute(ctx, path.Root("project_id"), &got).HasError())
			assert.Equal(t, expected, got)
		})
	}
}

func TestBlockStorageImportStateWithProject(t *testing.T) {
	ctx := context.Background()

	importResp := func(t *testing.T, r resource.ResourceWithIdentity) *resource.ImportStateResponse {
		s := testutils.GetResourceTestSchema(t, r).Schema
		i := testutils.GetResourceTestIdentitySchema(t, r).IdentitySchema
		return &resource.ImportStateResponse{
			State:    tfsdk.State{Schema: s, Raw: tftypes.NewValue(s.Type().TerraformType(ctx), nil)},
			Identity: &tfsdk.ResourceIdentity{Schema: i, Raw: tftypes.NewValue(i.Type().TerraformType(ctx), nil)},
		}
	}
	attribute := func(resp *resource.ImportStateResponse, name string) types.String {
		var v types.String
		assert.False(t, resp.State.GetAttribute(ctx, path.Root(name), &v).HasError())
		return v
	}

	for _, tc := range []struct {
		name string
		r    interface {
			resource.ResourceWithIdentity
			resource.ResourceWithImportState
		}
	}{
		{"volumes", &bsVolumes{}},
		{"schedule", &bsSchedule{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			resp := importResp(t, tc.r)
			tc.r.ImportState(ctx, resource.ImportStateRequest{ID: "project/" + bsTestProjectID + "/abc-1"}, resp)
			assert.False(t, resp.Diagnostics.HasError(), "%v", resp.Diagnostics)

			assert.Equal(t, types.StringValue("abc-1"), attribute(resp, "id"))
			assert.Equal(t, types.StringValue(bsTestProjectID), attribute(resp, "project_id"))

			var identity project.IdentityModel
			assert.False(t, resp.Identity.Get(ctx, &identity).HasError())
			assert.Equal(t, types.StringValue(bsTestProjectID), identity.ProjectID)
		})
	}

	t.Run("schedule_attach", func(t *testing.T) {
		r := &bsScheduleAttach{}

		resp := importResp(t, r)
		r.ImportState(ctx, resource.ImportStateRequest{ID: "project/" + bsTestProjectID + "/sch-1,vol-1"}, resp)
		assert.False(t, resp.Diagnostics.HasError(), "%v", resp.Diagnostics)
		assert.Equal(t, types.StringValue("sch-1"), attribute(resp, "schedule_id"))
		assert.Equal(t, types.StringValue("vol-1"), attribute(resp, "volume_id"))
		assert.Equal(t, types.StringValue(bsTestProjectID), attribute(resp, "project_id"))

		var identity scheduleAttachIdentityModel
		assert.False(t, resp.Identity.Get(ctx, &identity).HasError())
		assert.Equal(t, types.StringValue(bsTestProjectID), identity.ProjectID)

		resp = importResp(t, r)
		r.ImportState(ctx, resource.ImportStateRequest{Identity: &tfsdk.ResourceIdentity{
			Schema: resp.Identity.Schema,
			Raw: tftypes.NewValue(resp.Identity.Schema.Type().TerraformType(ctx), map[string]tftypes.Value{
				"schedule_id": tftypes.NewValue(tftypes.String, "sch-1"),
				"volume_id":   tftypes.NewValue(tftypes.String, "vol-1"),
				"project_id":  tftypes.NewValue(tftypes.String, nil),
			}),
		}}, resp)
		assert.False(t, resp.Diagnostics.HasError(), "%v", resp.Diagnostics)
		assert.Equal(t, types.StringValue("vol-1"), attribute(resp, "volume_id"))
		assert.Equal(t, types.StringNull(), attribute(resp, "project_id"))

		resp = importResp(t, r)
		r.ImportState(ctx, resource.ImportStateRequest{ID: "project/" + bsTestProjectID + "/sch-1"}, resp)
		assert.True(t, resp.Diagnostics.HasError())
	})
}
