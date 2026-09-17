package virtualmachines

import (
	"context"
	"testing"

	clientSDK "github.com/MagaluCloud/mgc-sdk-go/client"
	computeSdk "github.com/MagaluCloud/mgc-sdk-go/compute"
	"github.com/MagaluCloud/terraform-provider-mgc/mgc/internal/project"
	"github.com/MagaluCloud/terraform-provider-mgc/mgc/internal/testutils"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/stretchr/testify/assert"
)

const vmTestProjectID = "550e8400-e29b-41d4-a716-446655440000"

type recordingInstanceService struct {
	computeSdk.InstanceService
	gotCallOpts clientSDK.CallOptions
}

func (m *recordingInstanceService) Get(_ context.Context, callOpts clientSDK.CallOptions, id string, _ []computeSdk.InstanceExpand) (*computeSdk.Instance, error) {
	m.gotCallOpts = callOpts
	return &computeSdk.Instance{
		ID:          id,
		MachineType: &computeSdk.InstanceTypes{},
		Image:       &computeSdk.VmImage{},
		Network:     &computeSdk.Network{},
	}, nil
}

func TestVirtualMachineResourcesDeclareProject(t *testing.T) {
	ctx := context.Background()

	testCases := []struct {
		name  string
		r     resource.Resource
		model any
	}{
		{"instances", &vmInstances{}, &vmInstancesResourceModel{}},
		{"snapshots", &vmSnapshots{}, &vmSnapshotsResourceModel{}},
		{"interface_attach", &vmInterfaceAttach{}, &vmInterfaceAttachResourceModel{}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			s := testutils.GetResourceTestSchema(t, tc.r)

			state := tfsdk.State{Schema: s.Schema, Raw: testutils.ObjectWith(ctx, s.Schema.Type(), map[string]tftypes.Value{
				"project_id": tftypes.NewValue(tftypes.String, vmTestProjectID),
			})}
			assert.False(t, state.Get(ctx, tc.model).HasError(), "model must match the schema")
		})
	}
}

func TestVirtualMachineDataSourcesDeclareProject(t *testing.T) {
	ctx := context.Background()

	testCases := []struct {
		name  string
		ds    datasource.DataSource
		model any
	}{
		{"instance", &DataSourceVmInstance{}, &VMInstanceModel{}},
		{"instances", &DataSourceVmInstances{}, &VMInstancesModel{}},
		{"snapshots", &DataSourceVmSnapshots{}, &vmSnapshotsListModel{}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			s := testutils.GetDataSourceTestSchema(t, tc.ds)

			config := tfsdk.Config{Schema: s.Schema, Raw: testutils.ObjectWith(ctx, s.Schema.Type(), map[string]tftypes.Value{
				"project_id": tftypes.NewValue(tftypes.String, vmTestProjectID),
			})}
			assert.False(t, config.Get(ctx, tc.model).HasError(), "model must match the schema")
		})
	}
}

func TestVmInstancesReadKeepsProject(t *testing.T) {
	ctx := context.Background()

	for _, projectID := range []string{vmTestProjectID, ""} {
		expected := types.StringNull()
		if projectID != "" {
			expected = types.StringValue(projectID)
		}

		t.Run(projectID, func(t *testing.T) {
			mock := &recordingInstanceService{}
			r := &vmInstances{vmInstances: mock}
			s := testutils.GetResourceTestSchema(t, r)
			current := tfsdk.State{Schema: s.Schema, Raw: testutils.ObjectWith(ctx, s.Schema.Type(), map[string]tftypes.Value{
				"id":         tftypes.NewValue(tftypes.String, "inst-1"),
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

func TestVmInstancesImportStateWithProject(t *testing.T) {
	ctx := context.Background()
	r := &vmInstances{}

	s := testutils.GetResourceTestSchema(t, r)
	i := testutils.GetResourceTestIdentitySchema(t, r)

	resp := &resource.ImportStateResponse{
		State:    tfsdk.State{Schema: s.Schema, Raw: tftypes.NewValue(s.Schema.Type().TerraformType(ctx), nil)},
		Identity: &tfsdk.ResourceIdentity{Schema: i.IdentitySchema, Raw: tftypes.NewValue(i.IdentitySchema.Type().TerraformType(ctx), nil)},
	}
	r.ImportState(ctx, resource.ImportStateRequest{ID: "project/" + vmTestProjectID + "/inst-1"}, resp)
	assert.False(t, resp.Diagnostics.HasError(), "%v", resp.Diagnostics)

	var id, projectID types.String
	assert.False(t, resp.State.GetAttribute(ctx, path.Root("id"), &id).HasError())
	assert.False(t, resp.State.GetAttribute(ctx, path.Root("project_id"), &projectID).HasError())
	assert.Equal(t, types.StringValue("inst-1"), id)
	assert.Equal(t, types.StringValue(vmTestProjectID), projectID)

	var identity project.IdentityModel
	assert.False(t, resp.Identity.Get(ctx, &identity).HasError())
	assert.Equal(t, types.StringValue(vmTestProjectID), identity.ProjectID)
}
